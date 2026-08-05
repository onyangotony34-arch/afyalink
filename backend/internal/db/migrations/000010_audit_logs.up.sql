-- Append-only record of every read and write against patient-scoped data.
-- Deliberately holds no PHI: action/resource/resource_id only, never field
-- values. user_id is nullable so failed-auth attempts can still be recorded.
CREATE TABLE audit_logs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID REFERENCES users(id),
  action TEXT NOT NULL,
  resource TEXT NOT NULL,
  resource_id UUID,
  ip_address TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX audit_logs_user_id_idx ON audit_logs (user_id, created_at DESC);
CREATE INDEX audit_logs_resource_idx ON audit_logs (resource, resource_id, created_at DESC);
