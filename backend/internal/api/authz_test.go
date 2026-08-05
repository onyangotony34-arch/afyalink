package api_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/google/uuid"

	"github.com/OderoCeasar/afyalink/backend/internal/auth"
)

// TestUnauthenticatedAccessIsRejected covers the floor: no token, no data.
func TestUnauthenticatedAccessIsRejected(t *testing.T) {
	e := newEnv(t)

	routes := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/patients"},
		{http.MethodPost, "/api/patients"},
		{http.MethodGet, "/api/patients/" + e.fx.PatientA.String()},
		{http.MethodPost, "/api/patients/" + e.fx.PatientA.String() + "/discharge"},
		{http.MethodGet, "/api/patients/" + e.fx.PatientA.String() + "/discharge"},
		{http.MethodPost, "/api/patients/" + e.fx.PatientA.String() + "/medications"},
		{http.MethodGet, "/api/patients/" + e.fx.PatientA.String() + "/medications"},
		{http.MethodGet, "/api/patients/" + e.fx.PatientA.String() + "/checkins"},
		{http.MethodPost, "/api/patients/" + e.fx.PatientA.String() + "/notes"},
		{http.MethodGet, "/api/patients/" + e.fx.PatientA.String() + "/notes"},
		{http.MethodPost, "/api/checkins/" + e.fx.CheckinA.String() + "/respond"},
		{http.MethodGet, "/api/alerts"},
		{http.MethodPatch, "/api/alerts/" + e.fx.AlertA.String() + "/resolve"},
		{http.MethodPost, fmt.Sprintf("/api/medications/%s/logs/%s/taken", e.fx.MedicationA, e.fx.DoseLogA)},
		{http.MethodPost, "/api/auth/register"},
		{http.MethodGet, "/api/auth/me"},
	}

	for _, route := range routes {
		rec := e.do(route.method, route.path, "", nil)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("%s %s without a token: got %d, want 401", route.method, route.path, rec.Code)
		}
	}
}

