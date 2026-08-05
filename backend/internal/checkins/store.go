// Package checkins owns scheduled post-discharge check-ins, the patient's
// responses, and the red-flag rule engine that turns those responses into
// clinician alerts.
package checkins

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/OderoCeasar/afyalink/backend/internal/db"
)

// Status mirrors the checkin_status enum.
type Status string

const (
	StatusPending   Status = "pending"
	StatusCompleted Status = "completed"
	StatusMissed    Status = "missed"
)

// Template is a check-in questionnaire pinned to a day offset after discharge.
type Template struct {
	ID            uuid.UUID
	DayOffset     int
	Name          string
	QuestionsJSON []byte
}

// Checkin is a scheduled check-in for one patient, joined with its template.
type Checkin struct {
	ID            uuid.UUID
	PatientID     uuid.UUID
	TemplateID    uuid.UUID
	TemplateName  string
	DayOffset     int
	QuestionsJSON []byte
	ScheduledAt   time.Time
	Status        Status
	CompletedAt   *time.Time
}

// Response is one answered question.
type Response struct {
	ID         uuid.UUID
	CheckinID  uuid.UUID
	QuestionID string
	Answer     string
	IsRedFlag  bool
	CreatedAt  time.Time
}

// Store holds check-in queries.
type Store struct {
	pool *db.Pool
}

// NewStore builds the check-in store.
func NewStore(pool *db.Pool) *Store { return &Store{pool: pool} }

// Pool exposes the pool for the transactional respond path.
func (s *Store) Pool() *db.Pool { return s.pool }

// ListTemplates returns every template, ordered by day offset.
func (s *Store) ListTemplates(ctx context.Context) ([]Template, error) {
	const query = `
		SELECT id, day_offset, name, questions_json
		FROM checkin_templates
		ORDER BY day_offset`

	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("checkins: listing templates: %w", err)
	}
	defer rows.Close()

	var templates []Template
	for rows.Next() {
		var t Template
		if err := rows.Scan(&t.ID, &t.DayOffset, &t.Name, &t.QuestionsJSON); err != nil {
			return nil, fmt.Errorf("checkins: scanning template: %w", err)
		}
		templates = append(templates, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("checkins: iterating templates: %w", err)
	}
	return templates, nil
}

// ListForPatient returns a patient's check-ins, soonest first.
func (s *Store) ListForPatient(ctx context.Context, patientID uuid.UUID) ([]Checkin, error) {
	const query = `
		SELECT c.id, c.patient_id, c.template_id, t.name, t.day_offset,
		       t.questions_json, c.scheduled_at, c.status, c.completed_at
		FROM checkins c
		JOIN checkin_templates t ON t.id = c.template_id
		WHERE c.patient_id = $1
		ORDER BY c.scheduled_at`

	rows, err := s.pool.Query(ctx, query, patientID)
	if err != nil {
		return nil, fmt.Errorf("checkins: listing check-ins: %w", err)
	}
	defer rows.Close()

	var list []Checkin
	for rows.Next() {
		var c Checkin
		if err := rows.Scan(
			&c.ID, &c.PatientID, &c.TemplateID, &c.TemplateName, &c.DayOffset,
			&c.QuestionsJSON, &c.ScheduledAt, &c.Status, &c.CompletedAt,
		); err != nil {
			return nil, fmt.Errorf("checkins: scanning check-in: %w", err)
		}
		list = append(list, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("checkins: iterating check-ins: %w", err)
	}
	return list, nil
}

// GetByID loads one check-in with its template.
func (s *Store) GetByID(ctx context.Context, id uuid.UUID) (Checkin, error) {
	const query = `
		SELECT c.id, c.patient_id, c.template_id, t.name, t.day_offset,
		       t.questions_json, c.scheduled_at, c.status, c.completed_at
		FROM checkins c
		JOIN checkin_templates t ON t.id = c.template_id
		WHERE c.id = $1`

	var c Checkin
	err := s.pool.QueryRow(ctx, query, id).Scan(
		&c.ID, &c.PatientID, &c.TemplateID, &c.TemplateName, &c.DayOffset,
		&c.QuestionsJSON, &c.ScheduledAt, &c.Status, &c.CompletedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Checkin{}, db.ErrNotFound
	}
	if err != nil {
		return Checkin{}, fmt.Errorf("checkins: selecting check-in: %w", err)
	}
	return c, nil
}

// ListResponses returns the answers recorded for a check-in.
func (s *Store) ListResponses(ctx context.Context, checkinID uuid.UUID) ([]Response, error) {
	const query = `
		SELECT id, checkin_id, question_id, answer, is_red_flag, created_at
		FROM checkin_responses
		WHERE checkin_id = $1
		ORDER BY created_at`

	rows, err := s.pool.Query(ctx, query, checkinID)
	if err != nil {
		return nil, fmt.Errorf("checkins: listing responses: %w", err)
	}
	defer rows.Close()

	var responses []Response
	for rows.Next() {
		var r Response
		if err := rows.Scan(&r.ID, &r.CheckinID, &r.QuestionID, &r.Answer, &r.IsRedFlag, &r.CreatedAt); err != nil {
			return nil, fmt.Errorf("checkins: scanning response: %w", err)
		}
		responses = append(responses, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("checkins: iterating responses: %w", err)
	}
	return responses, nil
}

// ListResponsesForPatient returns every answer across a patient's check-ins,
// so the clinician's history view needs one query rather than one per
// check-in.
func (s *Store) ListResponsesForPatient(ctx context.Context, patientID uuid.UUID) ([]Response, error) {
	const query = `
		SELECT r.id, r.checkin_id, r.question_id, r.answer, r.is_red_flag, r.created_at
		FROM checkin_responses r
		JOIN checkins c ON c.id = r.checkin_id
		WHERE c.patient_id = $1
		ORDER BY r.created_at`

	rows, err := s.pool.Query(ctx, query, patientID)
	if err != nil {
		return nil, fmt.Errorf("checkins: listing patient responses: %w", err)
	}
	defer rows.Close()

	var responses []Response
	for rows.Next() {
		var r Response
		if err := rows.Scan(&r.ID, &r.CheckinID, &r.QuestionID, &r.Answer, &r.IsRedFlag, &r.CreatedAt); err != nil {
			return nil, fmt.Errorf("checkins: scanning patient response: %w", err)
		}
		responses = append(responses, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("checkins: iterating patient responses: %w", err)
	}
	return responses, nil
}
