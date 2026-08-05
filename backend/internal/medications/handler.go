package medications

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/OderoCeasar/afyalink/backend/internal/audit"
	"github.com/OderoCeasar/afyalink/backend/internal/auth"
	"github.com/OderoCeasar/afyalink/backend/internal/db"
	"github.com/OderoCeasar/afyalink/backend/internal/httpx"
	"github.com/OderoCeasar/afyalink/backend/internal/patients"
	"github.com/OderoCeasar/afyalink/backend/pkg/validator"
)

// scheduleWindow is how far either side of "now" the medication list returns
// dose logs.
//
// ±24h rather than a calendar day on purpose: it spans today in any nearby
// timezone, so the backend needs no timezone configuration and the frontend
// groups doses by the user's own local day.
const scheduleWindow = 24 * time.Hour

// Handler serves the medication endpoints.
type Handler struct {
	store  *Store
	access *patients.Access
}

// NewHandler builds the medication handler.
func NewHandler(store *Store, access *patients.Access) *Handler {
	return &Handler{store: store, access: access}
}

// MedicationResponse is the client-facing prescription shape, including the
// doses scheduled around now.
type MedicationResponse struct {
	ID             string        `json:"id"`
	PatientID      string        `json:"patient_id"`
	Name           string        `json:"name"`
	Dosage         string        `json:"dosage"`
	FrequencyHours int           `json:"frequency_hours"`
	StartDate      string        `json:"start_date"`
	EndDate        *string       `json:"end_date,omitempty"`
	Logs           []LogResponse `json:"logs"`
}

// LogResponse is the client-facing dose shape.
type LogResponse struct {
	ID           string  `json:"id"`
	MedicationID string  `json:"medication_id"`
	ScheduledAt  string  `json:"scheduled_at"`
	TakenAt      *string `json:"taken_at,omitempty"`
	Status       string  `json:"status"`
}

type createMedicationRequest struct {
	Name           string  `json:"name" validate:"required,min=1,max=200"`
	Dosage         string  `json:"dosage" validate:"required,min=1,max=100"`
	FrequencyHours int     `json:"frequency_hours" validate:"required,gte=1,lte=168"`
	StartDate      string  `json:"start_date" validate:"required,datetime=2006-01-02"`
	EndDate        *string `json:"end_date" validate:"omitempty,datetime=2006-01-02"`
}

// Create prescribes a medication. Mounted clinician-only.
func (h *Handler) Create(c *gin.Context) {
	principal := auth.MustPrincipal(c)

	patientID, ok := patients.ParseIDParam(c, "id")
	if !ok {
		return
	}

	if _, err := h.access.Authorize(c.Request.Context(), principal, patientID); err != nil {
		patients.RespondError(c, err)
		return
	}

	var req createMedicationRequest
	if err := validator.BindJSON(c, &req); err != nil {
		httpx.BadRequest(c, err)
		return
	}

	startDate, err := time.Parse(patients.DateLayout, req.StartDate)
	if err != nil {
		httpx.BadRequest(c, fieldError("start_date", "must be a date in YYYY-MM-DD form"))
		return
	}

	var endDate *time.Time
	if req.EndDate != nil {
		parsed, err := time.Parse(patients.DateLayout, *req.EndDate)
		if err != nil {
			httpx.BadRequest(c, fieldError("end_date", "must be a date in YYYY-MM-DD form"))
			return
		}
		if parsed.Before(startDate) {
			httpx.BadRequest(c, fieldError("end_date", "must not be before start_date"))
			return
		}
		endDate = &parsed
	}

	med, err := h.store.Create(c.Request.Context(), CreateParams{
		PatientID:      patientID,
		Name:           req.Name,
		Dosage:         req.Dosage,
		FrequencyHours: req.FrequencyHours,
		StartDate:      startDate,
		EndDate:        endDate,
	})
	if err != nil {
		httpx.Internal(c, err)
		return
	}

	audit.Record(c, audit.ActionCreate, audit.ResourceMedication, &med.ID)
	c.JSON(http.StatusCreated, gin.H{"medication": toMedicationResponse(med, nil)})
}

