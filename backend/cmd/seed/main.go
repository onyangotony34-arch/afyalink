// Command seed populates a development database with the check-in templates
// and a demo clinician, two demo patients, and a caregiver.
//
// It is idempotent: re-running it will not duplicate templates or accounts.
// It refuses to run against APP_ENV=production, because it installs accounts
// with published passwords.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/OderoCeasar/afyalink/backend/internal/auth"
	"github.com/OderoCeasar/afyalink/backend/internal/config"
	"github.com/OderoCeasar/afyalink/backend/internal/crypto"
	"github.com/OderoCeasar/afyalink/backend/internal/db"
)

// demoPassword is shared by every seeded account. It satisfies the 12
// character minimum the register endpoint enforces.
const demoPassword = "AfyaLinkDemo2026!"

// checkinTemplates are the Day 2 / 5 / 10 post-discharge questionnaires
// required by spec §9.
//
// The red_flag_if rules encode clinical escalation: chest pain and breathing
// difficulty are marked critical because either alone warrants an immediate
// call, whereas pain scores and missed medication are graded by count.
var checkinTemplates = []struct {
	DayOffset int
	Name      string
	Questions string
}{
	{
		DayOffset: 2,
		Name:      "Day 2 post-discharge check-in",
		Questions: `[
			{"id":"chest_pain","text":"Are you experiencing chest pain or tightness?","type":"yes_no","critical":true,"red_flag_if":{"op":"eq","value":"yes"}},
			{"id":"breathing","text":"Are you having difficulty breathing?","type":"yes_no","critical":true,"red_flag_if":{"op":"eq","value":"yes"}},
			{"id":"fever","text":"Have you had a fever in the last 24 hours?","type":"yes_no","red_flag_if":{"op":"eq","value":"yes"}},
			{"id":"wound","text":"Is your wound bleeding or leaking fluid?","type":"yes_no","red_flag_if":{"op":"eq","value":"yes"}},
			{"id":"pain_level","text":"How would you rate your pain right now?","type":"scale","scale_min":0,"scale_max":10,"red_flag_if":{"op":"gte","value":8}},
			{"id":"meds_taken","text":"Have you been able to take all your prescribed medication?","type":"yes_no","red_flag_if":{"op":"eq","value":"no"}}
		]`,
	},
	{
		DayOffset: 5,
		Name:      "Day 5 post-discharge check-in",
		Questions: `[
			{"id":"chest_pain","text":"Are you experiencing chest pain or tightness?","type":"yes_no","critical":true,"red_flag_if":{"op":"eq","value":"yes"}},
			{"id":"fever","text":"Have you had a fever in the last 24 hours?","type":"yes_no","red_flag_if":{"op":"eq","value":"yes"}},
			{"id":"wound_infection","text":"Is the wound area red, swollen, or producing pus?","type":"yes_no","red_flag_if":{"op":"eq","value":"yes"}},
			{"id":"pain_level","text":"How would you rate your pain right now?","type":"scale","scale_min":0,"scale_max":10,"red_flag_if":{"op":"gte","value":7}},
			{"id":"eating","text":"Are you able to eat and drink normally?","type":"yes_no","red_flag_if":{"op":"eq","value":"no"}},
			{"id":"meds_taken","text":"Have you been able to take all your prescribed medication?","type":"yes_no","red_flag_if":{"op":"eq","value":"no"}}
		]`,
	},
	{
		DayOffset: 10,
		Name:      "Day 10 post-discharge check-in",
		Questions: `[
			{"id":"new_symptoms","text":"Have you developed any new symptoms since your last check-in?","type":"yes_no","red_flag_if":{"op":"eq","value":"yes"}},
			{"id":"pain_level","text":"How would you rate your pain right now?","type":"scale","scale_min":0,"scale_max":10,"red_flag_if":{"op":"gte","value":6}},
			{"id":"mobility","text":"Are you able to move around as well as your clinician expected?","type":"yes_no","red_flag_if":{"op":"eq","value":"no"}},
			{"id":"meds_taken","text":"Have you been able to take all your prescribed medication?","type":"yes_no","red_flag_if":{"op":"eq","value":"no"}}
		]`,
	},
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "seed failed: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	// Guard rail: these accounts have a password committed to the repository.
	if cfg.IsProduction() {
		return errors.New("refusing to seed demo accounts with APP_ENV=production")
	}

	ctx := context.Background()

	if err := db.Migrate(cfg.DatabaseURL); err != nil {
		return err
	}

	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	cipher, err := crypto.New(cfg.PHIKey)
	if err != nil {
		return err
	}

	if err := seedTemplates(ctx, pool); err != nil {
		return err
	}

	clinicianID, err := upsertUser(ctx, pool, "clinician@afyalink.test", "+254700000001", auth.RoleClinician)
	if err != nil {
		return err
	}

	caregiverID, err := upsertUser(ctx, pool, "caregiver@afyalink.test", "+254700000002", auth.RoleCaregiver)
	if err != nil {
		return err
	}

	// A second caregiver with no patient link exists purely so the negative
	// authorisation tests have a caregiver who legitimately owns nothing.
	if _, err := upsertUser(ctx, pool, "caregiver2@afyalink.test", "+254700000005", auth.RoleCaregiver); err != nil {
		return err
	}

	patientUser1, err := upsertUser(ctx, pool, "patient1@afyalink.test", "+254700000003", auth.RolePatient)
	if err != nil {
		return err
	}

	patientUser2, err := upsertUser(ctx, pool, "patient2@afyalink.test", "+254700000004", auth.RolePatient)
	if err != nil {
		return err
	}

	// Discharged two days ago, so the Day 2 check-in falls due today and is
	// still inside its grace window (answerable), while Day 5 and Day 10 are
	// upcoming. That is the state the demo and the acceptance pass need.
	dischargeDate := time.Now().AddDate(0, 0, -2)

	// Prescriptions start today rather than at discharge. Backdating them
	// would have the scheduler immediately mark two days of un-recorded doses
	// as missed and bury the demo clinician's inbox under dozens of
	// low-severity alerts — correct behaviour, but useless as a starting
	// state. Starting today still leaves the morning doses overdue, so
	// missed-dose alerting is visible without drowning everything else.
	medStart := time.Now()

	patient1, err := upsertPatient(ctx, pool, cipher, patientArgs{
		UserID:           patientUser1,
		FullName:         "Jane Otieno",
		DOB:              time.Date(1988, time.March, 14, 0, 0, 0, 0, time.UTC),
		Diagnosis:        "Post-operative recovery following emergency appendectomy",
		Allergies:        strptr("Penicillin"),
		EmergencyContact: strptr("+254711234567"),
		CaregiverID:      &caregiverID,
		ClinicianID:      clinicianID,
	})
	if err != nil {
		return err
	}

	// Deliberately has no caregiver: proves a caregiver linked to patient 1
	// cannot reach patient 2.
	patient2, err := upsertPatient(ctx, pool, cipher, patientArgs{
		UserID:           patientUser2,
		FullName:         "Samuel Kiprop",
		DOB:              time.Date(1975, time.November, 2, 0, 0, 0, 0, time.UTC),
		Diagnosis:        "Community-acquired pneumonia, managed with IV antibiotics",
		Allergies:        nil,
		EmergencyContact: strptr("+254722987654"),
		CaregiverID:      nil,
		ClinicianID:      clinicianID,
	})
	if err != nil {
		return err
	}

	if err := upsertDischarge(ctx, pool, cipher, patient1, clinicianID, dischargeDate,
		"Appendectomy performed without complication. Wound dressed; review in two weeks. Advise rest, no heavy lifting for 14 days."); err != nil {
		return err
	}
	if err := upsertDischarge(ctx, pool, cipher, patient2, clinicianID, dischargeDate,
		"Completed five-day IV antibiotic course. Chest clear on discharge. Continue oral antibiotics and complete the full course."); err != nil {
		return err
	}

	if err := upsertMedication(ctx, pool, patient1, "Amoxicillin", "500mg", 8, medStart, medStart.AddDate(0, 0, 7)); err != nil {
		return err
	}
	if err := upsertMedication(ctx, pool, patient1, "Paracetamol", "1g", 6, medStart, medStart.AddDate(0, 0, 5)); err != nil {
		return err
	}
	if err := upsertMedication(ctx, pool, patient2, "Azithromycin", "250mg", 24, medStart, medStart.AddDate(0, 0, 5)); err != nil {
		return err
	}

	fmt.Println("Seed complete. Demo accounts (password for all:", demoPassword+")")
	fmt.Println()
	fmt.Println("  clinician@afyalink.test   clinician — owns both demo patients")
	fmt.Println("  caregiver@afyalink.test   caregiver — linked to Jane Otieno only")
	fmt.Println("  caregiver2@afyalink.test  caregiver — linked to nobody")
	fmt.Println("  patient1@afyalink.test    patient   — Jane Otieno")
	fmt.Println("  patient2@afyalink.test    patient   — Samuel Kiprop")
	fmt.Println()
	fmt.Println("Run the scheduler once to generate check-ins and doses:")
	fmt.Println("  go run ./cmd/scheduler -once")

	return nil
}

