package patients

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/OderoCeasar/afyalink/backend/internal/audit"
	"github.com/OderoCeasar/afyalink/backend/internal/auth"
	"github.com/OderoCeasar/afyalink/backend/internal/httpx"
	"github.com/OderoCeasar/afyalink/backend/pkg/validator"
)

// DateLayout is the wire format for date-only fields.
const DateLayout = "2006-01-02"

// Handler serves the patient, discharge, and clinical-note endpoints.
type Handler struct {
	service *Service
}

// NewHandler builds the patient handler.
func NewHandler(service *Service) *Handler { return &Handler{service: service} }

// PatientResponse is the client-facing patient shape. Fields the caller may
// not see are omitted entirely rather than nulled.
type PatientResponse struct {
	ID               string  `json:"id"`
	UserID           string  `json:"user_id"`
	FullName         string  `json:"full_name"`
	DOB              string  `json:"dob"`
	Diagnosis        *string `json:"diagnosis,omitempty"`
	Allergies        *string `json:"allergies,omitempty"`
	EmergencyContact *string `json:"emergency_contact,omitempty"`
	CaregiverID      *string `json:"caregiver_id,omitempty"`
	ClinicianID      string  `json:"clinician_id"`
	CreatedAt        string  `json:"created_at"`
}

func toPatientResponse(p Patient) PatientResponse {
	resp := PatientResponse{
		ID:               p.ID.String(),
		UserID:           p.UserID.String(),
		FullName:         p.FullName,
		DOB:              p.DOB.Format(DateLayout),
		Diagnosis:        p.Diagnosis,
		Allergies:        p.Allergies,
		EmergencyContact: p.EmergencyContact,
		ClinicianID:      p.ClinicianID.String(),
		CreatedAt:        p.CreatedAt.Format(time.RFC3339),
	}
	if p.CaregiverID != nil {
		id := p.CaregiverID.String()
		resp.CaregiverID = &id
	}
	return resp
}

// DischargeResponse is the client-facing discharge record shape.
type DischargeResponse struct {
	ID            string `json:"id"`
	PatientID     string `json:"patient_id"`
	ClinicianID   string `json:"clinician_id"`
	Summary       string `json:"summary,omitempty"`
	DischargeDate string `json:"discharge_date"`
	CreatedAt     string `json:"created_at"`
}

// NoteResponse is the client-facing clinical note shape.
type NoteResponse struct {
	ID          string `json:"id"`
	PatientID   string `json:"patient_id"`
	ClinicianID string `json:"clinician_id"`
	Body        string `json:"body"`
	CreatedAt   string `json:"created_at"`
}

type createPatientRequest struct {
	UserID           string  `json:"user_id" validate:"required,uuid"`
	FullName         string  `json:"full_name" validate:"required,min=2,max=200"`
	DOB              string  `json:"dob" validate:"required,datetime=2006-01-02"`
	Diagnosis        string  `json:"diagnosis" validate:"required,min=2,max=2000"`
	Allergies        *string `json:"allergies" validate:"omitempty,max=2000"`
	EmergencyContact *string `json:"emergency_contact" validate:"omitempty,max=200"`
	CaregiverID      *string `json:"caregiver_id" validate:"omitempty,uuid"`
}

