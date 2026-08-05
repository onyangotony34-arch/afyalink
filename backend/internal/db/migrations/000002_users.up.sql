CREATE TABLE users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email TEXT UNIQUE NOT NULL,
  phone TEXT,
  password_hash TEXT NOT NULL,
  role user_role NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Login looks users up by lower(email); the unique index keeps that lookup
-- index-backed and simultaneously enforces case-insensitive uniqueness so
-- Alice@x.com and alice@x.com cannot both register.
CREATE UNIQUE INDEX users_email_lower_key ON users (lower(email));
