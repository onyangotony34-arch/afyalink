package checkins

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// QuestionType enumerates the answer shapes a check-in question can take.
type QuestionType string

const (
	// QuestionYesNo accepts exactly "yes" or "no".
	QuestionYesNo QuestionType = "yes_no"
	// QuestionScale accepts an integer within [ScaleMin, ScaleMax].
	QuestionScale QuestionType = "scale"
)

// Comparison operators available to a red-flag rule.
const (
	OpEq  = "eq"
	OpNeq = "neq"
	OpGT  = "gt"
	OpGTE = "gte"
	OpLT  = "lt"
	OpLTE = "lte"
)

// RedFlagRule declares which answers to a question count as a red flag, for
// example {"op":"eq","value":"yes"} or {"op":"gte","value":8}.
type RedFlagRule struct {
	Op    string `json:"op"`
	Value any    `json:"value"`
}

// Question is one entry of checkin_templates.questions_json.
type Question struct {
	ID       string       `json:"id"`
	Text     string       `json:"text"`
	Type     QuestionType `json:"type"`
	ScaleMin *int         `json:"scale_min,omitempty"`
	ScaleMax *int         `json:"scale_max,omitempty"`

	// Critical marks a question whose red flag is severe on its own — chest
	// pain, for instance. One critical flag outranks any number of ordinary
	// ones when severity is derived.
	Critical bool `json:"critical,omitempty"`

	// RedFlagIf is optional: a question with no rule is informational and can
	// never raise an alert.
	RedFlagIf *RedFlagRule `json:"red_flag_if,omitempty"`
}

// Questions is a template's ordered question list.
type Questions []Question

// ParseQuestions decodes a template's questions_json.
func ParseQuestions(raw []byte) (Questions, error) {
	var questions Questions
	if err := json.Unmarshal(raw, &questions); err != nil {
		return nil, fmt.Errorf("checkins: decoding questions_json: %w", err)
	}
	if len(questions) == 0 {
		return nil, fmt.Errorf("checkins: template has no questions")
	}
	for _, q := range questions {
		if err := q.validate(); err != nil {
			return nil, err
		}
	}
	return questions, nil
}

// Find returns the question with the given id.
func (qs Questions) Find(id string) (Question, bool) {
	for _, q := range qs {
		if q.ID == id {
			return q, true
		}
	}
	return Question{}, false
}

// validate rejects a template that could not be evaluated consistently. A
// malformed template is a seeding bug, and failing here is far better than
// silently never flagging a symptom.
func (q Question) validate() error {
	if strings.TrimSpace(q.ID) == "" {
		return fmt.Errorf("checkins: question is missing an id")
	}
	switch q.Type {
	case QuestionYesNo:
	case QuestionScale:
		if q.ScaleMin == nil || q.ScaleMax == nil {
			return fmt.Errorf("checkins: scale question %q must define scale_min and scale_max", q.ID)
		}
		if *q.ScaleMin >= *q.ScaleMax {
			return fmt.Errorf("checkins: scale question %q has scale_min >= scale_max", q.ID)
		}
	default:
		return fmt.Errorf("checkins: question %q has unknown type %q", q.ID, q.Type)
	}

	if q.RedFlagIf != nil {
		switch q.RedFlagIf.Op {
		case OpEq, OpNeq, OpGT, OpGTE, OpLT, OpLTE:
		default:
			return fmt.Errorf("checkins: question %q has unknown red_flag_if op %q", q.ID, q.RedFlagIf.Op)
		}
	}
	return nil
}