// Create registers a patient under the calling clinician.
func (h *Handler) Create(c *gin.Context) {
	principal := auth.MustPrincipal(c)

	var req createPatientRequest
	if err := validator.BindJSON(c, &req); err != nil {
		httpx.BadRequest(c, err)
		return
	}

	// Validation above guarantees these parse; the errors are still checked
	// rather than discarded.
	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		httpx.BadRequest(c, &validator.ValidationError{
			Message: "request validation failed",
			Fields:  []validator.FieldError{{Field: "user_id", Message: "must be a valid UUID"}},
		})
		return
	}

	dob, err := time.Parse(DateLayout, req.DOB)
	if err != nil {
		httpx.BadRequest(c, &validator.ValidationError{
			Message: "request validation failed",
			Fields:  []validator.FieldError{{Field: "dob", Message: "must be a date in YYYY-MM-DD form"}},
		})
		return
	}
	if dob.After(time.Now()) {
		httpx.BadRequest(c, &validator.ValidationError{
			Message: "request validation failed",
			Fields:  []validator.FieldError{{Field: "dob", Message: "must not be in the future"}},
		})
		return
	}

	var caregiverID *uuid.UUID
	if req.CaregiverID != nil {
		parsed, err := uuid.Parse(*req.CaregiverID)
		if err != nil {
			httpx.BadRequest(c, &validator.ValidationError{
				Message: "request validation failed",
				Fields:  []validator.FieldError{{Field: "caregiver_id", Message: "must be a valid UUID"}},
			})
			return
		}
		caregiverID = &parsed
	}

	patient, err := h.service.Create(c.Request.Context(), principal, CreateInput{
		UserID:           userID,
		FullName:         req.FullName,
		DOB:              dob,
		Diagnosis:        req.Diagnosis,
		Allergies:        req.Allergies,
		EmergencyContact: req.EmergencyContact,
		CaregiverID:      caregiverID,
	})
	if err != nil {
		if errors.Is(err, ErrInvalidLink) {
			httpx.BadRequest(c, &validator.ValidationError{Message: err.Error()})
			return
		}
		httpx.Internal(c, err)
		return
	}

	audit.Record(c, audit.ActionCreate, audit.ResourcePatient, &patient.ID)
	c.JSON(http.StatusCreated, gin.H{"patient": toPatientResponse(patient)})
}

// Get returns one patient.
func (h *Handler) Get(c *gin.Context) {
	principal := auth.MustPrincipal(c)

	patientID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	patient, err := h.service.Get(c.Request.Context(), principal, patientID)
	if err != nil {
		respondPatientError(c, err)
		return
	}

	audit.Record(c, audit.ActionRead, audit.ResourcePatient, &patient.ID)
	c.JSON(http.StatusOK, gin.H{"patient": toPatientResponse(patient)})
}

// List returns every patient visible to the caller.
func (h *Handler) List(c *gin.Context) {
	principal := auth.MustPrincipal(c)

	list, err := h.service.List(c.Request.Context(), principal)
	if err != nil {
		respondPatientError(c, err)
		return
	}

	responses := make([]PatientResponse, 0, len(list))
	for _, p := range list {
		responses = append(responses, toPatientResponse(p))
	}

	audit.Record(c, audit.ActionList, audit.ResourcePatient, nil)
	c.JSON(http.StatusOK, gin.H{"patients": responses})
}

type createDischargeRequest struct {
	Summary       string `json:"summary" validate:"required,min=2,max=10000"`
	DischargeDate string `json:"discharge_date" validate:"required,datetime=2006-01-02"`
}

// CreateDischarge writes a discharge record.
func (h *Handler) CreateDischarge(c *gin.Context) {
	principal := auth.MustPrincipal(c)

	patientID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	var req createDischargeRequest
	if err := validator.BindJSON(c, &req); err != nil {
		httpx.BadRequest(c, err)
		return
	}

	dischargeDate, err := time.Parse(DateLayout, req.DischargeDate)
	if err != nil {
		httpx.BadRequest(c, &validator.ValidationError{
			Message: "request validation failed",
			Fields:  []validator.FieldError{{Field: "discharge_date", Message: "must be a date in YYYY-MM-DD form"}},
		})
		return
	}

	discharge, err := h.service.CreateDischarge(c.Request.Context(), principal, patientID, req.Summary, dischargeDate)
	if err != nil {
		respondPatientError(c, err)
		return
	}

	audit.Record(c, audit.ActionCreate, audit.ResourceDischarge, &discharge.ID)
	c.JSON(http.StatusCreated, gin.H{"discharge_record": toDischargeResponse(discharge)})
}