func seedTemplates(ctx context.Context, pool *db.Pool) error {
	const query = `
		INSERT INTO checkin_templates (day_offset, name, questions_json)
		VALUES ($1, $2, $3::jsonb)
		ON CONFLICT (day_offset) DO UPDATE
		SET name = EXCLUDED.name, questions_json = EXCLUDED.questions_json`

	for _, t := range checkinTemplates {
		if _, err := pool.Exec(ctx, query, t.DayOffset, t.Name, t.Questions); err != nil {
			return fmt.Errorf("seeding template day %d: %w", t.DayOffset, err)
		}
	}
	return nil
}

// upsertUser creates a user if the email is free, and returns the id either
// way so re-seeding is a no-op.
func upsertUser(ctx context.Context, pool *db.Pool, email, phone string, role auth.Role) (uuid.UUID, error) {
	var existing uuid.UUID
	err := pool.QueryRow(ctx, `SELECT id FROM users WHERE lower(email) = lower($1)`, email).Scan(&existing)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return uuid.UUID{}, fmt.Errorf("looking up %s: %w", email, err)
	}

	hash, err := auth.HashPassword(demoPassword)
	if err != nil {
		return uuid.UUID{}, err
	}

	var id uuid.UUID
	err = pool.QueryRow(ctx,
		`INSERT INTO users (email, phone, password_hash, role) VALUES (lower($1), $2, $3, $4) RETURNING id`,
		email, phone, hash, string(role),
	).Scan(&id)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("creating %s: %w", email, err)
	}
	return id, nil
}