// List returns a patient's medications with their surrounding doses.
func (h *Handler) List(c *gin.Context) {
	principal := auth.MustPrincipal(c)

	patientID, ok := patients.ParseIDParam(c, "id")
	if !ok {
		return
	}

	if _, err := h.access.Authorize(c.Request.Context(), principal, patientID); err != nil {
		patients.RespondError(c, err)
		return
	}

	ctx := c.Request.Context()

	meds, err := h.store.ListForPatient(ctx, patientID)
	if err != nil {
		httpx.Internal(c, err)
		return
	}

	now := time.Now()
	logs, err := h.store.ListLogsForPatientWindow(ctx, patientID, now.Add(-scheduleWindow), now.Add(scheduleWindow))
	if err != nil {
		httpx.Internal(c, err)
		return
	}

	byMedication := make(map[string][]Log, len(meds))
	for _, l := range logs {
		key := l.MedicationID.String()
		byMedication[key] = append(byMedication[key], l)
	}

	responses := make([]MedicationResponse, 0, len(meds))
	for _, m := range meds {
		responses = append(responses, toMedicationResponse(m, byMedication[m.ID.String()]))
	}

	audit.Record(c, audit.ActionList, audit.ResourceMedication, &patientID)
	c.JSON(http.StatusOK, gin.H{"medications": responses})
}

// MarkTaken records a dose as taken. Mounted patient-only.
func (h *Handler) MarkTaken(c *gin.Context) {
	principal := auth.MustPrincipal(c)

	medicationID, ok := patients.ParseIDParam(c, "id")
	if !ok {
		return
	}
	logID, ok := patients.ParseIDParam(c, "logId")
	if !ok {
		return
	}

	ctx := c.Request.Context()

	owner, err := h.store.GetLogOwner(ctx, medicationID, logID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			// A dose that does not exist and one belonging to another patient
			// must be indistinguishable.
			httpx.Forbidden(c)
			return
		}
		httpx.Internal(c, err)
		return
	}

	// Row-level check: the caller must own the patient this dose belongs to.
	// Route RBAC only established that they are *a* patient.
	if _, err := h.access.Authorize(ctx, principal, owner.PatientID); err != nil {
		patients.RespondError(c, err)
		return
	}

	updated, err := h.store.MarkTaken(ctx, logID, time.Now())
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			httpx.Conflict(c, "this dose has already been recorded as taken")
			return
		}
		httpx.Internal(c, err)
		return
	}

	audit.Record(c, audit.ActionUpdate, audit.ResourceMedicationLog, &updated.ID)
	c.JSON(http.StatusOK, gin.H{"log": toLogResponse(updated)})
}

func toMedicationResponse(m Medication, logs []Log) MedicationResponse {
	resp := MedicationResponse{
		ID:             m.ID.String(),
		PatientID:      m.PatientID.String(),
		Name:           m.Name,
		Dosage:         m.Dosage,
		FrequencyHours: m.FrequencyHours,
		StartDate:      m.StartDate.Format(patients.DateLayout),
		Logs:           make([]LogResponse, 0, len(logs)),
	}
	if m.EndDate != nil {
		end := m.EndDate.Format(patients.DateLayout)
		resp.EndDate = &end
	}
	for _, l := range logs {
		resp.Logs = append(resp.Logs, toLogResponse(l))
	}
	return resp
}

func toLogResponse(l Log) LogResponse {
	resp := LogResponse{
		ID:           l.ID.String(),
		MedicationID: l.MedicationID.String(),
		ScheduledAt:  l.ScheduledAt.Format(time.RFC3339),
		Status:       string(l.Status),
	}
	if l.TakenAt != nil {
		taken := l.TakenAt.Format(time.RFC3339)
		resp.TakenAt = &taken
	}
	return resp
}

func fieldError(field, message string) *validator.ValidationError {
	return &validator.ValidationError{
		Message: "request validation failed",
		Fields:  []validator.FieldError{{Field: field, Message: message}},
	}
}