// ListDischarges returns a patient's discharge records.
func (h *Handler) ListDischarges(c *gin.Context) {
	principal := auth.MustPrincipal(c)

	patientID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	records, err := h.service.ListDischarges(c.Request.Context(), principal, patientID)
	if err != nil {
		respondPatientError(c, err)
		return
	}

	responses := make([]DischargeResponse, 0, len(records))
	for _, d := range records {
		responses = append(responses, toDischargeResponse(d))
	}

	audit.Record(c, audit.ActionList, audit.ResourceDischarge, &patientID)
	c.JSON(http.StatusOK, gin.H{"discharge_records": responses})
}

type createNoteRequest struct {
	Body string `json:"body" validate:"required,min=2,max=10000"`
}

// CreateNote adds a clinical note.
func (h *Handler) CreateNote(c *gin.Context) {
	principal := auth.MustPrincipal(c)

	patientID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	var req createNoteRequest
	if err := validator.BindJSON(c, &req); err != nil {
		httpx.BadRequest(c, err)
		return
	}

	note, err := h.service.CreateNote(c.Request.Context(), principal, patientID, req.Body)
	if err != nil {
		respondPatientError(c, err)
		return
	}

	audit.Record(c, audit.ActionCreate, audit.ResourceClinicalNote, &note.ID)
	c.JSON(http.StatusCreated, gin.H{"note": toNoteResponse(note)})
}

// ListNotes returns a patient's clinical notes.
func (h *Handler) ListNotes(c *gin.Context) {
	principal := auth.MustPrincipal(c)

	patientID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	notes, err := h.service.ListNotes(c.Request.Context(), principal, patientID)
	if err != nil {
		respondPatientError(c, err)
		return
	}

	responses := make([]NoteResponse, 0, len(notes))
	for _, n := range notes {
		responses = append(responses, toNoteResponse(n))
	}

	audit.Record(c, audit.ActionList, audit.ResourceClinicalNote, &patientID)
	c.JSON(http.StatusOK, gin.H{"notes": responses})
}

func toDischargeResponse(d Discharge) DischargeResponse {
	return DischargeResponse{
		ID:            d.ID.String(),
		PatientID:     d.PatientID.String(),
		ClinicianID:   d.ClinicianID.String(),
		Summary:       d.Summary,
		DischargeDate: d.DischargeDate.Format(DateLayout),
		CreatedAt:     d.CreatedAt.Format(time.RFC3339),
	}
}

func toNoteResponse(n Note) NoteResponse {
	return NoteResponse{
		ID:          n.ID.String(),
		PatientID:   n.PatientID.String(),
		ClinicianID: n.ClinicianID.String(),
		Body:        n.Body,
		CreatedAt:   n.CreatedAt.Format(time.RFC3339),
	}
}

// parseIDParam reads a UUID path parameter, responding 400 on a malformed one.
//
// A malformed id is not a resource, so reporting it plainly reveals nothing
// about which patients exist.
func parseIDParam(c *gin.Context, name string) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param(name))
	if err != nil {
		httpx.BadRequest(c, &validator.ValidationError{
			Message: "request validation failed",
			Fields:  []validator.FieldError{{Field: name, Message: "must be a valid UUID"}},
		})
		return uuid.UUID{}, false
	}
	return id, true
}

// ParseIDParam is the exported form used by the other patient-scoped packages.
func ParseIDParam(c *gin.Context, name string) (uuid.UUID, bool) { return parseIDParam(c, name) }

// respondPatientError maps service errors to responses, collapsing every
// access failure into one indistinguishable 403.
func respondPatientError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrForbidden):
		httpx.Forbidden(c)
	case errors.Is(err, ErrInvalidLink):
		httpx.BadRequest(c, &validator.ValidationError{Message: err.Error()})
	default:
		httpx.Internal(c, err)
	}
}

// RespondError is the exported form used by the other patient-scoped packages
// so they collapse access failures identically.
func RespondError(c *gin.Context, err error) { respondPatientError(c, err) }
