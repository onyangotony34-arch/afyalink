// Package httpx centralises API error responses.
//
// The single most important rule here: patient-scoped resources answer
// "you may not have this" and "this does not exist" identically. If a
// caregiver guessing UUIDs could tell 403 from 404, the UUID space becomes an
// enumeration oracle over the patient roster.
package httpx

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/OderoCeasar/afyalink/backend/pkg/validator"
)

// ErrorResponse is the uniform error envelope for every failed request.
type ErrorResponse struct {
	Error  string                 `json:"error"`
	Code   string                 `json:"code"`
	Fields []validator.FieldError `json:"fields,omitempty"`
}

// Error codes are stable strings the frontend can branch on without parsing
// human-readable messages (and without those messages becoming an i18n
// problem on the client).
const (
	CodeValidation   = "validation_error"
	CodeUnauthorized = "unauthorized"
	CodeForbidden    = "forbidden"
	CodeNotFound     = "not_found"
	CodeConflict     = "conflict"
	CodeRateLimited  = "rate_limited"
	CodeInternal     = "internal_error"
)

// BadRequest renders a validation failure. A *validator.ValidationError is
// expanded into per-field detail; anything else gets a generic message so an
// internal error string cannot leak through this path.
func BadRequest(c *gin.Context, err error) {
	if verr, ok := err.(*validator.ValidationError); ok {
		c.AbortWithStatusJSON(http.StatusBadRequest, ErrorResponse{
			Error:  verr.Message,
			Code:   CodeValidation,
			Fields: verr.Fields,
		})
		return
	}
	c.AbortWithStatusJSON(http.StatusBadRequest, ErrorResponse{
		Error: "request could not be processed",
		Code:  CodeValidation,
	})
}

// Unauthorized signals missing or invalid credentials.
func Unauthorized(c *gin.Context, message string) {
	if message == "" {
		message = "authentication required"
	}
	c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{
		Error: message,
		Code:  CodeUnauthorized,
	})
}

// Forbidden is the catch-all denial for patient-scoped data.
//
// Callers must use this for a missing resource as well as an unowned one. The
// message is intentionally constant and uninformative — see the package doc.
func Forbidden(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusForbidden, ErrorResponse{
		Error: "you do not have access to this resource",
		Code:  CodeForbidden,
	})
}

// NotFound is only for resources that are not patient-scoped (for example an
// unrouted path). Never use it to report a patient record that exists but
// belongs to someone else.
func NotFound(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusNotFound, ErrorResponse{
		Error: "resource not found",
		Code:  CodeNotFound,
	})
}

// Conflict reports a violated uniqueness or state constraint.
func Conflict(c *gin.Context, message string) {
	if message == "" {
		message = "resource conflict"
	}
	c.AbortWithStatusJSON(http.StatusConflict, ErrorResponse{
		Error: message,
		Code:  CodeConflict,
	})
}

// TooManyRequests reports a tripped rate limit.
func TooManyRequests(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusTooManyRequests, ErrorResponse{
		Error: "too many requests, please try again shortly",
		Code:  CodeRateLimited,
	})
}

// Internal records the underlying error on the Gin context for the logging
// middleware and returns an opaque message. The real error never reaches the
// client, because for this service it may quote a decrypted PHI field or a
// connection string.
func Internal(c *gin.Context, err error) {
	if err != nil {
		_ = c.Error(err)
	}
	c.AbortWithStatusJSON(http.StatusInternalServerError, ErrorResponse{
		Error: "an unexpected error occurred",
		Code:  CodeInternal,
	})
}
