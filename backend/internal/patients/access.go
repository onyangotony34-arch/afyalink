// Package patients owns patient records, discharge records, clinical notes,
// and — critically — the row-level ownership check every other patient-scoped
// package depends on.
package patients

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/OderoCeasar/afyalink/backend/internal/auth"
	"github.com/OderoCeasar/afyalink/backend/internal/db"
)

// ErrForbidden is returned for every denied patient access.
//
// It is deliberately returned for a nonexistent patient as well as for one the
// caller does not own. Handlers translate it into a single opaque 403. If the
// two cases produced different responses, a caregiver could walk the UUID
// space and learn which patient ids are real — the exact leak spec §5.3
// forbids.
var ErrForbidden = errors.New("caller may not access this patient")

// Ref is the minimal patient row needed to make an access decision. It carries
// no encrypted fields, so an authorisation check never decrypts PHI it may
// turn out the caller is not allowed to see.
type Ref struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	CaregiverID *uuid.UUID
	ClinicianID uuid.UUID
}

// Access performs row-level ownership checks.
type Access struct {
	pool *db.Pool
}

// NewAccess builds the ownership checker.
func NewAccess(pool *db.Pool) *Access { return &Access{pool: pool} }

// Authorize resolves a patient and confirms the principal may reach it.
//
// The rule per role:
//
//	clinician — must be the patient's assigned clinician
//	caregiver — must be the patient's linked caregiver
//	patient   — must be the patient themself
//
// Note what is absent: there is no "any clinician can see any patient" branch.
// A clinician who is not this patient's clinician is refused exactly like a
// stranger, which is what makes the negative tests in §9 meaningful.
func (a *Access) Authorize(ctx context.Context, principal auth.Principal, patientID uuid.UUID) (Ref, error) {
	ref, err := a.get(ctx, patientID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return Ref{}, ErrForbidden
		}
		return Ref{}, err
	}

	switch principal.Role {
	case auth.RoleClinician:
		if ref.ClinicianID == principal.UserID {
			return ref, nil
		}
	case auth.RoleCaregiver:
		if ref.CaregiverID != nil && *ref.CaregiverID == principal.UserID {
			return ref, nil
		}
	case auth.RolePatient:
		if ref.UserID == principal.UserID {
			return ref, nil
		}
	}

	return Ref{}, ErrForbidden
}

// get loads a patient reference by id.
func (a *Access) get(ctx context.Context, patientID uuid.UUID) (Ref, error) {
	const query = `
		SELECT id, user_id, caregiver_id, clinician_id
		FROM patients
		WHERE id = $1`

	var ref Ref
	err := a.pool.QueryRow(ctx, query, patientID).
		Scan(&ref.ID, &ref.UserID, &ref.CaregiverID, &ref.ClinicianID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Ref{}, db.ErrNotFound
	}
	if err != nil {
		return Ref{}, fmt.Errorf("patients: selecting patient ref: %w", err)
	}
	return ref, nil
}

// VisiblePatientIDs returns every patient id the principal may see.
//
// Used by the collection endpoints (notably GET /api/alerts) so the scoping is
// expressed once, in SQL, rather than by fetching everything and filtering in
// Go — which is the pattern that tends to leak a row when someone later adds
// a code path that forgets the filter.
func (a *Access) VisiblePatientIDs(ctx context.Context, principal auth.Principal) ([]uuid.UUID, error) {
	var query string
	switch principal.Role {
	case auth.RoleClinician:
		query = `SELECT id FROM patients WHERE clinician_id = $1`
	case auth.RoleCaregiver:
		query = `SELECT id FROM patients WHERE caregiver_id = $1`
	case auth.RolePatient:
		query = `SELECT id FROM patients WHERE user_id = $1`
	default:
		return nil, ErrForbidden
	}

	rows, err := a.pool.Query(ctx, query, principal.UserID)
	if err != nil {
		return nil, fmt.Errorf("patients: selecting visible patients: %w", err)
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("patients: scanning visible patient id: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("patients: iterating visible patients: %w", err)
	}
	return ids, nil
}
