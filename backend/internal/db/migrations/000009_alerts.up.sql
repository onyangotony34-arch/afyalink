CREATE TABLE alerts (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  patient_id UUID NOT NULL REFERENCES patients(id),
  checkin_id UUID REFERENCES checkins(id),
  severity alert_severity NOT NULL,
  message TEXT NOT NULL,
  resolved_at TIMESTAMPTZ,
  resolved_by UUID REFERENCES users(id),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX alerts_patient_id_idx ON alerts (patient_id, created_at DESC);
CREATE INDEX alerts_unresolved_idx ON alerts (created_at DESC) WHERE resolved_at IS NULL;
