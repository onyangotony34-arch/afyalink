CREATE TABLE medications (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  patient_id UUID NOT NULL REFERENCES patients(id),
  name TEXT NOT NULL,
  dosage TEXT NOT NULL,
  frequency_hours INT NOT NULL,
  start_date DATE NOT NULL,
  end_date DATE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT medications_frequency_hours_positive CHECK (frequency_hours > 0),
  CONSTRAINT medications_end_after_start CHECK (end_date IS NULL OR end_date >= start_date)
);

CREATE INDEX medications_patient_id_idx ON medications (patient_id);

CREATE TABLE medication_logs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  medication_id UUID NOT NULL REFERENCES medications(id),
  scheduled_at TIMESTAMPTZ NOT NULL,
  taken_at TIMESTAMPTZ,
  status medication_log_status NOT NULL DEFAULT 'pending'
);

-- The scheduler regenerates dose rows on every tick; this unique index is what
-- makes that generation idempotent instead of duplicating doses each run.
CREATE UNIQUE INDEX medication_logs_med_scheduled_key
  ON medication_logs (medication_id, scheduled_at);
CREATE INDEX medication_logs_pending_idx
  ON medication_logs (scheduled_at) WHERE status = 'pending';
