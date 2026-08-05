package checkins

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/OderoCeasar/afyalink/backend/internal/audit"
	"github.com/OderoCeasar/afyalink/backend/internal/auth"
	"github.com/OderoCeasar/afyalink/backend/internal/httpx"
	"github.com/OderoCeasar/afyalink/backend/internal/patients"
	"github.com/OderoCeasar/afyalink/backend/pkg/validator"
)

// Handler serves the check-in endpoints.
type Handler struct {
	service *Service
}

// NewHandler builds the check-in handler.
func NewHandler(service *Service) *Handler { return &Handler{service: service} }

// QuestionResponse is a question as shown to a client.
//
// The red_flag_if rule is deliberately not serialised. Sending a patient the
// thresholds that trigger an alert invites answering just under them, and a
// caregiver has no use for the rule at all. Evaluation stays server-side.
type QuestionResponse struct {
	ID       string `json:"id"`
	Text     string `json:"text"`
	Type     string `json:"type"`
	ScaleMin *int   `json:"scale_min,omitempty"`
	ScaleMax *int   `json:"scale_max,omitempty"`
}

// ResponseItem is one recorded answer.
type ResponseItem struct {
	QuestionID string `json:"question_id"`
	Answer     string `json:"answer"`
	IsRedFlag  bool   `json:"is_red_flag"`
	CreatedAt  string `json:"created_at"`
}

// CheckinResponse is the client-facing check-in shape.
type CheckinResponse struct {
	ID           string             `json:"id"`
	PatientID    string             `json:"patient_id"`
	TemplateID   string             `json:"template_id"`
	TemplateName string             `json:"template_name"`
	DayOffset    int                `json:"day_offset"`
	ScheduledAt  string             `json:"scheduled_at"`
	Status       string             `json:"status"`
	CompletedAt  *string            `json:"completed_at,omitempty"`
	Questions    []QuestionResponse `json:"questions"`
	Responses    []ResponseItem     `json:"responses"`
}

// List returns a patient's check-ins with any recorded answers.
func (h *Handler) List(c *gin.Context) {
	principal := auth.MustPrincipal(c)

	patientID, ok := patients.ParseIDParam(c, "id")
	if !ok {
		return
	}

	list, allResponses, err := h.service.ListForPatient(c.Request.Context(), principal, patientID)
	if err != nil {
		respondCheckinError(c, err)
		return
	}

	byCheckin := make(map[string][]ResponseItem, len(list))
	for _, r := range allResponses {
		key := r.CheckinID.String()
		byCheckin[key] = append(byCheckin[key], ResponseItem{
			QuestionID: r.QuestionID,
			Answer:     r.Answer,
			IsRedFlag:  r.IsRedFlag,
			CreatedAt:  r.CreatedAt.Format(time.RFC3339),
		})
	}

	responses := make([]CheckinResponse, 0, len(list))
	for _, item := range list {
		resp, err := toCheckinResponse(item)
		if err != nil {
			httpx.Internal(c, err)
			return
		}
		resp.Responses = byCheckin[item.ID.String()]
		if resp.Responses == nil {
			resp.Responses = []ResponseItem{}
		}
		responses = append(responses, resp)
	}

	audit.Record(c, audit.ActionList, audit.ResourceCheckin, &patientID)
	c.JSON(http.StatusOK, gin.H{"checkins": responses})
}

type respondRequest struct {
	// Answers maps question id to answer. Values are strings even for scale
	// questions so the wire format stays uniform; the rule engine parses and
	// range-checks them against the template.
	Answers map[string]string `json:"answers" validate:"required,min=1"`
}

// Respond records a patient's answers, evaluating red flags synchronously.
func (h *Handler) Respond(c *gin.Context) {
	principal := auth.MustPrincipal(c)

	checkinID, ok := patients.ParseIDParam(c, "id")
	if !ok {
		return
	}

	var req respondRequest
	if err := validator.BindJSON(c, &req); err != nil {
		httpx.BadRequest(c, err)
		return
	}

	result, err := h.service.Respond(c.Request.Context(), principal, checkinID, req.Answers)
	if err != nil {
		respondCheckinError(c, err)
		return
	}

	resp, err := toCheckinResponse(result.Checkin)
	if err != nil {
		httpx.Internal(c, err)
		return
	}
	for _, r := range result.Responses {
		resp.Responses = append(resp.Responses, ResponseItem{
			QuestionID: r.QuestionID,
			Answer:     r.Answer,
			IsRedFlag:  r.IsRedFlag,
			CreatedAt:  r.CreatedAt.Format(time.RFC3339),
		})
	}

	body := gin.H{"checkin": resp}

	// The alert id is returned so the patient UI can acknowledge that a
	// clinician has been notified. The alert message is not returned — it
	// summarises symptoms for the clinician's inbox, not for the patient.
	if result.AlertID != nil {
		body["alert"] = gin.H{
			"id":       result.AlertID.String(),
			"severity": string(*result.AlertSeverity),
		}
	}

	audit.Record(c, audit.ActionRespond, audit.ResourceCheckin, &checkinID)
	c.JSON(http.StatusOK, body)
}

func toCheckinResponse(item Checkin) (CheckinResponse, error) {
	var questions Questions
	if err := json.Unmarshal(item.QuestionsJSON, &questions); err != nil {
		return CheckinResponse{}, err
	}

	resp := CheckinResponse{
		ID:           item.ID.String(),
		PatientID:    item.PatientID.String(),
		TemplateID:   item.TemplateID.String(),
		TemplateName: item.TemplateName,
		DayOffset:    item.DayOffset,
		ScheduledAt:  item.ScheduledAt.Format(time.RFC3339),
		Status:       string(item.Status),
		Questions:    make([]QuestionResponse, 0, len(questions)),
		Responses:    []ResponseItem{},
	}
	if item.CompletedAt != nil {
		at := item.CompletedAt.Format(time.RFC3339)
		resp.CompletedAt = &at
	}

	for _, q := range questions {
		resp.Questions = append(resp.Questions, QuestionResponse{
			ID:       q.ID,
			Text:     q.Text,
			Type:     string(q.Type),
			ScaleMin: q.ScaleMin,
			ScaleMax: q.ScaleMax,
		})
	}
	return resp, nil
}

func respondCheckinError(c *gin.Context, err error) {
	var answerErr *AnswerError
	switch {
	case errors.Is(err, patients.ErrForbidden):
		httpx.Forbidden(c)
	case errors.Is(err, ErrAlreadyCompleted):
		httpx.Conflict(c, "this check-in has already been completed")
	case errors.As(err, &answerErr):
		fields := make([]validator.FieldError, 0, len(answerErr.Problems))
		for id, problem := range answerErr.Problems {
			fields = append(fields, validator.FieldError{Field: "answers." + id, Message: problem})
		}
		httpx.BadRequest(c, &validator.ValidationError{
			Message: "check-in answers are invalid",
			Fields:  fields,
		})
	default:
		httpx.Internal(c, err)
	}
}