type patientArgs struct {
	UserID           uuid.UUID
	FullName         string
	DOB              time.Time
	Diagnosis        string
	Allergies        *string
	EmergencyContact *string
	CaregiverID      *uuid.UUID
	ClinicianID      uuid.UUID
}

func upsertPatient(ctx context.Context, pool *db.Pool, cipher *crypto.Cipher, args patientArgs) (uuid.UUID, error) {
	var existing uuid.UUID
	err := pool.QueryRow(ctx, `SELECT id FROM patients WHERE user_id = $1`, args.UserID).Scan(&existing)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return uuid.UUID{}, fmt.Errorf("looking up patient: %w", err)
	}

	diagnosisEnc, err := cipher.EncryptString(args.Diagnosis)
	if err != nil {
		return uuid.UUID{}, err
	}
	allergiesEnc, err := cipher.EncryptNullable(args.Allergies)
	if err != nil {
		return uuid.UUID{}, err
	}

	var id uuid.UUID
	err = pool.QueryRow(ctx, `
		INSERT INTO patients (user_id, full_name, dob, diagnosis_enc, allergies_enc,
		                      emergency_contact, caregiver_id, clinician_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id`,
		args.UserID, args.FullName, args.DOB, diagnosisEnc, allergiesEnc,
		args.EmergencyContact, args.CaregiverID, args.ClinicianID,
	).Scan(&id)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("creating patient %s: %w", args.FullName, err)
	}
	return id, nil
}

func upsertDischarge(ctx context.Context, pool *db.Pool, cipher *crypto.Cipher, patientID, clinicianID uuid.UUID, date time.Time, summary string) error {
	var exists bool
	if err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM discharge_records WHERE patient_id = $1)`, patientID).Scan(&exists); err != nil {
		return fmt.Errorf("checking discharge record: %w", err)
	}
	if exists {
		return nil
	}

	summaryEnc, err := cipher.EncryptString(summary)
	if err != nil {
		return err
	}

	_, err = pool.Exec(ctx,
		`INSERT INTO discharge_records (patient_id, clinician_id, summary_enc, discharge_date) VALUES ($1, $2, $3, $4)`,
		patientID, clinicianID, summaryEnc, date)
	if err != nil {
		return fmt.Errorf("creating discharge record: %w", err)
	}
	return nil
}

func upsertMedication(ctx context.Context, pool *db.Pool, patientID uuid.UUID, name, dosage string, frequencyHours int, start, end time.Time) error {
	var exists bool
	if err := pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM medications WHERE patient_id = $1 AND name = $2)`,
		patientID, name).Scan(&exists); err != nil {
		return fmt.Errorf("checking medication: %w", err)
	}
	if exists {
		return nil
	}

	_, err := pool.Exec(ctx,
		`INSERT INTO medications (patient_id, name, dosage, frequency_hours, start_date, end_date)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		patientID, name, dosage, frequencyHours, start, end)
	if err != nil {
		return fmt.Errorf("creating medication %s: %w", name, err)
	}
	return nil
}

func strptr(s string) *string { return &s }
