package api_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/OderoCeasar/afyalink/backend/internal/api"
	"github.com/OderoCeasar/afyalink/backend/internal/auth"
	"github.com/OderoCeasar/afyalink/backend/internal/scheduler"
)

// TestLoginRotatesAndRevokesRefreshTokens walks the whole session lifecycle.
func TestLoginRotatesAndRevokesRefreshTokens(t *testing.T) {
	e := newEnv(t)

	login := e.do(http.MethodPost, "/api/auth/login", "", map[string]any{
		"email":    "clinician.a@test.local",
		"password": testPassword,
	})
	requireStatus(t, login, http.StatusOK)

	body := decode(t, login)
	if body["access_token"].(string) == "" {
		t.Fatal("login returned an empty access token")
	}

	// The refresh token must travel only in the HttpOnly cookie, never the body.
	if strings.Contains(login.Body.String(), "refresh_token") {
		t.Fatalf("login response body mentions a refresh token: %s", login.Body.String())
	}

	cookie := refreshCookie(t, login)
	if !cookie.HttpOnly {
		t.Error("refresh cookie is not HttpOnly")
	}
	if cookie.SameSite != http.SameSiteStrictMode {
		t.Error("refresh cookie is not SameSite=Strict, which is what makes a separate CSRF token unnecessary")
	}

	// Refresh rotates: the old cookie value must stop working.
	refresh := e.doWithCookie(http.MethodPost, "/api/auth/refresh", cookie.Value, nil)
	requireStatus(t, refresh, http.StatusOK)

	rotated := refreshCookie(t, refresh)
	if rotated.Value == cookie.Value {
		t.Fatal("refresh did not rotate the token")
	}

	// Replaying the superseded token must fail AND revoke the whole family.
	replay := e.doWithCookie(http.MethodPost, "/api/auth/refresh", cookie.Value, nil)
	requireStatus(t, replay, http.StatusUnauthorized)

	afterReplay := e.doWithCookie(http.MethodPost, "/api/auth/refresh", rotated.Value, nil)
	if afterReplay.Code != http.StatusUnauthorized {
		t.Fatalf("after a detected replay the rotated token should also be revoked, got %d", afterReplay.Code)
	}
}

func TestLoginFailsIdenticallyForUnknownEmailAndWrongPassword(t *testing.T) {
	e := newEnv(t)

	wrongPassword := e.do(http.MethodPost, "/api/auth/login", "", map[string]any{
		"email":    "clinician.a@test.local",
		"password": "definitely-not-the-password",
	})
	unknownEmail := e.do(http.MethodPost, "/api/auth/login", "", map[string]any{
		"email":    "nobody@test.local",
		"password": testPassword,
	})

	requireStatus(t, wrongPassword, http.StatusUnauthorized)
	requireStatus(t, unknownEmail, http.StatusUnauthorized)

	if wrongPassword.Body.String() != unknownEmail.Body.String() {
		t.Fatalf("login responses differ and leak account existence:\n  wrong password: %s\n  unknown email:  %s",
			wrongPassword.Body.String(), unknownEmail.Body.String())
	}
}

