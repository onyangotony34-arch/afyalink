// Package validator binds and validates JSON request bodies.
//
// Binding here is deliberately stricter than Gin's ShouldBindJSON: the decoder
// rejects unknown fields and trailing content, and the body is size-capped.
// Silently ignoring an unrecognised field is how a typo'd `clinicianId` gets
// accepted as a no-op, or how a client smuggles a field the handler never
// intended to expose.
package validator

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// MaxBodyBytes caps request bodies. No endpoint in this API legitimately
// accepts a megabyte of JSON.
const MaxBodyBytes = 1 << 20 // 1 MiB

var validate *validator.Validate

func init() {
	validate = validator.New(validator.WithRequiredStructEnabled())

	// Report errors using the JSON field name the client actually sent, not
	// the Go struct field name.
	validate.RegisterTagNameFunc(func(field reflect.StructField) string {
		name := strings.SplitN(field.Tag.Get("json"), ",", 2)[0]
		if name == "-" || name == "" {
			return field.Name
		}
		return name
	})
}

// FieldError describes one rejected field. Messages never echo the submitted
// value, which for this API may be PHI.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidationError aggregates every problem found with a request body.
type ValidationError struct {
	Message string
	Fields  []FieldError
}

func (e *ValidationError) Error() string {
	if len(e.Fields) == 0 {
		return e.Message
	}
	parts := make([]string, 0, len(e.Fields))
	for _, f := range e.Fields {
		parts = append(parts, f.Field+": "+f.Message)
	}
	return e.Message + " (" + strings.Join(parts, "; ") + ")"
}

// BindJSON decodes the request body into dst and validates it. dst must be a
// pointer to a struct. The returned error is always a *ValidationError, so
// handlers can respond uniformly without inspecting decoder internals.
func BindJSON(c *gin.Context, dst any) error {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxBodyBytes)

	dec := json.NewDecoder(c.Request.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		return decodeError(err)
	}

	// Reject a second JSON document after the first; otherwise
	// `{"a":1}{"b":2}` silently binds only the first object.
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return &ValidationError{Message: "request body must contain exactly one JSON object"}
	}

	return Struct(dst)
}

// Struct validates an already-populated struct against its `validate` tags.
func Struct(dst any) error {
	if err := validate.Struct(dst); err != nil {
		var invalid *validator.InvalidValidationError
		if errors.As(err, &invalid) {
			return &ValidationError{Message: "invalid request"}
		}

		var verrs validator.ValidationErrors
		if errors.As(err, &verrs) {
			fields := make([]FieldError, 0, len(verrs))
			for _, ve := range verrs {
				fields = append(fields, FieldError{
					Field:   ve.Field(),
					Message: describe(ve),
				})
			}
			return &ValidationError{Message: "request validation failed", Fields: fields}
		}
		return &ValidationError{Message: "invalid request"}
	}
	return nil
}

// decodeError turns json decoder failures into client-safe messages.
func decodeError(err error) *ValidationError {
	var typeErr *json.UnmarshalTypeError
	if errors.As(err, &typeErr) {
		field := typeErr.Field
		if field == "" {
			field = "body"
		}
		return &ValidationError{
			Message: "request validation failed",
			Fields: []FieldError{{
				Field:   field,
				Message: fmt.Sprintf("must be of type %s", typeErr.Type.String()),
			}},
		}
	}

	var syntaxErr *json.SyntaxError
	if errors.As(err, &syntaxErr) {
		return &ValidationError{Message: "request body is not valid JSON"}
	}

	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return &ValidationError{Message: "request body is empty or truncated"}
	}

	// DisallowUnknownFields surfaces as a plain error with this prefix; it is
	// the only way to detect it without depending on encoding/json internals.
	if msg := err.Error(); strings.HasPrefix(msg, "json: unknown field ") {
		name := strings.Trim(strings.TrimPrefix(msg, "json: unknown field "), `"`)
		return &ValidationError{
			Message: "request validation failed",
			Fields:  []FieldError{{Field: name, Message: "unknown field"}},
		}
	}

	var maxBytesErr *http.MaxBytesError
	if errors.As(err, &maxBytesErr) {
		return &ValidationError{Message: "request body is too large"}
	}

	return &ValidationError{Message: "request body could not be parsed"}
}

// describe renders a validation failure without echoing the submitted value.
func describe(ve validator.FieldError) string {
	switch ve.Tag() {
	case "required":
		return "is required"
	case "email":
		return "must be a valid email address"
	case "min":
		return "must be at least " + ve.Param() + " in length or value"
	case "max":
		return "must be at most " + ve.Param() + " in length or value"
	case "gt":
		return "must be greater than " + ve.Param()
	case "gte":
		return "must be greater than or equal to " + ve.Param()
	case "lte":
		return "must be less than or equal to " + ve.Param()
	case "oneof":
		return "must be one of: " + strings.ReplaceAll(ve.Param(), " ", ", ")
	case "uuid", "uuid4":
		return "must be a valid UUID"
	case "datetime":
		return "must match the format " + ve.Param()
	default:
		return "is invalid"
	}
}
