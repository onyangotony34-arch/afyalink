# AfyaLink

Post-discharge patient recovery platform for the Kenyan market.

This repository implements one loop end to end:

> A clinician discharges a patient → the patient receives scheduled check-ins and
> medication reminders → symptom answers matching a red-flag rule trigger an
> alert to the clinician and caregiver → the clinician resolves it from a
> dashboard.

---

## Stack

| Layer | Choice |
|---|---|
| Backend | Go 1.25, Gin, `pgx/v5` (explicit SQL, no ORM) |
| Database | PostgreSQL 16, migrations via `golang-migrate` |
| Frontend | React 19 + Vite, Tailwind CSS v4, React Router |
| Auth | JWT access tokens (15 min) + rotating refresh tokens (7 days) |
| Password hashing | Argon2id |
| Field encryption | AES-256-GCM |

---

## Layout

```
backend/
  cmd/api/              HTTP server
  cmd/scheduler/        check-in & dose generation, overdue sweep
  cmd/seed/             demo data (development only)
  internal/
    api/                route table — the authorisation policy in one file
    auth/               Argon2id, JWT, refresh rotation
    middleware/         auth, RBAC, rate limit, CORS, audit, logging
    patients/           patients, discharge records, notes, ownership checks
    medications/        prescriptions and dose logs
    checkins/           check-ins and the red-flag rule engine
    alerts/             alert inbox and resolution
    crypto/             AES-256-GCM helpers
    audit/              append-only access trail
    db/                 pgx pool + embedded migrations
    scheduler/          the periodic jobs
    logging/            structured logging with PHI redaction
  pkg/validator/        strict JSON binding and validation
src/                    React frontend
  lib/                  api client, auth context, i18n copy table, formatting
  components/           small reusable pieces (AlertCard, PatientRow, CheckinForm, …)
  pages/                one directory per role
```

The Go module lives under `backend/` rather than at the repository root so it
can sit alongside the existing Vite app without the two build systems
colliding. The internal package layout matches the spec exactly.

---

## Running locally

**1. Start Postgres** (host port 5433, to avoid a system Postgres on 5432):

```bash
docker compose up -d postgres
```

**2. Configure the backend.** Both secrets are validated at startup — the
process refuses to boot without them, rather than falling back to a weak
default.

```bash
cd backend
cp .env.example .env
echo "JWT_SECRET=$(openssl rand -base64 32)" >> .env
echo "PHI_ENCRYPTION_KEY=$(openssl rand -base64 32)" >> .env
```

**3. Seed and start.** Migrations run automatically at boot.

```bash
go run ./cmd/seed              # demo accounts + Day 2/5/10 check-in templates
go run ./cmd/scheduler -once   # generate check-ins and dose logs
go run ./cmd/api               # http://localhost:8080
```

Run the scheduler continuously in another terminal with `go run ./cmd/scheduler`.

**4. Start the frontend:**

```bash
pnpm install
pnpm dev                       # http://localhost:5173, proxies /api to :8080
```

### Demo accounts

Password for all: `AfyaLinkDemo2026!`

| Email | Role | Notes |
|---|---|---|
| `clinician@afyalink.test` | clinician | Owns both demo patients |
| `caregiver@afyalink.test` | caregiver | Linked to Jane Otieno only |
| `caregiver2@afyalink.test` | caregiver | Linked to nobody — for negative testing |
| `patient1@afyalink.test` | patient | Jane Otieno |
| `patient2@afyalink.test` | patient | Samuel Kiprop |

---

## Tests

```bash
cd backend
go test ./...                    # unit tests; API integration tests skip

psql -h 127.0.0.1 -p 5433 -U afyalink -d postgres -c 'CREATE DATABASE afyalink_test'
export TEST_DATABASE_URL='postgres://afyalink:devpassword@127.0.0.1:5433/afyalink_test?sslmode=disable'
go test ./...                    # now includes the API integration tests
```

The integration suite runs against a real Postgres deliberately. Authorisation
is what is being tested, and the interesting failures — a `WHERE` clause that
forgets to scope by clinician, a join that lets one patient's dose log match
another's — only appear against the database.

`pnpm lint && pnpm build` covers the frontend.

---

## Security notes

Every item on the spec's §5.7 checklist is implemented. The ones worth
explaining:

**Argon2id parameters** — `m=19456 (19 MiB), t=2, p=1`, the current
[OWASP Password Storage Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html)
recommendation. OWASP lists `m=47104, t=1, p=1` as an equally strong
alternative; the 19 MiB variant is chosen because the deployment target is a
small container where memory is scarcer than CPU. Hashes are stored in PHC
format, so the cost can be raised later without invalidating existing
credentials.