// TestFullRecoveryLoop is the spec §9 acceptance path, start to finish.
func TestFullRecoveryLoop(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()

	clinicianToken := e.token(e.fx.ClinicianA, auth.RoleClinician)

	// 1. The clinician bootstraps a patient account and a caregiver account.
	patientEmail := "loop.patient@test.local"
	created := e.do(http.MethodPost, "/api/auth/register", clinicianToken, map[string]any{
		"email":    patientEmail,
		"password": "LoopPatientPassword1",
		"role":     "patient",
	})
	requireStatus(t, created, http.StatusCreated)
	patientUserID := decode(t, created)["user"].(map[string]any)["id"].(string)

	caregiverCreated := e.do(http.MethodPost, "/api/auth/register", clinicianToken, map[string]any{
		"email":    "loop.caregiver@test.local",
		"password": "LoopCaregiverPassword1",
		"role":     "caregiver",
	})
	requireStatus(t, caregiverCreated, http.StatusCreated)
	caregiverUserID := decode(t, caregiverCreated)["user"].(map[string]any)["id"].(string)

	// 2. The clinician creates the patient record.
	patientCreated := e.do(http.MethodPost, "/api/patients", clinicianToken, map[string]any{
		"user_id":           patientUserID,
		"full_name":         "Amina Wanjiru",
		"dob":               "1992-06-15",
		"diagnosis":         "Post-operative recovery following caesarean section",
		"allergies":         "Ibuprofen",
		"emergency_contact": "+254733111222",
		"caregiver_id":      caregiverUserID,
	})
	requireStatus(t, patientCreated, http.StatusCreated)
	patientID := decode(t, patientCreated)["patient"].(map[string]any)["id"].(string)

	// 3. Discharge, backdated so the Day 2 check-in is already due.
	dischargeDate := time.Now().AddDate(0, 0, -2).Format("2006-01-02")
	discharge := e.do(http.MethodPost, "/api/patients/"+patientID+"/discharge", clinicianToken, map[string]any{
		"summary":        "Caesarean section without complication. Wound reviewed, mother and baby stable.",
		"discharge_date": dischargeDate,
	})
	requireStatus(t, discharge, http.StatusCreated)

	// 4. Prescribe a medication.
	medication := e.do(http.MethodPost, "/api/patients/"+patientID+"/medications", clinicianToken, map[string]any{
		"name":            "Amoxicillin",
		"dosage":          "500mg",
		"frequency_hours": 8,
		"start_date":      time.Now().Format("2006-01-02"),
	})
	requireStatus(t, medication, http.StatusCreated)

	// 5. The scheduler generates the Day 2 / 5 / 10 check-ins and dose logs.
	worker := scheduler.New(testPool, slog.New(slog.NewTextHandler(io.Discard, nil)), time.Minute)
	if _, err := worker.RunOnce(ctx); err != nil {
		t.Fatalf("scheduler pass: %v", err)
	}

	patientToken := e.token(uuid.MustParse(patientUserID), auth.RolePatient)

	checkinsResp := e.do(http.MethodGet, "/api/patients/"+patientID+"/checkins", patientToken, nil)
	requireStatus(t, checkinsResp, http.StatusOK)

	checkinList := decode(t, checkinsResp)["checkins"].([]any)
	if len(checkinList) != 1 {
		// The fixture template set only contains a Day 2 template.
		t.Fatalf("got %d generated check-ins, want 1 (day 2)", len(checkinList))
	}

	dayTwo := checkinList[0].(map[string]any)
	if got := int(dayTwo["day_offset"].(float64)); got != 2 {
		t.Fatalf("generated check-in day_offset: got %d, want 2", got)
	}
	checkinID := dayTwo["id"].(string)

	// The rule thresholds must not be exposed to the client.
	if strings.Contains(checkinsResp.Body.String(), "red_flag_if") {
		t.Fatal("check-in payload leaks the red-flag rules to the client")
	}

	// 6. The patient answers with a critical red flag.
	responded := e.do(http.MethodPost, "/api/checkins/"+checkinID+"/respond", patientToken, map[string]any{
		"answers": map[string]string{
			"chest_pain": "yes",
			"fever":      "no",
			"pain_level": "3",
		},
	})
	requireStatus(t, responded, http.StatusOK)

	respondBody := decode(t, responded)
	alertInfo, ok := respondBody["alert"].(map[string]any)
	if !ok {
		t.Fatalf("a chest-pain red flag did not raise an alert: %s", responded.Body.String())
	}
	if alertInfo["severity"].(string) != "critical" {
		t.Fatalf("severity: got %v, want critical (chest_pain is a designated critical question)", alertInfo["severity"])
	}

	// 7. The alert is on the clinician's inbox in the same request cycle.
	clinicianAlerts := e.do(http.MethodGet, "/api/alerts", clinicianToken, nil)
	requireStatus(t, clinicianAlerts, http.StatusOK)

	alertID := findAlertForPatient(t, clinicianAlerts, patientID)
	if alertID == "" {
		t.Fatalf("the clinician's inbox does not contain the new alert: %s", clinicianAlerts.Body.String())
	}

	// Critical must sort ahead of the pre-existing high-severity fixture alert.
	firstAlert := decode(t, clinicianAlerts)["alerts"].([]any)[0].(map[string]any)
	if firstAlert["severity"].(string) != "critical" {
		t.Fatalf("alert inbox is not sorted with the most severe first, got %v", firstAlert["severity"])
	}

	// 8. The linked caregiver sees it; an unlinked caregiver does not.
	caregiverToken := e.token(uuid.MustParse(caregiverUserID), auth.RoleCaregiver)
	caregiverAlerts := e.do(http.MethodGet, "/api/alerts", caregiverToken, nil)
	requireStatus(t, caregiverAlerts, http.StatusOK)

	if findAlertForPatient(t, caregiverAlerts, patientID) == "" {
		t.Fatal("the linked caregiver cannot see their patient's alert")
	}

	strangerAlerts := e.do(http.MethodGet, "/api/alerts", e.token(e.fx.CaregiverB, auth.RoleCaregiver), nil)
	requireStatus(t, strangerAlerts, http.StatusOK)
	if findAlertForPatient(t, strangerAlerts, patientID) != "" {
		t.Fatal("an unlinked caregiver can see another patient's alert")
	}

	// 9. Answering twice is refused rather than raising a duplicate alert.
	repeat := e.do(http.MethodPost, "/api/checkins/"+checkinID+"/respond", patientToken, map[string]any{
		"answers": map[string]string{"chest_pain": "yes", "fever": "yes", "pain_level": "9"},
	})
	requireStatus(t, repeat, http.StatusConflict)

	// 10. The clinician resolves the alert; resolving twice conflicts.
	resolved := e.do(http.MethodPatch, "/api/alerts/"+alertID+"/resolve", clinicianToken, nil)
	requireStatus(t, resolved, http.StatusOK)

	if !decode(t, resolved)["alert"].(map[string]any)["resolved"].(bool) {
		t.Fatal("alert not marked resolved")
	}

	again := e.do(http.MethodPatch, "/api/alerts/"+alertID+"/resolve", clinicianToken, nil)
	requireStatus(t, again, http.StatusConflict)
}

