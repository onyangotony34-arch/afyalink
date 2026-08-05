package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/OderoCeasar/afyalink/backend/internal/api"
	"github.com/OderoCeasar/afyalink/backend/internal/auth"
	"github.com/OderoCeasar/afyalink/backend/internal/config"
	"github.com/OderoCeasar/afyalink/backend/internal/crypto"
	"github.com/OderoCeasar/afyalink/backend/internal/db"
)

// These tests exercise the real router against a real Postgres. Authorisation
// is the thing being tested, and a mocked store could only prove that the mock
// agrees with itself — the interesting failures (a WHERE clause that forgets to
// scope by clinician, a join that lets one patient's log match another's) only
// show up against the database.
var (
	testPool   *db.Pool
	testConfig *config.Config
	testCipher *crypto.Cipher
	testJWT    *auth.JWTManager
)

func TestMain(m *testing.M) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		fmt.Fprintln(os.Stderr, "TEST_DATABASE_URL is not set; skipping API integration tests")
		fmt.Fprintln(os.Stderr, "  start the dev database with: docker compose up -d postgres")
		fmt.Fprintln(os.Stderr, "  then: createdb afyalink_test and export TEST_DATABASE_URL")
		os.Exit(0)
	}

	gin.SetMode(gin.TestMode)

	if err := db.Migrate(databaseURL); err != nil {
		fmt.Fprintf(os.Stderr, "migrating test database: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
	pool, err := db.Connect(ctx, databaseURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "connecting to test database: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()
	testPool = pool

	phiKey := make([]byte, 32)
	for i := range phiKey {
		phiKey[i] = byte(i * 7)
	}

	testConfig = &config.Config{
		Env:             "test",
		Port:            "0",
		DatabaseURL:     databaseURL,
		JWTSecret:       []byte(strings.Repeat("test-jwt-secret!", 2)),
		PHIKey:          phiKey,
		CORSOrigins:     []string{"http://localhost:5173"},
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 7 * 24 * time.Hour,
		LoginRateLimit:  1000, // effectively off; rate limiting has its own test
		LoginRateBurst:  1000,
		SchedulerPeriod: time.Minute,
	}

	if testCipher, err = crypto.New(testConfig.PHIKey); err != nil {
		fmt.Fprintf(os.Stderr, "building test cipher: %v\n", err)
		os.Exit(1)
	}
	testJWT = auth.NewJWTManager(testConfig.JWTSecret, testConfig.AccessTokenTTL)

	os.Exit(m.Run())
}

// env is one test's isolated world: a fresh router over a freshly seeded
// database.
type env struct {
	t      *testing.T
	router *gin.Engine
	fx     fixtures
}

// fixtures is the cast of characters every authorisation test needs.
//
// The shape is the point: two clinicians, two patients, and two caregivers,
// with A and B linked to disjoint patients. Every "wrong owner" test is then
// just "actor from set B reaches for resource from set A".
type fixtures struct {
	ClinicianA  uuid.UUID
	ClinicianB  uuid.UUID
	CaregiverA  uuid.UUID
	CaregiverB  uuid.UUID
	PatientUsrA uuid.UUID
	PatientUsrB uuid.UUID

	PatientA uuid.UUID
	PatientB uuid.UUID

	TemplateID uuid.UUID
	CheckinA   uuid.UUID
	CheckinB   uuid.UUID

	MedicationA uuid.UUID
	MedicationB uuid.UUID
	DoseLogA    uuid.UUID
	DoseLogB    uuid.UUID

	AlertA uuid.UUID
	AlertB uuid.UUID
}

func newEnv(t *testing.T) *env {
	t.Helper()

	if testPool == nil {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	truncateAll(t)

	router, err := api.New(api.Deps{
		Config: testConfig,
		Pool:   testPool,
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		Cipher: testCipher,
	})
	if err != nil {
		t.Fatalf("building router: %v", err)
	}

	return &env{t: t, router: router, fx: seedFixtures(t)}
}

func truncateAll(t *testing.T) {
	t.Helper()

	const query = `
		TRUNCATE audit_logs, alerts, checkin_responses, checkins, checkin_templates,
		         medication_logs, medications, clinical_notes, discharge_records,
		         patients, refresh_tokens, users
		RESTART IDENTITY CASCADE`

	if _, err := testPool.Exec(context.Background(), query); err != nil {
		t.Fatalf("truncating test database: %v", err)
	}
}

func seedFixtures(t *testing.T) fixtures {
	t.Helper()
	ctx := context.Background()

	var fx fixtures

	hash, err := auth.HashPassword(testPassword)
	if err != nil {
		t.Fatalf("hashing fixture password: %v", err)
	}

	newUser := func(email string, role auth.Role) uuid.UUID {
		var id uuid.UUID
		err := testPool.QueryRow(ctx,
			`INSERT INTO users (email, password_hash, role) VALUES ($1, $2, $3) RETURNING id`,
			email, hash, string(role)).Scan(&id)
		if err != nil {
			t.Fatalf("creating fixture user %s: %v", email, err)
		}
		return id
	}

	fx.ClinicianA = newUser("clinician.a@test.local", auth.RoleClinician)
	fx.ClinicianB = newUser("clinician.b@test.local", auth.RoleClinician)
	fx.CaregiverA = newUser("caregiver.a@test.local", auth.RoleCaregiver)
	fx.CaregiverB = newUser("caregiver.b@test.local", auth.RoleCaregiver)
	fx.PatientUsrA = newUser("patient.a@test.local", auth.RolePatient)
	fx.PatientUsrB = newUser("patient.b@test.local", auth.RolePatient)

	newPatient := func(userID, caregiverID, clinicianID uuid.UUID, name, diagnosis string) uuid.UUID {
		diagnosisEnc, err := testCipher.EncryptString(diagnosis)
		if err != nil {
			t.Fatalf("encrypting fixture diagnosis: %v", err)
		}
		allergies := "Penicillin"
		allergiesEnc, err := testCipher.EncryptString(allergies)
		if err != nil {
			t.Fatalf("encrypting fixture allergies: %v", err)
		}

		var id uuid.UUID
		err = testPool.QueryRow(ctx, `
			INSERT INTO patients (user_id, full_name, dob, diagnosis_enc, allergies_enc,
			                      emergency_contact, caregiver_id, clinician_id)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id`,
			userID, name, time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC), diagnosisEnc, allergiesEnc,
			"+254700000000", caregiverID, clinicianID).Scan(&id)
		if err != nil {
			t.Fatalf("creating fixture patient %s: %v", name, err)
		}
		return id
	}

	fx.PatientA = newPatient(fx.PatientUsrA, fx.CaregiverA, fx.ClinicianA, "Patient A", "Diagnosis A — appendectomy")
	fx.PatientB = newPatient(fx.PatientUsrB, fx.CaregiverB, fx.ClinicianB, "Patient B", "Diagnosis B — pneumonia")

	// Template with one critical and one ordinary question, so severity
	// derivation can be exercised end to end.
	const questions = `[
		{"id":"chest_pain","text":"Chest pain?","type":"yes_no","critical":true,"red_flag_if":{"op":"eq","value":"yes"}},
		{"id":"fever","text":"Fever?","type":"yes_no","red_flag_if":{"op":"eq","value":"yes"}},
		{"id":"pain_level","text":"Pain level","type":"scale","scale_min":0,"scale_max":10,"red_flag_if":{"op":"gte","value":8}}
	]`
	if err := testPool.QueryRow(ctx,
		`INSERT INTO checkin_templates (day_offset, name, questions_json) VALUES (2, 'Day 2', $1::jsonb) RETURNING id`,
		questions).Scan(&fx.TemplateID); err != nil {
		t.Fatalf("creating fixture template: %v", err)
	}

	newCheckin := func(patientID uuid.UUID) uuid.UUID {
		var id uuid.UUID
		if err := testPool.QueryRow(ctx,
			`INSERT INTO checkins (patient_id, template_id, scheduled_at) VALUES ($1, $2, now()) RETURNING id`,
			patientID, fx.TemplateID).Scan(&id); err != nil {
			t.Fatalf("creating fixture check-in: %v", err)
		}
		return id
	}
	fx.CheckinA = newCheckin(fx.PatientA)
	fx.CheckinB = newCheckin(fx.PatientB)

	newMedication := func(patientID uuid.UUID, name string) (uuid.UUID, uuid.UUID) {
		var medID, logID uuid.UUID
		if err := testPool.QueryRow(ctx,
			`INSERT INTO medications (patient_id, name, dosage, frequency_hours, start_date)
			 VALUES ($1, $2, '500mg', 8, current_date) RETURNING id`,
			patientID, name).Scan(&medID); err != nil {
			t.Fatalf("creating fixture medication: %v", err)
		}
		if err := testPool.QueryRow(ctx,
			`INSERT INTO medication_logs (medication_id, scheduled_at) VALUES ($1, now()) RETURNING id`,
			medID).Scan(&logID); err != nil {
			t.Fatalf("creating fixture dose log: %v", err)
		}
		return medID, logID
	}
	fx.MedicationA, fx.DoseLogA = newMedication(fx.PatientA, "Amoxicillin")
	fx.MedicationB, fx.DoseLogB = newMedication(fx.PatientB, "Azithromycin")

	newAlert := func(patientID uuid.UUID) uuid.UUID {
		var id uuid.UUID
		if err := testPool.QueryRow(ctx,
			`INSERT INTO alerts (patient_id, severity, message) VALUES ($1, 'high', 'fixture alert') RETURNING id`,
			patientID).Scan(&id); err != nil {
			t.Fatalf("creating fixture alert: %v", err)
		}
		return id
	}
	fx.AlertA = newAlert(fx.PatientA)
	fx.AlertB = newAlert(fx.PatientB)

	// A discharge record on patient A, so discharge reads have something to
	// return and check-in generation has a date to work from.
	summaryEnc, err := testCipher.EncryptString("Fixture discharge summary")
	if err != nil {
		t.Fatalf("encrypting fixture summary: %v", err)
	}
	if _, err := testPool.Exec(ctx,
		`INSERT INTO discharge_records (patient_id, clinician_id, summary_enc, discharge_date)
		 VALUES ($1, $2, $3, current_date - 2)`,
		fx.PatientA, fx.ClinicianA, summaryEnc); err != nil {
		t.Fatalf("creating fixture discharge record: %v", err)
	}

	return fx
}

const testPassword = "AfyaLinkTest2026!"

// token mints an access token directly rather than going through login.
//
// The authorisation tests are about what a *validly authenticated* caller may
// reach; routing them all through the login endpoint would just make every
// test slower and would conflate two different failures. Login has its own
// test.
func (e *env) token(userID uuid.UUID, role auth.Role) string {
	e.t.Helper()

	token, err := testJWT.Issue(userID, role)
	if err != nil {
		e.t.Fatalf("issuing test token: %v", err)
	}
	return token
}

// do performs a request against the router. An empty token sends no
// Authorization header.
func (e *env) do(method, path, token string, body any) *httptest.ResponseRecorder {
	e.t.Helper()

	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			e.t.Fatalf("encoding request body: %v", err)
		}
		reader = bytes.NewReader(encoded)
	}

	req := httptest.NewRequest(method, path, reader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	return rec
}

// decode unmarshals a JSON response body.
func decode(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()

	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decoding response %q: %v", rec.Body.String(), err)
	}
	return out
}

// requireStatus asserts a response code, quoting the body on failure.
func requireStatus(t *testing.T, rec *httptest.ResponseRecorder, want int) {
	t.Helper()

	if rec.Code != want {
		t.Fatalf("status: got %d, want %d (body: %s)", rec.Code, want, rec.Body.String())
	}
}
