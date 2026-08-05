-- Not in the spec's §4 schema, but §3 and §6 both require clinicians to add
-- clinical notes, so the table is added here rather than faked in the UI.
-- Note bodies are free-text clinical observations, i.e. PHI, so the body is
-- AES-256-GCM encrypted like every other clinical narrative field.
CREATE TABLE clinical_notes (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  patient_id UUID NOT NULL REFERENCES patients(id),
  clinician_id UUID NOT NULL REFERENCES users(id),
  body_enc BYTEA NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX clinical_notes_patient_id_idx ON clinical_notes (patient_id, created_at DESC);