// ValidateAnswer checks that an answer is well-formed for its question type.
//
// This runs before rule evaluation, so a nonsense answer is a 400 rather than
// something that quietly fails to match a red-flag rule and gets recorded as
// "not a concern".
func (q Question) ValidateAnswer(answer string) error {
	normalized := strings.ToLower(strings.TrimSpace(answer))

	switch q.Type {
	case QuestionYesNo:
		if normalized != "yes" && normalized != "no" {
			return fmt.Errorf("answer must be either \"yes\" or \"no\"")
		}
		return nil

	case QuestionScale:
		value, err := strconv.Atoi(normalized)
		if err != nil {
			return fmt.Errorf("answer must be a whole number between %d and %d", *q.ScaleMin, *q.ScaleMax)
		}
		if value < *q.ScaleMin || value > *q.ScaleMax {
			return fmt.Errorf("answer must be between %d and %d", *q.ScaleMin, *q.ScaleMax)
		}
		return nil

	default:
		return fmt.Errorf("question has an unsupported type")
	}
}

// IsRedFlag evaluates an answer against the question's rule.
//
// A question with no rule never flags. An answer that cannot be compared
// (a non-numeric answer to a numeric rule) does not flag either — but that
// case is unreachable in practice because ValidateAnswer runs first.
func (q Question) IsRedFlag(answer string) bool {
	if q.RedFlagIf == nil {
		return false
	}

	normalized := strings.ToLower(strings.TrimSpace(answer))

	// Numeric comparison when the rule's value is a number. JSON numbers
	// decode into float64, which is why the rule value is matched on that
	// type rather than int.
	if threshold, ok := numericValue(q.RedFlagIf.Value); ok {
		given, err := strconv.ParseFloat(normalized, 64)
		if err != nil {
			return false
		}
		switch q.RedFlagIf.Op {
		case OpEq:
			return given == threshold
		case OpNeq:
			return given != threshold
		case OpGT:
			return given > threshold
		case OpGTE:
			return given >= threshold
		case OpLT:
			return given < threshold
		case OpLTE:
			return given <= threshold
		default:
			return false
		}
	}

	// String comparison otherwise. Only equality operators are meaningful;
	// ordering strings like "yes" and "no" would be nonsense.
	want, ok := q.RedFlagIf.Value.(string)
	if !ok {
		return false
	}
	want = strings.ToLower(strings.TrimSpace(want))

	switch q.RedFlagIf.Op {
	case OpEq:
		return normalized == want
	case OpNeq:
		return normalized != want
	default:
		return false
	}
}

func numericValue(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case json.Number:
		f, err := n.Float64()
		return f, err == nil
	default:
		return 0, false
	}
}

// Severity mirrors the alert_severity enum.
type Severity string

const (
	SeverityLow      Severity = "low"
	SeverityMedium   Severity = "medium"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)

// DeriveSeverity maps a set of flagged questions to an alert severity, per
// spec §5.5:
//
//	any question marked critical  -> critical
//	two or more flags             -> high
//	exactly one flag              -> medium
//	no flags                      -> no alert (ok is false)
//
// The critical check comes first deliberately: a single chest-pain flag must
// outrank two mild ones, so it cannot be reached by counting.
func DeriveSeverity(flagged []Question) (Severity, bool) {
	if len(flagged) == 0 {
		return "", false
	}

	for _, q := range flagged {
		if q.Critical {
			return SeverityCritical, true
		}
	}

	if len(flagged) >= 2 {
		return SeverityHigh, true
	}
	return SeverityMedium, true
}

// AlertMessage summarises the flagged symptoms for the clinician's inbox.
//
// It quotes question text, which is fixed template copy, and never the
// patient's answers — the answers live in checkin_responses behind the
// ownership checks, while alert messages are comparatively widely read.
func AlertMessage(templateName string, flagged []Question) string {
	labels := make([]string, 0, len(flagged))
	for _, q := range flagged {
		labels = append(labels, q.Text)
	}

	if len(flagged) == 1 {
		return fmt.Sprintf("%s: red flag reported — %s", templateName, labels[0])
	}
	return fmt.Sprintf("%s: %d red flags reported — %s", templateName, len(flagged), strings.Join(labels, "; "))
}
