// Package alerts owns clinician-facing alerts raised by the red-flag engine
// and by the scheduler's missed-dose sweep.
package alerts

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/OderoCeasar/afyalink/backend/internal/auth"
	"github.com/OderoCeasar/afyalink/backend/internal/db"
)

// Severity mirrors the alert_severity enum. The Postgres enum is declared
// low < medium < high < critical, so `ORDER BY severity DESC` sorts most
// severe first without a CASE expression.
type Severity string

const (
	SeverityLow      Severity = "low"
	SeverityMedium   Severity = "medium"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)

// Alert is one row of the alerts table, joined with the patient's name for
// display. full_name is not an encrypted column, so this join costs no
// decryption.
type Alert struct {
	ID              uuid.UUID
	PatientID       uuid.UUID
	PatientFullName string
	CheckinID       *uuid.UUID
	Severity        Severity
	Message         string
	ResolvedAt      *time.Time
	ResolvedBy      *uuid.UUID
	CreatedAt       time.Time
}

// Store holds alert queries.
type Store struct {
	pool *db.Pool
}

// NewStore builds the alert store.
func NewStore(pool *db.Pool) *Store { return &Store{pool: pool} }

// Querier is satisfied by both *pgxpool.Pool and pgx.Tx, so an alert can be
// raised inside the same transaction as the check-in response that caused it.
type Querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// Create inserts an alert using the pool.
func (s *Store) Create(ctx context.Context, patientID uuid.UUID, checkinID *uuid.UUID, severity Severity, message string) (uuid.UUID, error) {
	return CreateWith(ctx, s.pool, patientID, checkinID, severity, message)
}

// CreateWith inserts an alert using any querier, including a transaction.
func CreateWith(ctx context.Context, q Querier, patientID uuid.UUID, checkinID *uuid.UUID, severity Severity, message string) (uuid.UUID, error) {
	const query = `
		INSERT INTO alerts (patient_id, checkin_id, severity, message)
		VALUES ($1, $2, $3, $4)
		RETURNING id`

	var id uuid.UUID
	if err := q.QueryRow(ctx, query, patientID, checkinID, string(severity), message).Scan(&id); err != nil {
		return uuid.UUID{}, fmt.Errorf("alerts: inserting alert: %w", err)
	}
	return id, nil
}

// ListForPrincipal returns alerts for every patient the caller may see,
// unresolved first and most severe first within each group.
//
// The role scoping lives in the WHERE clause rather than in a Go filter after
// the fact: a query that can only ever return the caller's rows cannot leak
// one through a later refactor that forgets to filter.
func (s *Store) ListForPrincipal(ctx context.Context, principal auth.Principal, includeResolved bool) ([]Alert, error) {
	var scope string
	switch principal.Role {
	case auth.RoleClinician:
		scope = "p.clinician_id = $1"
	case auth.RoleCaregiver:
		scope = "p.caregiver_id = $1"
	case auth.RolePatient:
		scope = "p.user_id = $1"
	default:
		return nil, fmt.Errorf("alerts: unknown role %q", principal.Role)
	}

	resolvedFilter := ""
	if !includeResolved {
		resolvedFilter = " AND a.resolved_at IS NULL"
	}

	// Both fragments are compile-time constants selected by closed switches
	// and a boolean; the only client-derived value is bound as $1.
	query := `
		SELECT a.id, a.patient_id, p.full_name, a.checkin_id, a.severity,
		       a.message, a.resolved_at, a.resolved_by, a.created_at
		FROM alerts a
		JOIN patients p ON p.id = a.patient_id
		WHERE ` + scope + resolvedFilter + `
		ORDER BY (a.resolved_at IS NULL) DESC, a.severity DESC, a.created_at DESC`

	rows, err := s.pool.Query(ctx, query, principal.UserID)
	if err != nil {
		return nil, fmt.Errorf("alerts: listing alerts: %w", err)
	}
	defer rows.Close()

	var list []Alert
	for rows.Next() {
		var a Alert
		if err := rows.Scan(
			&a.ID, &a.PatientID, &a.PatientFullName, &a.CheckinID, &a.Severity,
			&a.Message, &a.ResolvedAt, &a.ResolvedBy, &a.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("alerts: scanning alert: %w", err)
		}
		list = append(list, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("alerts: iterating alerts: %w", err)
	}
	return list, nil
}

// GetPatientID resolves which patient an alert belongs to, so the resolve
// endpoint can run an ownership check before touching the row.
func (s *Store) GetPatientID(ctx context.Context, alertID uuid.UUID) (uuid.UUID, error) {
	const query = `SELECT patient_id FROM alerts WHERE id = $1`

	var patientID uuid.UUID
	err := s.pool.QueryRow(ctx, query, alertID).Scan(&patientID)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.UUID{}, db.ErrNotFound
	}
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("alerts: resolving alert patient: %w", err)
	}
	return patientID, nil
}

// Resolve marks an alert resolved.
//
// The `resolved_at IS NULL` guard means a second resolve affects no rows and
// is reported as a conflict, rather than silently overwriting who resolved it
// and when — that history matters in a clinical audit.
func (s *Store) Resolve(ctx context.Context, alertID, resolvedBy uuid.UUID) (Alert, error) {
	const query = `
		UPDATE alerts a
		SET resolved_at = now(), resolved_by = $2
		FROM patients p
		WHERE a.id = $1 AND a.resolved_at IS NULL AND p.id = a.patient_id
		RETURNING a.id, a.patient_id, p.full_name, a.checkin_id, a.severity,
		          a.message, a.resolved_at, a.resolved_by, a.created_at`

	var a Alert
	err := s.pool.QueryRow(ctx, query, alertID, resolvedBy).Scan(
		&a.ID, &a.PatientID, &a.PatientFullName, &a.CheckinID, &a.Severity,
		&a.Message, &a.ResolvedAt, &a.ResolvedBy, &a.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Alert{}, db.ErrNotFound
	}
	if err != nil {
		return Alert{}, fmt.Errorf("alerts: resolving alert: %w", err)
	}
	return a, nil
}
