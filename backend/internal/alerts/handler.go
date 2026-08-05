package alerts

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
)

// Handler serves the alert endpoints.
type Handler struct {
	store  *Store
	access *patients.Access
}

// NewHandler builds the alert handler.
func NewHandler(store *Store, access *patients.Access) *Handler {
	return &Handler{store: store, access: access}
}

// AlertResponse is the client-facing alert shape.
type AlertResponse struct {
	ID              string  `json:"id"`
	PatientID       string  `json:"patient_id"`
	PatientFullName string  `json:"patient_full_name"`
	CheckinID       *string `json:"checkin_id,omitempty"`
	Severity        string  `json:"severity"`
	Message         string  `json:"message"`
	Resolved        bool    `json:"resolved"`
	ResolvedAt      *string `json:"resolved_at,omitempty"`
	ResolvedBy      *string `json:"resolved_by,omitempty"`
	CreatedAt       string  `json:"created_at"`
}

// List returns the caller's alerts, most severe unresolved first.
//
// Pass ?include_resolved=true for the full history; the default is the
// unresolved inbox, which is what the clinician dashboard opens on.
func (h *Handler) List(c *gin.Context) {
	principal := auth.MustPrincipal(c)

	includeResolved := c.Query("include_resolved") == "true"

	list, err := h.store.ListForPrincipal(c.Request.Context(), principal, includeResolved)
	if err != nil {
		httpx.Internal(c, err)
		return
	}

	responses := make([]AlertResponse, 0, len(list))
	for _, a := range list {
		responses = append(responses, toAlertResponse(a))
	}

	audit.Record(c, audit.ActionList, audit.ResourceAlert, nil)
	c.JSON(http.StatusOK, gin.H{"alerts": responses})
}

// Resolve marks an alert resolved. Mounted clinician-only.
func (h *Handler) Resolve(c *gin.Context) {
	principal := auth.MustPrincipal(c)

	alertID, ok := patients.ParseIDParam(c, "id")
	if !ok {
		return
	}

	ctx := c.Request.Context()

	patientID, err := h.store.GetPatientID(ctx, alertID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			// An alert that does not exist and one on another clinician's
			// patient are reported identically.
			httpx.Forbidden(c)
			return
		}
		httpx.Internal(c, err)
		return
	}

	// Row-level check: route RBAC only proved the caller is a clinician, not
	// that this is their patient.
	if _, err := h.access.Authorize(ctx, principal, patientID); err != nil {
		patients.RespondError(c, err)
		return
	}

	resolved, err := h.store.Resolve(ctx, alertID, principal.UserID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			httpx.Conflict(c, "this alert has already been resolved")
			return
		}
		httpx.Internal(c, err)
		return
	}

	audit.Record(c, audit.ActionResolve, audit.ResourceAlert, &resolved.ID)
	c.JSON(http.StatusOK, gin.H{"alert": toAlertResponse(resolved)})
}

func toAlertResponse(a Alert) AlertResponse {
	resp := AlertResponse{
		ID:              a.ID.String(),
		PatientID:       a.PatientID.String(),
		PatientFullName: a.PatientFullName,
		Severity:        string(a.Severity),
		Message:         a.Message,
		Resolved:        a.ResolvedAt != nil,
		CreatedAt:       a.CreatedAt.Format(time.RFC3339),
	}
	if a.CheckinID != nil {
		id := a.CheckinID.String()
		resp.CheckinID = &id
	}
	if a.ResolvedAt != nil {
		at := a.ResolvedAt.Format(time.RFC3339)
		resp.ResolvedAt = &at
	}
	if a.ResolvedBy != nil {
		by := a.ResolvedBy.String()
		resp.ResolvedBy = &by
	}
	return resp
}
