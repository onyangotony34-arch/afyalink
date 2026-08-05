-- summary_enc holds AES-256-GCM ciphertext.
CREATE TABLE discharge_records (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  patient_id UUID NOT NULL REFERENCES patients(id),
  clinician_id UUID NOT NULL REFERENCES users(id),
  summary_enc BYTEA NOT NULL,
  discharge_date DATE NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX discharge_records_patient_id_idx ON discharge_records (patient_id);