**Two layers of authorisation, both required.** `RequireRole` on the route
answers "may a caregiver call this endpoint at all". A row-level ownership check
in the handler answers "may *this* caregiver see *this* patient". There is no
"any clinician can see any patient" path — a clinician who is not the patient's
assigned clinician is refused exactly like a stranger.

**Missing and forbidden are indistinguishable.** Every patient-scoped resource
returns the same opaque 403 whether it does not exist or belongs to someone
else. Otherwise the UUID space becomes an enumeration oracle over the patient
roster. A test asserts the two responses are byte-identical.

**Login failures are indistinguishable.** An unknown email and a wrong password
return the same body, and the unknown-email path performs a throwaway Argon2id
verification so the response time does not reveal which accounts exist.

**Refresh token replay revokes the session family.** Tokens rotate on every use
and are stored only as SHA-256 hashes. Presenting an already-rotated token means
either theft or a client bug, and the two cannot be told apart — so every live
token for that user is revoked and they must sign in again.

**CSRF.** The refresh token lives in an HttpOnly, `SameSite=Strict` cookie scoped
to `/api/auth`, so a browser will not attach it to a cross-site request. Every
other state-changing endpoint authenticates with a `Bearer` header, which
browsers never attach automatically. That is why no separate CSRF token is
needed. **If that cookie is ever relaxed to `SameSite=None`, one must be added.**

**Logging never sees PHI.** Request logs record the matched route template
(`/api/patients/:id`), never the raw URL, because raw paths embed patient UUIDs.
The logger additionally scrubs attributes whose keys look sensitive — a backstop
against a careless call site, not a licence to pass it PHI.

**Field encryption.** `diagnosis`, `allergies`, discharge `summary`, and clinical
note bodies are AES-256-GCM encrypted in the service layer before insert. The
nonce is random per write, so identical diagnoses do not produce identical
ciphertext — otherwise anyone with database access could group patients by
condition without decrypting anything.

### Production secrets

`.env` is for local development only. A production deployment needs a real
secrets manager (Fly.io secrets, Render environment groups, AWS Secrets Manager)
holding:

| Secret | Notes |
|---|---|
| `DATABASE_URL` | With `sslmode=require`. |
| `JWT_SECRET` | ≥32 bytes. Rotating it invalidates outstanding access tokens; users recover silently via their refresh cookie, so rotation is low-impact. |
| `PHI_ENCRYPTION_KEY` | Exactly 32 bytes. **Rotating this without re-encrypting existing rows makes every stored PHI field permanently unreadable.** There is no key-version column in this MVP — adding one is a prerequisite for key rotation. |
| `CORS_ORIGINS` | Exact frontend origin. A wildcard is rejected at startup. |
| `TRUSTED_PROXIES` | The platform's proxy CIDRs. Empty (the default) means no `X-Forwarded-For` is believed. Leave it empty unless you are behind a known proxy — otherwise clients can forge their source IP and bypass the per-IP login rate limit. |

Also set `APP_ENV=production`, which enables `Secure` cookies and HSTS, and
makes `cmd/seed` refuse to run.

---

## Known limitations

Deliberate MVP boundaries, not oversights:

- **Rate limiting is in-process.** With more than one API replica each enforces
  its own bucket, multiplying the effective limit by the replica count. A shared
  store (Redis) is explicitly out of scope for this pass — Phase 2.
- **Audit writes are best-effort after the response.** The response is already
  sent when the audit row is written, so a failure is logged loudly but cannot
  fail the request. Production should alert on those log lines.
- **No PHI key rotation.** See `PHI_ENCRYPTION_KEY` above.
- **Registration creates patients and caregivers only.** Clinician accounts are
  provisioned out of band (the seed command), so a compromised clinician session
  cannot mint more privileged peers.
- **Caregivers do not see the diagnosis.** Spec §3 grants caregivers check-ins,
  medications, and alerts; allergies are included because a caregiver acts on
  them in an emergency, but the diagnosis narrative and clinical notes are
  withheld. This is a policy judgement — one branch in
  `internal/patients/service.go` (`decryptFor`) changes it.

## Not built in this pass

Hospital EMR integration, pharmacy/lab APIs, WhatsApp/SMS/USSD delivery, M-Pesa,
mobile app, CHW role, AI risk scoring, chronic disease and maternal tracks,
telemedicine chat, Redis, push notifications, offline sync, hospital-admin
analytics. The schema and API deliberately assume none of them exist.

The marketing copy on the landing page still describes SMS delivery. That is
existing copy about the product vision, not a claim about this build.