// TestWrongRoleIsForbidden is the "wrong role" half of spec §9: for every
// patient-data endpoint, a caller holding a role the route does not grant is
// refused.
func TestWrongRoleIsForbidden(t *testing.T) {
	e := newEnv(t)

	patientA := e.fx.PatientA.String()

	cases := []struct {
		name   string
		method string
		path   string
		// actor is a role that must NOT be able to call this route, using an
		// account that legitimately owns the target patient — so the only
		// reason for refusal is the role itself.
		userID uuid.UUID
		role   auth.Role
		body   any
	}{
		{
			name:   "caregiver cannot create patients",
			method: http.MethodPost, path: "/api/patients",
			userID: e.fx.CaregiverA, role: auth.RoleCaregiver,
			body: map[string]any{"user_id": uuid.NewString(), "full_name": "X", "dob": "1990-01-01", "diagnosis": "test"},
		},
		{
			name:   "patient cannot create patients",
			method: http.MethodPost, path: "/api/patients",
			userID: e.fx.PatientUsrA, role: auth.RolePatient,
			body: map[string]any{"user_id": uuid.NewString(), "full_name": "X", "dob": "1990-01-01", "diagnosis": "test"},
		},
		{
			name:   "caregiver cannot write a discharge record",
			method: http.MethodPost, path: "/api/patients/" + patientA + "/discharge",
			userID: e.fx.CaregiverA, role: auth.RoleCaregiver,
			body: map[string]any{"summary": "unauthorised", "discharge_date": "2026-01-01"},
		},
		{
			name:   "patient cannot write their own discharge record",
			method: http.MethodPost, path: "/api/patients/" + patientA + "/discharge",
			userID: e.fx.PatientUsrA, role: auth.RolePatient,
			body: map[string]any{"summary": "unauthorised", "discharge_date": "2026-01-01"},
		},
		{
			name:   "caregiver cannot prescribe medication",
			method: http.MethodPost, path: "/api/patients/" + patientA + "/medications",
			userID: e.fx.CaregiverA, role: auth.RoleCaregiver,
			body: map[string]any{"name": "X", "dosage": "1g", "frequency_hours": 8, "start_date": "2026-01-01"},
		},
		{
			name:   "patient cannot prescribe their own medication",
			method: http.MethodPost, path: "/api/patients/" + patientA + "/medications",
			userID: e.fx.PatientUsrA, role: auth.RolePatient,
			body: map[string]any{"name": "X", "dosage": "1g", "frequency_hours": 8, "start_date": "2026-01-01"},
		},
		{
			name:   "caregiver cannot read clinical notes",
			method: http.MethodGet, path: "/api/patients/" + patientA + "/notes",
			userID: e.fx.CaregiverA, role: auth.RoleCaregiver,
		},
		{
			name:   "patient cannot read clinical notes",
			method: http.MethodGet, path: "/api/patients/" + patientA + "/notes",
			userID: e.fx.PatientUsrA, role: auth.RolePatient,
		},
		{
			name:   "caregiver cannot write clinical notes",
			method: http.MethodPost, path: "/api/patients/" + patientA + "/notes",
			userID: e.fx.CaregiverA, role: auth.RoleCaregiver,
			body: map[string]any{"body": "unauthorised note"},
		},
		{
			name:   "caregiver cannot respond to a check-in",
			method: http.MethodPost, path: "/api/checkins/" + e.fx.CheckinA.String() + "/respond",
			userID: e.fx.CaregiverA, role: auth.RoleCaregiver,
			body: map[string]any{"answers": map[string]string{"chest_pain": "no"}},
		},
		{
			name:   "clinician cannot respond to a check-in on the patient's behalf",
			method: http.MethodPost, path: "/api/checkins/" + e.fx.CheckinA.String() + "/respond",
			userID: e.fx.ClinicianA, role: auth.RoleClinician,
			body: map[string]any{"answers": map[string]string{"chest_pain": "no"}},
		},
		{
			name:   "clinician cannot mark a dose taken for a patient",
			method: http.MethodPost, path: fmt.Sprintf("/api/medications/%s/logs/%s/taken", e.fx.MedicationA, e.fx.DoseLogA),
			userID: e.fx.ClinicianA, role: auth.RoleClinician,
		},
		{
			name:   "caregiver cannot mark a dose taken",
			method: http.MethodPost, path: fmt.Sprintf("/api/medications/%s/logs/%s/taken", e.fx.MedicationA, e.fx.DoseLogA),
			userID: e.fx.CaregiverA, role: auth.RoleCaregiver,
		},
		{
			name:   "caregiver cannot resolve an alert",
			method: http.MethodPatch, path: "/api/alerts/" + e.fx.AlertA.String() + "/resolve",
			userID: e.fx.CaregiverA, role: auth.RoleCaregiver,
		},
		{
			name:   "patient cannot resolve an alert",
			method: http.MethodPatch, path: "/api/alerts/" + e.fx.AlertA.String() + "/resolve",
			userID: e.fx.PatientUsrA, role: auth.RolePatient,
		},
		{
			name:   "patient cannot list the alert inbox",
			method: http.MethodGet, path: "/api/alerts",
			userID: e.fx.PatientUsrA, role: auth.RolePatient,
		},
		{
			name:   "caregiver cannot register accounts",
			method: http.MethodPost, path: "/api/auth/register",
			userID: e.fx.CaregiverA, role: auth.RoleCaregiver,
			body: map[string]any{"email": "x@test.local", "password": "LongEnoughPassword1", "role": "patient"},
		},
		{
			name:   "patient cannot register accounts",
			method: http.MethodPost, path: "/api/auth/register",
			userID: e.fx.PatientUsrA, role: auth.RolePatient,
			body: map[string]any{"email": "y@test.local", "password": "LongEnoughPassword1", "role": "patient"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rec := e.do(c.method, c.path, e.token(c.userID, c.role), c.body)
			requireStatus(t, rec, http.StatusForbidden)
		})
	}
}

