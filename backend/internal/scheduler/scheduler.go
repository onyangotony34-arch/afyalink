// Package scheduler generates the recurring rows the recovery loop depends on:
// post-discharge check-ins, medication dose logs, and the overdue sweep that
// turns unanswered rows into missed ones.
//
// Every statement here is idempotent — generation relies on the unique indexes
// declared in the migrations plus ON CONFLICT DO NOTHING, and the sweeps only
// touch rows still in 'pending'. A tick that runs twice, or two schedulers
// running at once, produce the same result as one.
package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/OderoCeasar/afyalink/backend/internal/db"
)

// clinicTimezone anchors generated schedules to Kenyan local time.
//
// Without it, "day 2 at 09:00" would land at 09:00 UTC, i.e. noon in Nairobi,
// and a dose meant for the morning would notify a patient at lunchtime.
const clinicTimezone = "Africa/Nairobi"

// Scheduling defaults.
const (
	// checkinHourLocal is the local hour a generated check-in becomes due.
	checkinHourLocal = 9

	// doseStartHourLocal is the local hour the first dose of a prescription
	// is scheduled; subsequent doses follow frequency_hours from there.
	doseStartHourLocal = 8

	// doseHorizon is how far ahead dose logs are materialised. Generating the
	// full course up front would create thousands of rows for a long
	// prescription; a rolling window keeps the table proportional to active
	// treatment.
	doseHorizon = 48 * time.Hour

	// doseGraceHours is how long after its scheduled time a dose may still be
	// marked taken before it counts as missed.
	doseGraceHours = 2

	// checkinGraceHours is the equivalent for check-ins. It is a full day
	// because a patient recovering at home should not be marked delinquent for
	// answering that evening instead of that morning.
	checkinGraceHours = 24
)

// Scheduler runs the periodic generation and sweep jobs.
type Scheduler struct {
	pool   *db.Pool
	logger *slog.Logger
	period time.Duration
}

// New builds a scheduler.
func New(pool *db.Pool, logger *slog.Logger, period time.Duration) *Scheduler {
	return &Scheduler{pool: pool, logger: logger, period: period}
}

// Result reports what one tick changed, for logging and for the end-to-end
// test to assert against.
type Result struct {
	CheckinsCreated    int64
	DoseLogsCreated    int64
	CheckinsMissed     int64
	DosesMissed        int64
	MissedDoseAlerts   int64
	GenerationDuration time.Duration
}

// Run ticks until the context is cancelled. It runs one pass immediately so a
// freshly started process does not wait a full period before catching up.
func (s *Scheduler) Run(ctx context.Context) error {
	ticker := time.NewTicker(s.period)
	defer ticker.Stop()

	s.runOnceLogged(ctx)

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("scheduler stopping")
			return ctx.Err()
		case <-ticker.C:
			s.runOnceLogged(ctx)
		}
	}
}

func (s *Scheduler) runOnceLogged(ctx context.Context) {
	result, err := s.RunOnce(ctx)
	if err != nil {
		// A failed tick is logged and the loop continues: a transient database
		// blip must not kill the process that generates every future check-in.
		s.logger.Error("scheduler tick failed", "err", err)
		return
	}

	s.logger.Info("scheduler tick",
		"checkins_created", result.CheckinsCreated,
		"dose_logs_created", result.DoseLogsCreated,
		"checkins_missed", result.CheckinsMissed,
		"doses_missed", result.DosesMissed,
		"missed_dose_alerts", result.MissedDoseAlerts,
		"duration_ms", result.GenerationDuration.Milliseconds(),
	)
}

// RunOnce performs a single pass. Exported so cmd/scheduler -once and the
// end-to-end tests can drive it deterministically instead of waiting on a
// ticker.
func (s *Scheduler) RunOnce(ctx context.Context) (Result, error) {
	start := time.Now()
	var result Result
	var err error

	if result.CheckinsCreated, err = s.generateCheckins(ctx); err != nil {
		return result, err
	}
	if result.DoseLogsCreated, err = s.generateDoseLogs(ctx); err != nil {
		return result, err
	}
	if result.CheckinsMissed, err = s.sweepMissedCheckins(ctx); err != nil {
		return result, err
	}
	if result.DosesMissed, result.MissedDoseAlerts, err = s.sweepMissedDoses(ctx); err != nil {
		return result, err
	}

	result.GenerationDuration = time.Since(start)
	return result, nil
}

