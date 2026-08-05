-- Core domain enums. Kept in their own migration so later migrations can
-- reference them without ordering ambiguity.

CREATE TYPE user_role AS ENUM ('patient', 'caregiver', 'clinician');
CREATE TYPE alert_severity AS ENUM ('low', 'medium', 'high', 'critical');
CREATE TYPE medication_log_status AS ENUM ('pending', 'taken', 'missed');
CREATE TYPE checkin_status AS ENUM ('pending', 'completed', 'missed');
