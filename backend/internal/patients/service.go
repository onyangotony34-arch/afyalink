package patients

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/OderoCeasar/afyalink/backend/internal/auth"
	"github.com/OderoCeasar/afyalink/backend/internal/crypto"
	"github.com/OderoCeasar/afyalink/backend/internal/db"
)

// ErrInvalidLink is returned when a patient is linked to a user account of the
// wrong role, or to an account that already has a patient record.
var ErrInvalidLink = errors.New("invalid user link")

// Service applies encryption and access policy over the patient store.
type Service struct {
	store  *Store
	access *Access
	cipher *crypto.Cipher
}

// NewService wires the patient service.
func NewService(store *Store, access *Access, cipher *crypto.Cipher) *Service {
	return &Service{store: store, access: access, cipher: cipher}
}

// Patient is the decrypted, client-facing view of a patient.
//
// Diagnosis and Allergies are pointers because whether they are populated
// depends on the caller's role — see decryptFor. A nil field means "not
// visible to you", which is rendered as an absent JSON key rather than an
// empty string, so the frontend never shows a blank "Diagnosis:" label.
type Patient struct {
	ID               uuid.UUID
	UserID           uuid.UUID
	FullName         string
	DOB              time.Time
	Diagnosis        *string
	Allergies        *string
	EmergencyContact *string
	CaregiverID      *uuid.UUID
	ClinicianID      uuid.UUID
	CreatedAt        time.Time
}

// Discharge is the decrypted view of a discharge record.
type Discharge struct {
	ID            uuid.UUID
	PatientID     uuid.UUID
	ClinicianID   uuid.UUID
	Summary       string
	DischargeDate time.Time
	CreatedAt     time.Time
}

// Note is the decrypted view of a clinical note.
type Note struct {
	ID          uuid.UUID
	PatientID   uuid.UUID
	ClinicianID uuid.UUID
	Body        string
	CreatedAt   time.Time
}

// CreateInput is the validated input for creating a patient.
type CreateInput struct {
	UserID           uuid.UUID
	FullName         string
	DOB              time.Time
	Diagnosis        string
	Allergies        *string
	EmergencyContact *string
	CaregiverID      *uuid.UUID
}

// Create registers a patient under the calling clinician.
//
// The clinician id is taken from the authenticated principal, never from the
// request body — otherwise a clinician could assign a patient to a colleague
// and quietly gain or shed responsibility for them.
func (s *Service) Create(ctx context.Context, clinician auth.Principal, in CreateInput) (Patient, error) {
	// Linked accounts must actually hold the role the link implies. Without
	// this, a clinician account could be attached as a patient's user_id, and
	// that clinician would then pass the "patient viewing self" ownership
	// branch on someone else's record.
	if err := s.requireRole(ctx, in.UserID, auth.RolePatient); err != nil {
		return Patient{}, fmt.Errorf("%w: user_id must reference a patient account", err)
	}
	if in.CaregiverID != nil {
		if err := s.requireRole(ctx, *in.CaregiverID, auth.RoleCaregiver); err != nil {
			return Patient{}, fmt.Errorf("%w: caregiver_id must reference a caregiver account", err)
		}
	}

	exists, err := s.store.PatientExistsForUser(ctx, in.UserID)
	if err != nil {
		return Patient{}, err
	}
	if exists {
		return Patient{}, fmt.Errorf("%w: this account already has a patient record", ErrInvalidLink)
	}

	diagnosisEnc, err := s.cipher.EncryptString(in.Diagnosis)
	if err != nil {
		return Patient{}, err
	}
	allergiesEnc, err := s.cipher.EncryptNullable(in.Allergies)
	if err != nil {
		return Patient{}, err
	}

	record, err := s.store.Create(ctx, CreateParams{
		UserID:           in.UserID,
		FullName:         in.FullName,
		DOB:              in.DOB,
		DiagnosisEnc:     diagnosisEnc,
		AllergiesEnc:     allergiesEnc,
		EmergencyContact: in.EmergencyContact,
		CaregiverID:      in.CaregiverID,
		ClinicianID:      clinician.UserID,
	})
	if err != nil {
		return Patient{}, err
	}

	return s.decryptFor(clinician, record)
}

// Get returns a patient after an ownership check.
func (s *Service) Get(ctx context.Context, principal auth.Principal, patientID uuid.UUID) (Patient, error) {
	if _, err := s.access.Authorize(ctx, principal, patientID); err != nil {
		return Patient{}, err
	}

	record, err := s.store.GetByID(ctx, patientID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return Patient{}, ErrForbidden
		}
		return Patient{}, err
	}

	return s.decryptFor(principal, record)
}

// List returns every patient visible to the caller.
func (s *Service) List(ctx context.Context, principal auth.Principal) ([]Patient, error) {
	records, err := s.store.ListForPrincipal(ctx, principal)
	if err != nil {
		return nil, err
	}

	result := make([]Patient, 0, len(records))
	for _, record := range records {
		p, err := s.decryptFor(principal, record)
		if err != nil {
			return nil, err
		}
		result = append(result, p)
	}
	return result, nil
}