// generateCheckins materialises one check-in per template per patient, offset
// from that patient's most recent discharge date.
//
// DISTINCT ON picks the latest discharge: a readmitted patient's check-in
// schedule should follow their newest discharge, not their first.
func (s *Scheduler) generateCheckins(ctx context.Context) (int64, error) {
	const query = `
		WITH latest_discharge AS (
			SELECT DISTINCT ON (patient_id) patient_id, discharge_date
			FROM discharge_records
			ORDER BY patient_id, discharge_date DESC, created_at DESC
		)
		INSERT INTO checkins (patient_id, template_id, scheduled_at)
		SELECT
			ld.patient_id,
			t.id,
			((ld.discharge_date + t.day_offset)::timestamp + make_interval(hours => $1))
				AT TIME ZONE $2
		FROM latest_discharge ld
		CROSS JOIN checkin_templates t
		ON CONFLICT (patient_id, template_id) DO NOTHING`

	tag, err := s.pool.Exec(ctx, query, checkinHourLocal, clinicTimezone)
	if err != nil {
		return 0, fmt.Errorf("scheduler: generating check-ins: %w", err)
	}
	return tag.RowsAffected(), nil
}

// generateDoseLogs materialises dose rows from each prescription's start date
// up to the rolling horizon, stopping at end_date where one is set.
func (s *Scheduler) generateDoseLogs(ctx context.Context) (int64, error) {
	const query = `
		INSERT INTO medication_logs (medication_id, scheduled_at)
		SELECT m.id, dose_time
		FROM medications m
		CROSS JOIN LATERAL generate_series(
			(m.start_date::timestamp + make_interval(hours => $1)) AT TIME ZONE $3,
			LEAST(
				$2::timestamptz,
				COALESCE(
					((m.end_date + 1)::timestamp) AT TIME ZONE $3,
					$2::timestamptz
				)
			),
			make_interval(hours => m.frequency_hours)
		) AS dose_time
		ON CONFLICT (medication_id, scheduled_at) DO NOTHING`

	horizon := time.Now().Add(doseHorizon)

	tag, err := s.pool.Exec(ctx, query, doseStartHourLocal, horizon, clinicTimezone)
	if err != nil {
		return 0, fmt.Errorf("scheduler: generating dose logs: %w", err)
	}
	return tag.RowsAffected(), nil
}

// sweepMissedCheckins marks overdue pending check-ins as missed.
//
// Per spec §5.6 this raises no alert: an unanswered check-in is visible on the
// clinician's patient timeline, and alerting on every one would bury the
// red-flag alerts that actually need attention.
func (s *Scheduler) sweepMissedCheckins(ctx context.Context) (int64, error) {
	const query = `
		UPDATE checkins
		SET status = 'missed'
		WHERE status = 'pending'
		  AND scheduled_at < now() - make_interval(hours => $1)`

	tag, err := s.pool.Exec(ctx, query, checkinGraceHours)
	if err != nil {
		return 0, fmt.Errorf("scheduler: sweeping missed check-ins: %w", err)
	}
	return tag.RowsAffected(), nil
}

// sweepMissedDoses marks overdue doses missed and raises a low-severity alert
// for each, in one statement.
//
// The CTE matters: marking the dose and raising its alert in a single
// statement means there is no window in which a dose is missed but unalerted,
// and the `status = 'pending'` predicate guarantees each dose alerts exactly
// once no matter how often the sweep runs.
func (s *Scheduler) sweepMissedDoses(ctx context.Context) (int64, int64, error) {
	const query = `
		WITH newly_missed AS (
			UPDATE medication_logs
			SET status = 'missed'
			WHERE status = 'pending'
			  AND scheduled_at < now() - make_interval(hours => $1)
			RETURNING id, medication_id, scheduled_at
		),
		raised AS (
			INSERT INTO alerts (patient_id, severity, message)
			SELECT
				m.patient_id,
				'low',
				'Missed dose: ' || m.name || ' ' || m.dosage || ', scheduled for '
					|| to_char(nm.scheduled_at AT TIME ZONE $2, 'DD Mon HH24:MI')
			FROM newly_missed nm
			JOIN medications m ON m.id = nm.medication_id
			RETURNING id
		)
		SELECT
			(SELECT count(*) FROM newly_missed),
			(SELECT count(*) FROM raised)`

	var missed, alerted int64
	if err := s.pool.QueryRow(ctx, query, doseGraceHours, clinicTimezone).Scan(&missed, &alerted); err != nil {
		return 0, 0, fmt.Errorf("scheduler: sweeping missed doses: %w", err)
	}
	return missed, alerted, nil
}
