-- questions_json shape, per question:
--   {"id": "chest_pain",
--    "text": "Are you experiencing chest pain?",
--    "type": "yes_no" | "scale",
--    "scale_min": 0, "scale_max": 10,        -- scale questions only
--    "critical": true,                        -- optional; escalates severity
--    "red_flag_if": {"op": "eq"|"gte"|"lte", "value": "yes" | 8}}
CREATE TABLE checkin_templates (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  day_offset INT NOT NULL,
  name TEXT NOT NULL,
  questions_json JSONB NOT NULL
);

-- Check-in generation is keyed off day_offset, and seeding must not install the
-- same template twice.
CREATE UNIQUE INDEX checkin_templates_day_offset_key ON checkin_templates (day_offset);

CREATE TABLE checkins (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  patient_id UUID NOT NULL REFERENCES patients(id),
  template_id UUID NOT NULL REFERENCES checkin_templates(id),
  scheduled_at TIMESTAMPTZ NOT NULL,
  status checkin_status NOT NULL DEFAULT 'pending',
  completed_at TIMESTAMPTZ
);

-- Same idempotency argument as medication_logs: the scheduler re-runs
-- generation every tick and must not duplicate a patient's day-2 check-in.
CREATE UNIQUE INDEX checkins_patient_template_key ON checkins (patient_id, template_id);
CREATE INDEX checkins_patient_scheduled_idx ON checkins (patient_id, scheduled_at);
CREATE INDEX checkins_pending_idx ON checkins (scheduled_at) WHERE status = 'pending';

CREATE TABLE checkin_responses (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  checkin_id UUID NOT NULL REFERENCES checkins(id),
  question_id TEXT NOT NULL,
  answer TEXT NOT NULL,
  is_red_flag BOOLEAN NOT NULL DEFAULT false,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- One answer per question per check-in; also blocks a replayed respond call
-- from stacking duplicate answers.
CREATE UNIQUE INDEX checkin_responses_checkin_question_key
  ON checkin_responses (checkin_id, question_id);