// A missed dose must raise a low-severity alert — spec §9.
func TestMissedDoseRaisesLowSeverityAlert(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()

	// Backdate the fixture dose beyond the grace period.
	if _, err := testPool.Exec(ctx,
		`UPDATE medication_logs SET scheduled_at = now() - interval '6 hours' WHERE id = $1`,
		e.fx.DoseLogA); err != nil {
		t.Fatalf("backdating dose: %v", err)
	}

	worker := scheduler.New(testPool, slog.New(slog.NewTextHandler(io.Discard, nil)), time.Minute)
	result, err := worker.RunOnce(ctx)
	if err != nil {
		t.Fatalf("scheduler pass: %v", err)
	}
	if result.DosesMissed == 0 {
		t.Fatal("the overdue dose was not marked missed")
	}
	if result.MissedDoseAlerts == 0 {
		t.Fatal("no alert was raised for the missed dose")
	}

	var severity, message string
	if err := testPool.QueryRow(ctx,
		`SELECT severity, message FROM alerts
		 WHERE patient_id = $1 AND severity = 'low' ORDER BY created_at DESC LIMIT 1`,
		e.fx.PatientA).Scan(&severity, &message); err != nil {
		t.Fatalf("querying the missed-dose alert: %v", err)
	}
	if severity != "low" {
		t.Fatalf("severity: got %q, want low", severity)
	}
	if !strings.Contains(message, "Amoxicillin") {
		t.Fatalf("alert message should name the medication, got %q", message)
	}

	// A second sweep must not re-alert for the same dose.
	before := countAlerts(t, e.fx.PatientA)
	if _, err := worker.RunOnce(ctx); err != nil {
		t.Fatalf("second scheduler pass: %v", err)
	}
	if after := countAlerts(t, e.fx.PatientA); after != before {
		t.Fatalf("a repeated sweep duplicated alerts: %d -> %d", before, after)
	}
}

