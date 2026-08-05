-- diagnosis_enc / allergies_enc hold AES-256-GCM ciphertext produced by
-- internal/crypto. Plaintext must never reach this table.
CREATE TABLE patients (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id),
  full_name TEXT NOT NULL,
  dob DATE NOT NULL,
  diagnosis_enc BYTEA NOT NULL,
  allergies_enc BYTEA,
  emergency_contact TEXT,
  caregiver_id UUID REFERENCES users(id),
  clinician_id UUID NOT NULL REFERENCES users(id),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Ownership checks resolve a patient by each of these three columns on
-- essentially every request, so all three are indexed.
CREATE UNIQUE INDEX patients_user_id_key ON patients (user_id);
CREATE INDEX patients_clinician_id_idx ON patients (clinician_id);
CREATE INDEX patients_caregiver_id_idx ON patients (caregiver_id);