// CreateDischarge writes a discharge record for a patient the clinician owns.
func (s *Service) CreateDischarge(ctx context.Context, principal auth.Principal, patientID uuid.UUID, summary string, dischargeDate time.Time) (Discharge, error) {
	if _, err := s.access.Authorize(ctx, principal, patientID); err != nil {
		return Discharge{}, err
	}

	summaryEnc, err := s.cipher.EncryptString(summary)
	if err != nil {
		return Discharge{}, err
	}

	record, err := s.store.CreateDischarge(ctx, patientID, principal.UserID, summaryEnc, dischargeDate)
	if err != nil {
		return Discharge{}, err
	}

	return Discharge{
		ID:            record.ID,
		PatientID:     record.PatientID,
		ClinicianID:   record.ClinicianID,
		Summary:       summary,
		DischargeDate: record.DischargeDate,
		CreatedAt:     record.CreatedAt,
	}, nil
}

// ListDischarges returns a patient's discharge records.
//
// The summary is a clinical narrative, so it is decrypted for the clinician
// and the patient but withheld from the caregiver, whose grant in spec §3
// covers check-ins, medications, and alerts. The dates still come back so a
// caregiver's timeline is complete.
func (s *Service) ListDischarges(ctx context.Context, principal auth.Principal, patientID uuid.UUID) ([]Discharge, error) {
	if _, err := s.access.Authorize(ctx, principal, patientID); err != nil {
		return nil, err
	}

	records, err := s.store.ListDischarges(ctx, patientID)
	if err != nil {
		return nil, err
	}

	result := make([]Discharge, 0, len(records))
	for _, record := range records {
		d := Discharge{
			ID:            record.ID,
			PatientID:     record.PatientID,
			ClinicianID:   record.ClinicianID,
			DischargeDate: record.DischargeDate,
			CreatedAt:     record.CreatedAt,
		}

		if principal.Role != auth.RoleCaregiver {
			summary, err := s.cipher.DecryptString(record.SummaryEnc)
			if err != nil {
				return nil, err
			}
			d.Summary = summary
		}
		result = append(result, d)
	}
	return result, nil
}

// CreateNote adds a clinical note. Route-level RBAC already restricts this to
// clinicians; the ownership check restricts it to their own patients.
func (s *Service) CreateNote(ctx context.Context, principal auth.Principal, patientID uuid.UUID, body string) (Note, error) {
	if _, err := s.access.Authorize(ctx, principal, patientID); err != nil {
		return Note{}, err
	}

	bodyEnc, err := s.cipher.EncryptString(body)
	if err != nil {
		return Note{}, err
	}

	record, err := s.store.CreateNote(ctx, patientID, principal.UserID, bodyEnc)
	if err != nil {
		return Note{}, err
	}

	return Note{
		ID:          record.ID,
		PatientID:   record.PatientID,
		ClinicianID: record.ClinicianID,
		Body:        body,
		CreatedAt:   record.CreatedAt,
	}, nil
}

// ListNotes returns a patient's clinical notes.
//
// Clinicians only. Spec §3 denies caregivers clinical notes outright, and this
// service refuses the role directly rather than relying solely on the route's
// RBAC — defence in depth against a future route being mounted carelessly.
func (s *Service) ListNotes(ctx context.Context, principal auth.Principal, patientID uuid.UUID) ([]Note, error) {
	if principal.Role != auth.RoleClinician {
		return nil, ErrForbidden
	}
	if _, err := s.access.Authorize(ctx, principal, patientID); err != nil {
		return nil, err
	}

	records, err := s.store.ListNotes(ctx, patientID)
	if err != nil {
		return nil, err
	}

	result := make([]Note, 0, len(records))
	for _, record := range records {
		body, err := s.cipher.DecryptString(record.BodyEnc)
		if err != nil {
			return nil, err
		}
		result = append(result, Note{
			ID:          record.ID,
			PatientID:   record.PatientID,
			ClinicianID: record.ClinicianID,
			Body:        body,
			CreatedAt:   record.CreatedAt,
		})
	}
	return result, nil
}

// decryptFor decrypts a patient's PHI according to the caller's role.
//
// Field-level policy:
//
//	clinician — diagnosis and allergies (they are treating this patient)
//	patient   — diagnosis and allergies (their own record)
//	caregiver — allergies only
//
// Allergies reach the caregiver because that is the field a caregiver acts on
// in an emergency; the diagnosis narrative is withheld under the same
// least-privilege reading of spec §3 that withholds clinical notes. This is a
// policy judgement, not a technical constraint — it is one branch to change if
// caregivers should see more.
func (s *Service) decryptFor(principal auth.Principal, record Record) (Patient, error) {
	p := Patient{
		ID:               record.ID,
		UserID:           record.UserID,
		FullName:         record.FullName,
		DOB:              record.DOB,
		EmergencyContact: record.EmergencyContact,
		CaregiverID:      record.CaregiverID,
		ClinicianID:      record.ClinicianID,
		CreatedAt:        record.CreatedAt,
	}

	if principal.Role == auth.RoleClinician || principal.Role == auth.RolePatient {
		diagnosis, err := s.cipher.DecryptString(record.DiagnosisEnc)
		if err != nil {
			return Patient{}, err
		}
		p.Diagnosis = &diagnosis
	}

	allergies, err := s.cipher.DecryptNullable(record.AllergiesEnc)
	if err != nil {
		return Patient{}, err
	}
	p.Allergies = allergies

	return p, nil
}

// requireRole confirms a linked user exists and holds the expected role.
func (s *Service) requireRole(ctx context.Context, userID uuid.UUID, want auth.Role) error {
	role, err := s.store.UserRole(ctx, userID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return ErrInvalidLink
		}
		return err
	}
	if role != want {
		return ErrInvalidLink
	}
	return nil
}
