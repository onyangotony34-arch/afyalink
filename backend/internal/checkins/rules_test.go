package checkins

import (
	"testing"
)

func intPtr(v int) *int { return &v }

func yesNoQuestion(id string, critical bool, flagOn string) Question {
	return Question{
		ID:        id,
		Text:      id,
		Type:      QuestionYesNo,
		Critical:  critical,
		RedFlagIf: &RedFlagRule{Op: OpEq, Value: flagOn},
	}
}

func scaleQuestion(id string, op string, threshold float64) Question {
	return Question{
		ID:        id,
		Text:      id,
		Type:      QuestionScale,
		ScaleMin:  intPtr(0),
		ScaleMax:  intPtr(10),
		RedFlagIf: &RedFlagRule{Op: op, Value: threshold},
	}
}

func TestParseQuestionsAcceptsValidTemplate(t *testing.T) {
	raw := []byte(`[
		{"id":"chest_pain","text":"Chest pain?","type":"yes_no","critical":true,"red_flag_if":{"op":"eq","value":"yes"}},
		{"id":"pain","text":"Pain level","type":"scale","scale_min":0,"scale_max":10,"red_flag_if":{"op":"gte","value":8}}
	]`)

	questions, err := ParseQuestions(raw)
	if err != nil {
		t.Fatalf("ParseQuestions: %v", err)
	}
	if len(questions) != 2 {
		t.Fatalf("got %d questions, want 2", len(questions))
	}

	q, ok := questions.Find("chest_pain")
	if !ok {
		t.Fatal("Find(chest_pain) returned not found")
	}
	if !q.Critical {
		t.Fatal("chest_pain should be marked critical")
	}
}

// A malformed template must fail loudly at parse time. Accepting one would
// mean a symptom silently never flags.
func TestParseQuestionsRejectsMalformedTemplates(t *testing.T) {
	cases := map[string]string{
		"empty list":          `[]`,
		"missing id":          `[{"text":"x","type":"yes_no"}]`,
		"unknown type":        `[{"id":"a","text":"x","type":"freetext"}]`,
		"scale without range": `[{"id":"a","text":"x","type":"scale"}]`,
		"inverted range":      `[{"id":"a","text":"x","type":"scale","scale_min":10,"scale_max":0}]`,
		"unknown operator":    `[{"id":"a","text":"x","type":"yes_no","red_flag_if":{"op":"matches","value":"y"}}]`,
	}

	for name, raw := range cases {
		if _, err := ParseQuestions([]byte(raw)); err == nil {
			t.Errorf("%s: ParseQuestions accepted an invalid template", name)
		}
	}
}

func TestValidateAnswer(t *testing.T) {
	yesNo := yesNoQuestion("chest_pain", true, "yes")
	scale := scaleQuestion("pain", OpGTE, 8)

	valid := []struct {
		q      Question
		answer string
	}{
		{yesNo, "yes"},
		{yesNo, "no"},
		{yesNo, "YES"},   // case-insensitive
		{yesNo, "  no "}, // trimmed
		{scale, "0"},
		{scale, "10"},
		{scale, "7"},
	}
	for _, c := range valid {
		if err := c.q.ValidateAnswer(c.answer); err != nil {
			t.Errorf("ValidateAnswer(%q) on %s: unexpected error %v", c.answer, c.q.ID, err)
		}
	}

	invalid := []struct {
		q      Question
		answer string
	}{
		{yesNo, ""},
		{yesNo, "maybe"},
		{yesNo, "1"},
		{scale, "11"}, // above scale_max
		{scale, "-1"}, // below scale_min
		{scale, "high"},
		{scale, "7.5"}, // scale answers are whole numbers
	}
	for _, c := range invalid {
		if err := c.q.ValidateAnswer(c.answer); err == nil {
			t.Errorf("ValidateAnswer(%q) on %s: expected an error, got nil", c.answer, c.q.ID)
		}
	}
}

func TestIsRedFlagYesNo(t *testing.T) {
	q := yesNoQuestion("chest_pain", true, "yes")

	if !q.IsRedFlag("yes") {
		t.Error(`"yes" should flag`)
	}
	if !q.IsRedFlag("YES") {
		t.Error(`"YES" should flag (comparison is case-insensitive)`)
	}
	if q.IsRedFlag("no") {
		t.Error(`"no" should not flag`)
	}
}

