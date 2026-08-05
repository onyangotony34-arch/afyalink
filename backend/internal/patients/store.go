package patients

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

// Record is a patient row with PHI still encrypted. Decryption happens in the
// service layer, only after authorisation has succeeded.
type Record struct {
	ID               uuid.UUID
	UserID           uuid.UUID
	FullName         string
	DOB              time.Time
	DiagnosisEnc     []byte
	AllergiesEnc     []byte
	EmergencyContact *string
	CaregiverID      *uuid.UUID
	ClinicianID      uuid.UUID
	CreatedAt        time.Time
}

// DischargeRecord is a discharge row with its summary still encrypted.
type DischargeRecord struct {
	ID            uuid.UUID
	PatientID     uuid.UUID
	ClinicianID   uuid.UUID
	SummaryEnc    []byte
	DischargeDate time.Time
	CreatedAt     time.Time
}

// ClinicalNote is a note row with its body still encrypted.
type ClinicalNote struct {
	ID          uuid.UUID
	PatientID   uuid.UUID
	ClinicianID uuid.UUID
	BodyEnc     []byte
	CreatedAt   time.Time
}

// Store holds patient-domain queries.
type Store struct {
	pool *db.Pool
}

// NewStore builds the patient store.
func NewStore(pool *db.Pool) *Store { return &Store{pool: pool} }

// CreateParams carries an already-encrypted patient for insertion. The service
// layer encrypts before calling, so no plaintext PHI ever reaches this file.
type CreateParams struct {
	UserID           uuid.UUID
	FullName         string
	DOB              time.Time
	DiagnosisEnc     []byte
	AllergiesEnc     []byte
	EmergencyContact *string
	CaregiverID      *uuid.UUID
	ClinicianID      uuid.UUID
}

// Create inserts a patient.
func (s *Store) Create(ctx context.Context, p CreateParams) (Record, error) {
	const query = `
		INSERT INTO patients (
			user_id, full_name, dob, diagnosis_enc, allergies_enc,
			emergency_contact, caregiver_id, clinician_id
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, user_id, full_name, dob, diagnosis_enc, allergies_enc,
		          emergency_contact, caregiver_id, clinician_id, created_at`

	var r Record
	err := s.pool.QueryRow(ctx, query,
		p.UserID, p.FullName, p.DOB, p.DiagnosisEnc, p.AllergiesEnc,
		p.EmergencyContact, p.CaregiverID, p.ClinicianID,
	).Scan(
		&r.ID, &r.UserID, &r.FullName, &r.DOB, &r.DiagnosisEnc, &r.AllergiesEnc,
		&r.EmergencyContact, &r.CaregiverID, &r.ClinicianID, &r.CreatedAt,
	)
	if err != nil {
		return Record{}, fmt.Errorf("patients: inserting patient: %w", err)
	}
	return r, nil
}

// GetByID loads a full patient row.
func (s *Store) GetByID(ctx context.Context, id uuid.UUID) (Record, error) {
	const query = `
		SELECT id, user_id, full_name, dob, diagnosis_enc, allergies_enc,
		       emergency_contact, caregiver_id, clinician_id, created_at
		FROM patients
		WHERE id = $1`

	var r Record
	err := s.pool.QueryRow(ctx, query, id).Scan(
		&r.ID, &r.UserID, &r.FullName, &r.DOB, &r.DiagnosisEnc, &r.AllergiesEnc,
		&r.EmergencyContact, &r.CaregiverID, &r.ClinicianID, &r.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Record{}, db.ErrNotFound
	}
	if err != nil {
		return Record{}, fmt.Errorf("patients: selecting patient: %w", err)
	}
	return r, nil
}

// ListForPrincipal returns the patients visible to the caller, scoped in SQL
// by role.
func (s *Store) ListForPrincipal(ctx context.Context, principal auth.Principal) ([]Record, error) {
	const base = `
		SELECT id, user_id, full_name, dob, diagnosis_enc, allergies_enc,
		       emergency_contact, caregiver_id, clinician_id, created_at
		FROM patients
		WHERE `

	var where string
	switch principal.Role {
	case auth.RoleClinician:
		where = "clinician_id = $1"
	case auth.RoleCaregiver:
		where = "caregiver_id = $1"
	case auth.RolePatient:
		where = "user_id = $1"
	default:
		return nil, ErrForbidden
	}

	// The WHERE fragment is a compile-time constant chosen by a closed switch,
	// never client input; the principal id is still a bind parameter.
	rows, err := s.pool.Query(ctx, base+where+" ORDER BY created_at DESC", principal.UserID)
	if err != nil {
		return nil, fmt.Errorf("patients: listing patients: %w", err)
	}
	defer rows.Close()

	var records []Record
	for rows.Next() {
		var r Record
		if err := rows.Scan(
			&r.ID, &r.UserID, &r.FullName, &r.DOB, &r.DiagnosisEnc, &r.AllergiesEnc,
			&r.EmergencyContact, &r.CaregiverID, &r.ClinicianID, &r.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("patients: scanning patient: %w", err)
		}
		records = append(records, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("patients: iterating patients: %w", err)
	}
	return records, nil
}

// UserRole returns a user's role, used to validate that a patient link points
// at an account of the expected kind.
func (s *Store) UserRole(ctx context.Context, userID uuid.UUID) (auth.Role, error) {
	const query = `SELECT role FROM users WHERE id = $1`

	var role auth.Role
	err := s.pool.QueryRow(ctx, query, userID).Scan(&role)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", db.ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("patients: selecting user role: %w", err)
	}
	return role, nil
}