// TestWrongOwnerIsForbidden is the "wrong owner" half of spec §9: a caller
// with the correct role for the route, but no relationship to the target
// patient, is refused.
//
// This is the check a route-level RBAC guard cannot make. Every actor below
// would sail through RequireRole.
func TestWrongOwnerIsForbidden(t *testing.T) {
	e := newEnv(t)

	// Everything targets patient A / their resources, while every actor belongs
	// to patient B's care team.
	patientA := e.fx.PatientA.String()

	cases := []struct {
		name   string
		method string
		path   string
		userID uuid.UUID
		role   auth.Role
		body   any
	}{
		{
			name:   "other clinician cannot read the patient",
			method: http.MethodGet, path: "/api/patients/" + patientA,
			userID: e.fx.ClinicianB, role: auth.RoleClinician,
		},
		{
			name:   "unlinked caregiver cannot read the patient",
			method: http.MethodGet, path: "/api/patients/" + patientA,
			userID: e.fx.CaregiverB, role: auth.RoleCaregiver,
		},
		{
			name:   "other patient cannot read the patient",
			method: http.MethodGet, path: "/api/patients/" + patientA,
			userID: e.fx.PatientUsrB, role: auth.RolePatient,
		},
		{
			name:   "other clinician cannot write a discharge record",
			method: http.MethodPost, path: "/api/patients/" + patientA + "/discharge",
			userID: e.fx.ClinicianB, role: auth.RoleClinician,
			body: map[string]any{"summary": "unauthorised", "discharge_date": "2026-01-01"},
		},
		{
			name:   "unlinked caregiver cannot read discharge records",
			method: http.MethodGet, path: "/api/patients/" + patientA + "/discharge",
			userID: e.fx.CaregiverB, role: auth.RoleCaregiver,
		},
		{
			name:   "other clinician cannot prescribe medication",
			method: http.MethodPost, path: "/api/patients/" + patientA + "/medications",
			userID: e.fx.ClinicianB, role: auth.RoleClinician,
			body: map[string]any{"name": "X", "dosage": "1g", "frequency_hours": 8, "start_date": "2026-01-01"},
		},
		{
			name:   "unlinked caregiver cannot list medications",
			method: http.MethodGet, path: "/api/patients/" + patientA + "/medications",
			userID: e.fx.CaregiverB, role: auth.RoleCaregiver,
		},
		{
			name:   "other patient cannot list medications",
			method: http.MethodGet, path: "/api/patients/" + patientA + "/medications",
			userID: e.fx.PatientUsrB, role: auth.RolePatient,
		},
		{
			name:   "unlinked caregiver cannot list check-ins",
			method: http.MethodGet, path: "/api/patients/" + patientA + "/checkins",
			userID: e.fx.CaregiverB, role: auth.RoleCaregiver,
		},
		{
			name:   "other clinician cannot read clinical notes",
			method: http.MethodGet, path: "/api/patients/" + patientA + "/notes",
			userID: e.fx.ClinicianB, role: auth.RoleClinician,
		},
		{
			name:   "other clinician cannot write clinical notes",
			method: http.MethodPost, path: "/api/patients/" + patientA + "/notes",
			userID: e.fx.ClinicianB, role: auth.RoleClinician,
			body: map[string]any{"body": "unauthorised note"},
		},
		{
			name:   "other patient cannot answer this patient's check-in",
			method: http.MethodPost, path: "/api/checkins/" + e.fx.CheckinA.String() + "/respond",
			userID: e.fx.PatientUsrB, role: auth.RolePatient,
			body: map[string]any{"answers": map[string]string{"chest_pain": "no", "fever": "no", "pain_level": "1"}},
		},
		{
			name:   "other patient cannot mark this patient's dose taken",
			method: http.MethodPost, path: fmt.Sprintf("/api/medications/%s/logs/%s/taken", e.fx.MedicationA, e.fx.DoseLogA),
			userID: e.fx.PatientUsrB, role: auth.RolePatient,
		},
		{
			name:   "other clinician cannot resolve the alert",
			method: http.MethodPatch, path: "/api/alerts/" + e.fx.AlertA.String() + "/resolve",
			userID: e.fx.ClinicianB, role: auth.RoleClinician,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rec := e.do(c.method, c.path, e.token(c.userID, c.role), c.body)
			requireStatus(t, rec, http.StatusForbidden)
		})
	}
}

// A patient's own dose log must not be reachable by pairing it with a
// medication the caller does own. Without the medication_id predicate in the
// ownership join, this would succeed.
func TestDoseLogCannotBeClaimedViaAnotherMedication(t *testing.T) {
	e := newEnv(t)

	path := fmt.Sprintf("/api/medications/%s/logs/%s/taken", e.fx.MedicationA, e.fx.DoseLogB)

	rec := e.do(http.MethodPost, path, e.token(e.fx.PatientUsrA, auth.RolePatient), nil)
	requireStatus(t, rec, http.StatusForbidden)
}