// Adherence questions flag on "no", so the inverse polarity must work.
func TestIsRedFlagInvertedPolarity(t *testing.T) {
	q := yesNoQuestion("meds_taken", false, "no")

	if !q.IsRedFlag("no") {
		t.Error(`"no" should flag for a meds-adherence question`)
	}
	if q.IsRedFlag("yes") {
		t.Error(`"yes" should not flag for a meds-adherence question`)
	}
}

func TestIsRedFlagScaleBoundaries(t *testing.T) {
	q := scaleQuestion("pain", OpGTE, 8)

	if q.IsRedFlag("7") {
		t.Error("7 should not flag when the rule is gte 8")
	}
	if !q.IsRedFlag("8") {
		t.Error("8 should flag when the rule is gte 8 (boundary is inclusive)")
	}
	if !q.IsRedFlag("10") {
		t.Error("10 should flag when the rule is gte 8")
	}
}

func TestIsRedFlagOperators(t *testing.T) {
	cases := []struct {
		op        string
		threshold float64
		answer    string
		want      bool
	}{
		{OpGT, 5, "5", false},
		{OpGT, 5, "6", true},
		{OpLT, 3, "2", true},
		{OpLT, 3, "3", false},
		{OpLTE, 3, "3", true},
		{OpEq, 4, "4", true},
		{OpEq, 4, "5", false},
		{OpNeq, 4, "5", true},
	}

	for _, c := range cases {
		q := scaleQuestion("q", c.op, c.threshold)
		if got := q.IsRedFlag(c.answer); got != c.want {
			t.Errorf("%s %v against answer %q: got %v, want %v", c.op, c.threshold, c.answer, got, c.want)
		}
	}
}

// A question with no rule is informational and must never raise an alert.
func TestQuestionWithoutRuleNeverFlags(t *testing.T) {
	q := Question{ID: "mood", Text: "How are you feeling?", Type: QuestionYesNo}

	if q.IsRedFlag("yes") || q.IsRedFlag("no") {
		t.Error("a question with no red_flag_if rule must never flag")
	}
}

// Severity mapping is spec §5.5 and is the difference between a clinician
// being paged and not.
func TestDeriveSeverity(t *testing.T) {
	ordinary := yesNoQuestion("fever", false, "yes")
	ordinary2 := yesNoQuestion("wound", false, "yes")
	critical := yesNoQuestion("chest_pain", true, "yes")

	if _, raise := DeriveSeverity(nil); raise {
		t.Error("no flags must raise no alert")
	}

	got, raise := DeriveSeverity([]Question{ordinary})
	if !raise || got != SeverityMedium {
		t.Errorf("single ordinary flag: got %q raise=%v, want medium true", got, raise)
	}

	got, raise = DeriveSeverity([]Question{ordinary, ordinary2})
	if !raise || got != SeverityHigh {
		t.Errorf("two ordinary flags: got %q raise=%v, want high true", got, raise)
	}

	got, raise = DeriveSeverity([]Question{critical})
	if !raise || got != SeverityCritical {
		t.Errorf("single critical flag: got %q raise=%v, want critical true", got, raise)
	}
}

// A lone critical flag must outrank several ordinary ones. If severity were
// derived by counting first, this case would come back "high".
func TestCriticalOutranksCount(t *testing.T) {
	critical := yesNoQuestion("chest_pain", true, "yes")
	ordinary := yesNoQuestion("fever", false, "yes")

	got, raise := DeriveSeverity([]Question{ordinary, critical, ordinary})
	if !raise || got != SeverityCritical {
		t.Fatalf("critical among ordinary flags: got %q, want critical", got)
	}
}

// Alert messages reach the clinician inbox; they must quote fixed template
// copy and never the patient's answers.
func TestAlertMessageOmitsAnswers(t *testing.T) {
	q := Question{ID: "chest_pain", Text: "Are you experiencing chest pain?", Type: QuestionYesNo}

	message := AlertMessage("Day 2 check-in", []Question{q})

	if !contains(message, "Are you experiencing chest pain?") {
		t.Errorf("message should quote the question text, got %q", message)
	}
	if contains(message, "yes") {
		t.Errorf("message must not contain the patient's answer, got %q", message)
	}
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && indexOf(haystack, needle) >= 0
}

func indexOf(haystack, needle string) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}
