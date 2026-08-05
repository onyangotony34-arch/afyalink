package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/OderoCeasar/afyalink/backend/internal/auth"
	"github.com/OderoCeasar/afyalink/backend/internal/httpx"
)

// requestIDHeader is echoed back so a client-reported problem can be traced to
// a log line without the client quoting any request content.
const requestIDHeader = "X-Request-ID"

// RequestID assigns each request a correlation id.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := uuid.NewString()
		c.Set("request_id", id)
		c.Writer.Header().Set(requestIDHeader, id)
		c.Next()
	}
}

// RequestLogger emits one structured line per request.
//
// It logs the matched route template (c.FullPath()) rather than the raw URL.
// That is a PHI decision, not a cosmetic one: raw paths embed patient UUIDs,
// and query strings could carry anything a client appended. Neither belongs in
// a log aggregator. Resource identifiers reach the audit table instead, which
// is access-controlled.
func RequestLogger(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		route := c.FullPath()
		if route == "" {
			route = "unmatched"
		}

		attrs := []any{
			"request_id", c.GetString("request_id"),
			"method", c.Request.Method,
			"route", route,
			"status", c.Writer.Status(),
			"duration_ms", time.Since(start).Milliseconds(),
			"ip", c.ClientIP(),
		}

		if principal, ok := auth.PrincipalFrom(c); ok {
			attrs = append(attrs, "user_id", principal.UserID.String(), "role", principal.Role.String())
		}

		switch {
		case len(c.Errors) > 0:
			// Errors staged by httpx.Internal. They are logged here and never
			// returned to the client.
			attrs = append(attrs, "err", c.Errors.String())
			logger.Error("request failed", attrs...)
		case c.Writer.Status() >= http.StatusInternalServerError:
			logger.Error("request failed", attrs...)
		case c.Writer.Status() >= http.StatusBadRequest:
			logger.Warn("request rejected", attrs...)
		default:
			logger.Info("request", attrs...)
		}
	}
}

// Recovery converts a panic into a 500 without leaking the stack trace to the
// client. The trace goes to the log, where it is visible to operators only.
func Recovery(logger *slog.Logger) gin.HandlerFunc {
	return gin.CustomRecoveryWithWriter(nil, func(c *gin.Context, recovered any) {
		logger.Error("panic recovered",
			"request_id", c.GetString("request_id"),
			"route", c.FullPath(),
			"panic", recovered,
		)
		httpx.Internal(c, nil)
	})
}