// A nonexistent patient and an inaccessible one must be indistinguishable,
// otherwise the UUID space becomes an enumeration oracle over the roster.
func TestMissingAndForbiddenAreIndistinguishable(t *testing.T) {
	e := newEnv(t)

	token := e.token(e.fx.CaregiverB, auth.RoleCaregiver)

	forbidden := e.do(http.MethodGet, "/api/patients/"+e.fx.PatientA.String(), token, nil)
	missing := e.do(http.MethodGet, "/api/patients/"+uuid.NewString(), token, nil)

	if forbidden.Code != missing.Code {
		t.Fatalf("status differs: existing-but-forbidden %d, nonexistent %d", forbidden.Code, missing.Code)
	}
	if forbidden.Body.String() != missing.Body.String() {
		t.Fatalf("body differs:\n  existing-but-forbidden: %s\n  nonexistent:            %s",
			forbidden.Body.String(), missing.Body.String())
	}
}

// Collection endpoints must scope by role rather than returning everything.
func TestCollectionsAreScopedToTheCaller(t *testing.T) {
	e := newEnv(t)

	t.Run("clinician sees only their own patients", func(t *testing.T) {
		rec := e.do(http.MethodGet, "/api/patients", e.token(e.fx.ClinicianA, auth.RoleClinician), nil)
		requireStatus(t, rec, http.StatusOK)

		list := decode(t, rec)["patients"].([]any)
		if len(list) != 1 {
			t.Fatalf("got %d patients, want 1", len(list))
		}
		if got := list[0].(map[string]any)["id"].(string); got != e.fx.PatientA.String() {
			t.Fatalf("got patient %s, want %s", got, e.fx.PatientA)
		}
	})

	t.Run("caregiver sees only their linked patient", func(t *testing.T) {
		rec := e.do(http.MethodGet, "/api/patients", e.token(e.fx.CaregiverB, auth.RoleCaregiver), nil)
		requireStatus(t, rec, http.StatusOK)

		list := decode(t, rec)["patients"].([]any)
		if len(list) != 1 {
			t.Fatalf("got %d patients, want 1", len(list))
		}
		if got := list[0].(map[string]any)["id"].(string); got != e.fx.PatientB.String() {
			t.Fatalf("caregiver B got patient %s, want %s", got, e.fx.PatientB)
		}
	})

	t.Run("alert inbox is scoped to the caller's patients", func(t *testing.T) {
		rec := e.do(http.MethodGet, "/api/alerts", e.token(e.fx.ClinicianA, auth.RoleClinician), nil)
		requireStatus(t, rec, http.StatusOK)

		list := decode(t, rec)["alerts"].([]any)
		if len(list) != 1 {
			t.Fatalf("got %d alerts, want 1", len(list))
		}
		if got := list[0].(map[string]any)["patient_id"].(string); got != e.fx.PatientA.String() {
			t.Fatalf("clinician A saw an alert for patient %s", got)
		}
	})
}

// Field-level policy: the diagnosis narrative is decrypted for the clinician
// and the patient, and withheld from the caregiver.
func TestDiagnosisVisibilityByRole(t *testing.T) {
	e := newEnv(t)

	path := "/api/patients/" + e.fx.PatientA.String()

	cases := []struct {
		name        string
		userID      uuid.UUID
		role        auth.Role
		wantVisible bool
	}{
		{"clinician sees the diagnosis", e.fx.ClinicianA, auth.RoleClinician, true},
		{"patient sees their own diagnosis", e.fx.PatientUsrA, auth.RolePatient, true},
		{"caregiver does not see the diagnosis", e.fx.CaregiverA, auth.RoleCaregiver, false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rec := e.do(http.MethodGet, path, e.token(c.userID, c.role), nil)
			requireStatus(t, rec, http.StatusOK)

			patient := decode(t, rec)["patient"].(map[string]any)
			_, present := patient["diagnosis"]

			if present != c.wantVisible {
				t.Fatalf("diagnosis present=%v, want %v (body: %s)", present, c.wantVisible, rec.Body.String())
			}

			// Allergies reach every authorised role — a caregiver acts on them.
			if _, ok := patient["allergies"]; !ok {
				t.Error("allergies should be visible to every authorised role")
			}
		})
	}
}
