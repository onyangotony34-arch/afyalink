package checkins

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"

	"github.com/OderoCeasar/afyalink/backend/internal/alerts"
	"github.com/OderoCeasar/afyalink/backend/internal/auth"
	"github.com/OderoCeasar/afyalink/backend/internal/db"
	"github.com/OderoCeasar/afyalink/backend/internal/patients"
)

// ErrAlreadyCompleted is returned when a check-in has already been answered.
var ErrAlreadyCompleted = errors.New("this check-in has already been completed")

// AnswerError reports a rejected answer set. It names the offending question
// ids without echoing the answers themselves.
type AnswerError struct {
	Problems map[string]string
}

func (e *AnswerError) Error() string {
	ids := make([]string, 0, len(e.Problems))
	for id := range e.Problems {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return fmt.Sprintf("check-in answers are invalid: %v", ids)
}

// Service applies access policy and the red-flag engine over the store.
type Service struct {
	store  *Store
	access *patients.Access
}

// NewService wires the check-in service.
func NewService(store *Store, access *patients.Access) *Service {
	return &Service{store: store, access: access}
}

// RespondResult reports what a submitted check-in produced.
type RespondResult struct {
	Checkin   Checkin
	Responses []Response
	// AlertID is set when the answers tripped the red-flag engine. The alert
	// is created in the same request, synchronously, so the clinician sees it
	// on their next dashboard poll — spec §5.5.
	AlertID       *uuid.UUID
	AlertSeverity *Severity
}

// ListForPatient returns a patient's check-ins after an ownership check.
func (s *Service) ListForPatient(ctx context.Context, principal auth.Principal, patientID uuid.UUID) ([]Checkin, []Response, error) {
	if _, err := s.access.Authorize(ctx, principal, patientID); err != nil {
		return nil, nil, err
	}

	list, err := s.store.ListForPatient(ctx, patientID)
	if err != nil {
		return nil, nil, err
	}

	responses, err := s.store.ListResponsesForPatient(ctx, patientID)
	if err != nil {
		return nil, nil, err
	}

	return list, responses, nil
}

// Respond records a patient's answers and raises an alert if any red flag
// fires.
//
// Everything after validation happens in one transaction: the responses, the
// status change, and the alert. A half-applied check-in — answers recorded but
// no alert — is the single worst failure mode this system has, so it is made
// impossible rather than merely unlikely.
func (s *Service) Respond(
	ctx context.Context,
	principal auth.Principal,
	checkinID uuid.UUID,
	answers map[string]string,
) (RespondResult, error) {
	checkin, err := s.store.GetByID(ctx, checkinID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			// Indistinguishable from "not yours".
			return RespondResult{}, patients.ErrForbidden
		}
		return RespondResult{}, err
	}

	if _, err := s.access.Authorize(ctx, principal, checkin.PatientID); err != nil {
		return RespondResult{}, err
	}

	if checkin.Status == StatusCompleted {
		return RespondResult{}, ErrAlreadyCompleted
	}

	questions, err := ParseQuestions(checkin.QuestionsJSON)
	if err != nil {
		return RespondResult{}, err
	}

	evaluated, flagged, err := evaluate(questions, answers)
	if err != nil {
		return RespondResult{}, err
	}

	tx, err := s.store.Pool().Begin(ctx)
	if err != nil {
		return RespondResult{}, fmt.Errorf("checkins: beginning respond transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Guarding on the current status inside the transaction makes two
	// concurrent submissions safe: exactly one updates a row, and the loser is
	// reported as already completed rather than raising a duplicate alert.
	const complete = `
		UPDATE checkins
		SET status = 'completed', completed_at = now()
		WHERE id = $1 AND status <> 'completed'`

	tag, err := tx.Exec(ctx, complete, checkinID)
	if err != nil {
		return RespondResult{}, fmt.Errorf("checkins: completing check-in: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return RespondResult{}, ErrAlreadyCompleted
	}

	const insertResponse = `
		INSERT INTO checkin_responses (checkin_id, question_id, answer, is_red_flag)
		VALUES ($1, $2, $3, $4)
		RETURNING id, checkin_id, question_id, answer, is_red_flag, created_at`

	recorded := make([]Response, 0, len(evaluated))
	for _, e := range evaluated {
		var r Response
		if err := tx.QueryRow(ctx, insertResponse, checkinID, e.QuestionID, e.Answer, e.IsRedFlag).
			Scan(&r.ID, &r.CheckinID, &r.QuestionID, &r.Answer, &r.IsRedFlag, &r.CreatedAt); err != nil {
			return RespondResult{}, fmt.Errorf("checkins: inserting response: %w", err)
		}
		recorded = append(recorded, r)
	}

	result := RespondResult{Responses: recorded}

	if severity, raise := DeriveSeverity(flagged); raise {
		alertSeverity := alerts.Severity(severity)
		message := AlertMessage(checkin.TemplateName, flagged)

		alertID, err := alerts.CreateWith(ctx, tx, checkin.PatientID, &checkinID, alertSeverity, message)
		if err != nil {
			return RespondResult{}, err
		}

		result.AlertID = &alertID
		result.AlertSeverity = &severity
	}

	if err := tx.Commit(ctx); err != nil {
		return RespondResult{}, fmt.Errorf("checkins: committing check-in response: %w", err)
	}

	completedAt := time.Now()
	checkin.Status = StatusCompleted
	checkin.CompletedAt = &completedAt
	result.Checkin = checkin

	return result, nil
}

// evaluatedAnswer pairs an answer with its red-flag verdict.
type evaluatedAnswer struct {
	QuestionID string
	Answer     string
	IsRedFlag  bool
}

// evaluate validates the submitted answers against the template and applies
// each question's red-flag rule.
//
// Every question must be answered and no unknown ids are accepted. A partial
// submission is rejected rather than stored, because a check-in missing the
// chest-pain question would otherwise be recorded as completed with no flag.
func evaluate(questions Questions, answers map[string]string) ([]evaluatedAnswer, []Question, error) {
	problems := make(map[string]string)

	for id := range answers {
		if _, ok := questions.Find(id); !ok {
			problems[id] = "is not a question in this check-in"
		}
	}

	evaluated := make([]evaluatedAnswer, 0, len(questions))
	var flagged []Question

	for _, q := range questions {
		answer, ok := answers[q.ID]
		if !ok {
			problems[q.ID] = "is required"
			continue
		}

		if err := q.ValidateAnswer(answer); err != nil {
			problems[q.ID] = err.Error()
			continue
		}

		isRedFlag := q.IsRedFlag(answer)
		if isRedFlag {
			flagged = append(flagged, q)
		}

		evaluated = append(evaluated, evaluatedAnswer{
			QuestionID: q.ID,
			Answer:     answer,
			IsRedFlag:  isRedFlag,
		})
	}

	if len(problems) > 0 {
		return nil, nil, &AnswerError{Problems: problems}
	}
	return evaluated, flagged, nil
}
