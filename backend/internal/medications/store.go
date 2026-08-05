// Package medications owns prescriptions and their generated dose logs.
//
// Medication name and dosage are not encrypted. That is a deliberate reading
// of spec §4, which marks only diagnosis, allergies, and discharge summary as
// encrypted columns — the scheduler must query medications in bulk to generate
// doses, which encryption would make impossible without decrypting every row
// on every tick. The narrative fields that reveal a condition stay encrypted.
package medications

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/OderoCeasar/afyalink/backend/internal/db"
)

// LogStatus mirrors the medication_log_status enum.
type LogStatus string

const (
	LogPending LogStatus = "pending"
	LogTaken   LogStatus = "taken"
	LogMissed  LogStatus = "missed"
)

// Medication is a prescription row.
type Medication struct {
	ID             uuid.UUID
	PatientID      uuid.UUID
	Name           string
	Dosage         string
	FrequencyHours int
	StartDate      time.Time
	EndDate        *time.Time
	CreatedAt      time.Time
}

// Log is a single scheduled dose.
type Log struct {
	ID           uuid.UUID
	MedicationID uuid.UUID
	ScheduledAt  time.Time
	TakenAt      *time.Time
	Status       LogStatus
}

// Store holds medication queries.
type Store struct {
	pool *db.Pool
}

// NewStore builds the medication store.
func NewStore(pool *db.Pool) *Store { return &Store{pool: pool} }

// CreateParams is the validated input for a new prescription.
type CreateParams struct {
	PatientID      uuid.UUID
	Name           string
	Dosage         string
	FrequencyHours int
	StartDate      time.Time
	EndDate        *time.Time
}

// Create inserts a prescription.
func (s *Store) Create(ctx context.Context, p CreateParams) (Medication, error) {
	const query = `
		INSERT INTO medications (patient_id, name, dosage, frequency_hours, start_date, end_date)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, patient_id, name, dosage, frequency_hours, start_date, end_date, created_at`

	var m Medication
	err := s.pool.QueryRow(ctx, query, p.PatientID, p.Name, p.Dosage, p.FrequencyHours, p.StartDate, p.EndDate).
		Scan(&m.ID, &m.PatientID, &m.Name, &m.Dosage, &m.FrequencyHours, &m.StartDate, &m.EndDate, &m.CreatedAt)
	if err != nil {
		return Medication{}, fmt.Errorf("medications: inserting medication: %w", err)
	}
	return m, nil
}

// ListForPatient returns a patient's prescriptions.
func (s *Store) ListForPatient(ctx context.Context, patientID uuid.UUID) ([]Medication, error) {
	const query = `
		SELECT id, patient_id, name, dosage, frequency_hours, start_date, end_date, created_at
		FROM medications
		WHERE patient_id = $1
		ORDER BY created_at DESC`

	rows, err := s.pool.Query(ctx, query, patientID)
	if err != nil {
		return nil, fmt.Errorf("medications: listing medications: %w", err)
	}
	defer rows.Close()

	var meds []Medication
	for rows.Next() {
		var m Medication
		if err := rows.Scan(&m.ID, &m.PatientID, &m.Name, &m.Dosage, &m.FrequencyHours, &m.StartDate, &m.EndDate, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("medications: scanning medication: %w", err)
		}
		meds = append(meds, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("medications: iterating medications: %w", err)
	}
	return meds, nil
}

// ListLogsForPatientWindow returns every dose for a patient scheduled inside
// the given window, so the caller can render a schedule without issuing one
// query per medication.
func (s *Store) ListLogsForPatientWindow(ctx context.Context, patientID uuid.UUID, from, to time.Time) ([]Log, error) {
	const query = `
		SELECT l.id, l.medication_id, l.scheduled_at, l.taken_at, l.status
		FROM medication_logs l
		JOIN medications m ON m.id = l.medication_id
		WHERE m.patient_id = $1
		  AND l.scheduled_at >= $2
		  AND l.scheduled_at < $3
		ORDER BY l.scheduled_at`

	rows, err := s.pool.Query(ctx, query, patientID, from, to)
	if err != nil {
		return nil, fmt.Errorf("medications: listing dose logs: %w", err)
	}
	defer rows.Close()

	var logs []Log
	for rows.Next() {
		var l Log
		if err := rows.Scan(&l.ID, &l.MedicationID, &l.ScheduledAt, &l.TakenAt, &l.Status); err != nil {
			return nil, fmt.Errorf("medications: scanning dose log: %w", err)
		}
		logs = append(logs, l)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("medications: iterating dose logs: %w", err)
	}
	return logs, nil
}

// LogOwner identifies which patient a dose log ultimately belongs to, so the
// "mark as taken" endpoint can run an ownership check on a nested resource.
type LogOwner struct {
	LogID        uuid.UUID
	MedicationID uuid.UUID
	PatientID    uuid.UUID
	Status       LogStatus
}

// GetLogOwner resolves a dose log to its patient.
//
// The medication id from the URL is matched as well as the log id: without
// that join condition, a caller could pass a medication they own together with
// someone else's log id and have the ownership check pass against the wrong
// row.
func (s *Store) GetLogOwner(ctx context.Context, medicationID, logID uuid.UUID) (LogOwner, error) {
	const query = `
		SELECT l.id, l.medication_id, m.patient_id, l.status
		FROM medication_logs l
		JOIN medications m ON m.id = l.medication_id
		WHERE l.id = $1 AND l.medication_id = $2`

	var owner LogOwner
	err := s.pool.QueryRow(ctx, query, logID, medicationID).
		Scan(&owner.LogID, &owner.MedicationID, &owner.PatientID, &owner.Status)
	if errors.Is(err, pgx.ErrNoRows) {
		return LogOwner{}, db.ErrNotFound
	}
	if err != nil {
		return LogOwner{}, fmt.Errorf("medications: resolving dose log owner: %w", err)
	}
	return owner, nil
}

// MarkTaken records a dose as taken.
//
// The `status <> 'taken'` guard makes the call idempotent-ish under a double
// tap: the second call affects zero rows and the caller is told the dose was
// already recorded, instead of the taken_at timestamp silently moving.
func (s *Store) MarkTaken(ctx context.Context, logID uuid.UUID, takenAt time.Time) (Log, error) {
	const query = `
		UPDATE medication_logs
		SET status = 'taken', taken_at = $2
		WHERE id = $1 AND status <> 'taken'
		RETURNING id, medication_id, scheduled_at, taken_at, status`

	var l Log
	err := s.pool.QueryRow(ctx, query, logID, takenAt).
		Scan(&l.ID, &l.MedicationID, &l.ScheduledAt, &l.TakenAt, &l.Status)
	if errors.Is(err, pgx.ErrNoRows) {
		return Log{}, db.ErrNotFound
	}
	if err != nil {
		return Log{}, fmt.Errorf("medications: marking dose taken: %w", err)
	}
	return l, nil
}