// PatientExistsForUser reports whether a patient record already exists for a
// user account, so the one-patient-per-account rule can be reported as a
// validation error instead of a raw constraint violation.
func (s *Store) PatientExistsForUser(ctx context.Context, userID uuid.UUID) (bool, error) {
	const query = `SELECT EXISTS (SELECT 1 FROM patients WHERE user_id = $1)`

	var exists bool
	if err := s.pool.QueryRow(ctx, query, userID).Scan(&exists); err != nil {
		return false, fmt.Errorf("patients: checking existing patient: %w", err)
	}
	return exists, nil
}

// CreateDischarge inserts a discharge record.
func (s *Store) CreateDischarge(ctx context.Context, patientID, clinicianID uuid.UUID, summaryEnc []byte, dischargeDate time.Time) (DischargeRecord, error) {
	const query = `
		INSERT INTO discharge_records (patient_id, clinician_id, summary_enc, discharge_date)
		VALUES ($1, $2, $3, $4)
		RETURNING id, patient_id, clinician_id, summary_enc, discharge_date, created_at`

	var d DischargeRecord
	err := s.pool.QueryRow(ctx, query, patientID, clinicianID, summaryEnc, dischargeDate).
		Scan(&d.ID, &d.PatientID, &d.ClinicianID, &d.SummaryEnc, &d.DischargeDate, &d.CreatedAt)
	if err != nil {
		return DischargeRecord{}, fmt.Errorf("patients: inserting discharge record: %w", err)
	}
	return d, nil
}

// ListDischarges returns a patient's discharge records, newest first.
func (s *Store) ListDischarges(ctx context.Context, patientID uuid.UUID) ([]DischargeRecord, error) {
	const query = `
		SELECT id, patient_id, clinician_id, summary_enc, discharge_date, created_at
		FROM discharge_records
		WHERE patient_id = $1
		ORDER BY discharge_date DESC, created_at DESC`

	rows, err := s.pool.Query(ctx, query, patientID)
	if err != nil {
		return nil, fmt.Errorf("patients: listing discharge records: %w", err)
	}
	defer rows.Close()

	var records []DischargeRecord
	for rows.Next() {
		var d DischargeRecord
		if err := rows.Scan(&d.ID, &d.PatientID, &d.ClinicianID, &d.SummaryEnc, &d.DischargeDate, &d.CreatedAt); err != nil {
			return nil, fmt.Errorf("patients: scanning discharge record: %w", err)
		}
		records = append(records, d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("patients: iterating discharge records: %w", err)
	}
	return records, nil
}

// CreateNote inserts a clinical note.
func (s *Store) CreateNote(ctx context.Context, patientID, clinicianID uuid.UUID, bodyEnc []byte) (ClinicalNote, error) {
	const query = `
		INSERT INTO clinical_notes (patient_id, clinician_id, body_enc)
		VALUES ($1, $2, $3)
		RETURNING id, patient_id, clinician_id, body_enc, created_at`

	var n ClinicalNote
	err := s.pool.QueryRow(ctx, query, patientID, clinicianID, bodyEnc).
		Scan(&n.ID, &n.PatientID, &n.ClinicianID, &n.BodyEnc, &n.CreatedAt)
	if err != nil {
		return ClinicalNote{}, fmt.Errorf("patients: inserting clinical note: %w", err)
	}
	return n, nil
}

// ListNotes returns a patient's clinical notes, newest first.
func (s *Store) ListNotes(ctx context.Context, patientID uuid.UUID) ([]ClinicalNote, error) {
	const query = `
		SELECT id, patient_id, clinician_id, body_enc, created_at
		FROM clinical_notes
		WHERE patient_id = $1
		ORDER BY created_at DESC`

	rows, err := s.pool.Query(ctx, query, patientID)
	if err != nil {
		return nil, fmt.Errorf("patients: listing clinical notes: %w", err)
	}
	defer rows.Close()

	var notes []ClinicalNote
	for rows.Next() {
		var n ClinicalNote
		if err := rows.Scan(&n.ID, &n.PatientID, &n.ClinicianID, &n.BodyEnc, &n.CreatedAt); err != nil {
			return nil, fmt.Errorf("patients: scanning clinical note: %w", err)
		}
		notes = append(notes, n)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("patients: iterating clinical notes: %w", err)
	}
	return notes, nil
}