// Generation must be idempotent — the scheduler re-runs constantly.
func TestSchedulerGenerationIsIdempotent(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()

	// The fixtures pre-create a check-in per patient, which the unique
	// (patient_id, template_id) index would make the first pass skip. Clear
	// them so generation has real work to do and the idempotency claim is
	// actually exercised rather than trivially true.
	if _, err := testPool.Exec(ctx, `DELETE FROM checkin_responses`); err != nil {
		t.Fatalf("clearing responses: %v", err)
	}
	if _, err := testPool.Exec(ctx, `UPDATE alerts SET checkin_id = NULL`); err != nil {
		t.Fatalf("detaching alerts: %v", err)
	}
	if _, err := testPool.Exec(ctx, `DELETE FROM checkins`); err != nil {
		t.Fatalf("clearing check-ins: %v", err)
	}

	worker := scheduler.New(testPool, slog.New(slog.NewTextHandler(io.Discard, nil)), time.Minute)

	first, err := worker.RunOnce(ctx)
	if err != nil {
		t.Fatalf("first pass: %v", err)
	}
	// Only patient A has a discharge record, so exactly one check-in is due.
	if first.CheckinsCreated != 1 {
		t.Fatalf("first pass created %d check-ins, want 1", first.CheckinsCreated)
	}

	second, err := worker.RunOnce(ctx)
	if err != nil {
		t.Fatalf("second pass: %v", err)
	}
	if second.CheckinsCreated != 0 {
		t.Fatalf("second pass created %d duplicate check-ins", second.CheckinsCreated)
	}
	if second.DoseLogsCreated != 0 {
		t.Fatalf("second pass created %d duplicate dose logs", second.DoseLogsCreated)
	}

	_ = e
}

// Every read and write of patient-scoped data must leave an audit row.
func TestAuditLogRecordsPatientDataAccess(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()

	token := e.token(e.fx.ClinicianA, auth.RoleClinician)

	requireStatus(t, e.do(http.MethodGet, "/api/patients/"+e.fx.PatientA.String(), token, nil), http.StatusOK)

	var count int
	err := testPool.QueryRow(ctx,
		`SELECT count(*) FROM audit_logs
		 WHERE user_id = $1 AND action = 'read' AND resource = 'patient' AND resource_id = $2`,
		e.fx.ClinicianA, e.fx.PatientA).Scan(&count)
	if err != nil {
		t.Fatalf("querying audit log: %v", err)
	}
	if count != 1 {
		t.Fatalf("got %d audit rows for the patient read, want 1", count)
	}

	// A denied access must not be recorded as an actual access.
	requireStatus(t, e.do(http.MethodGet, "/api/patients/"+e.fx.PatientB.String(), token, nil), http.StatusForbidden)

	if err := testPool.QueryRow(ctx,
		`SELECT count(*) FROM audit_logs WHERE resource_id = $1`, e.fx.PatientB).Scan(&count); err != nil {
		t.Fatalf("querying audit log: %v", err)
	}
	if count != 0 {
		t.Fatalf("a forbidden read wrote %d audit rows; it should write none", count)
	}
}

