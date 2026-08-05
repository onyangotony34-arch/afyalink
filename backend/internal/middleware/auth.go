// Package middleware holds the cross-cutting HTTP concerns: authentication,
// role checks, rate limiting, CORS, audit flushing, and request logging.
package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/OderoCeasar/afyalink/backend/internal/auth"
	"github.com/OderoCeasar/afyalink/backend/internal/httpx"
)

// RequireAuth verifies the Bearer access token and attaches the resulting
// Principal to the request.
//
// Identity comes from the verified token and nowhere else. No handler in this
// codebase reads a user id from a body field or query parameter.
func RequireAuth(jwtManager *auth.JWTManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			httpx.Unauthorized(c, "authentication required")
			return
		}

		// Split on the first space only, and require the exact "Bearer"
		// scheme, so "Bearer  x" or "bearerx" cannot slip through.
		scheme, token, found := strings.Cut(header, " ")
		if !found || !strings.EqualFold(scheme, "Bearer") || strings.TrimSpace(token) == "" {
			httpx.Unauthorized(c, "malformed Authorization header")
			return
		}

		principal, err := jwtManager.Verify(strings.TrimSpace(token))
		if err != nil {
			httpx.Unauthorized(c, "invalid or expired access token")
			return
		}

		auth.SetPrincipal(c, principal)
		c.Next()
	}
}