// Unknown fields must be rejected rather than silently ignored.
func TestUnknownFieldsAreRejected(t *testing.T) {
	e := newEnv(t)

	rec := e.do(http.MethodPost, "/api/patients", e.token(e.fx.ClinicianA, auth.RoleClinician), map[string]any{
		"user_id":      uuid.NewString(),
		"full_name":    "Test Patient",
		"dob":          "1990-01-01",
		"diagnosis":    "test",
		"clinician_id": uuid.NewString(), // not a settable field — assignment comes from the token
	})
	requireStatus(t, rec, http.StatusBadRequest)

	if !strings.Contains(rec.Body.String(), "unknown field") {
		t.Fatalf("expected an unknown-field error, got %s", rec.Body.String())
	}
}

// The login endpoint must be rate limited.
func TestLoginIsRateLimited(t *testing.T) {
	e := newEnv(t)

	limited := *testConfig
	limited.LoginRateLimit = 1
	limited.LoginRateBurst = 3

	router, err := api.New(api.Deps{
		Config: &limited,
		Pool:   testPool,
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		Cipher: testCipher,
	})
	if err != nil {
		t.Fatalf("building rate-limited router: %v", err)
	}

	strict := &env{t: t, router: router, fx: e.fx}

	var sawTooMany bool
	for i := 0; i < 10; i++ {
		rec := strict.do(http.MethodPost, "/api/auth/login", "", map[string]any{
			"email":    "clinician.a@test.local",
			"password": "wrong-password",
		})
		if rec.Code == http.StatusTooManyRequests {
			sawTooMany = true
			break
		}
	}

	if !sawTooMany {
		t.Fatal("ten rapid login attempts were never rate limited")
	}
}

// CORS must reflect only the configured origin.
func TestCORSRejectsUnknownOrigins(t *testing.T) {
	e := newEnv(t)

	allowed := httptest.NewRequest(http.MethodOptions, "/api/auth/login", nil)
	allowed.Header.Set("Origin", "http://localhost:5173")
	allowedRec := httptest.NewRecorder()
	e.router.ServeHTTP(allowedRec, allowed)

	if got := allowedRec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
		t.Fatalf("allowed origin: got %q, want the configured origin", got)
	}

	denied := httptest.NewRequest(http.MethodOptions, "/api/auth/login", nil)
	denied.Header.Set("Origin", "https://evil.example")
	deniedRec := httptest.NewRecorder()
	e.router.ServeHTTP(deniedRec, denied)

	if got := deniedRec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("an unlisted origin was allowed: %q", got)
	}
}

// --- helpers ---

// doWithCookie issues a request carrying the refresh cookie.
func (e *env) doWithCookie(method, path, cookieValue string, body any) *httptest.ResponseRecorder {
	e.t.Helper()

	req := httptest.NewRequest(method, path, nil)
	req.AddCookie(&http.Cookie{Name: "afyalink_refresh", Value: cookieValue})

	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	return rec
}

func refreshCookie(t *testing.T, rec *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()

	for _, cookie := range rec.Result().Cookies() {
		if cookie.Name == "afyalink_refresh" && cookie.Value != "" {
			return cookie
		}
	}
	t.Fatalf("no refresh cookie in response headers: %v", rec.Header())
	return nil
}

// findAlertForPatient returns the id of the first alert for a patient, or "".
func findAlertForPatient(t *testing.T, rec *httptest.ResponseRecorder, patientID string) string {
	t.Helper()

	for _, item := range decode(t, rec)["alerts"].([]any) {
		alert := item.(map[string]any)
		if alert["patient_id"].(string) == patientID {
			return alert["id"].(string)
		}
	}
	return ""
}

func countAlerts(t *testing.T, patientID uuid.UUID) int {
	t.Helper()

	var count int
	if err := testPool.QueryRow(context.Background(),
		`SELECT count(*) FROM alerts WHERE patient_id = $1`, patientID).Scan(&count); err != nil {
		t.Fatalf("counting alerts: %v", err)
	}
	return count
}
