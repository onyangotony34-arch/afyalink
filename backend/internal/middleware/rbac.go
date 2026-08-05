package middleware

import (
	"github.com/gin-gonic/gin"

	"github.com/OderoCeasar/afyalink/backend/internal/auth"
	"github.com/OderoCeasar/afyalink/backend/internal/httpx"
)

// RequireRole restricts a route to the listed roles.
//
// This is coarse, route-level authorisation only. It answers "may a caregiver
// call this endpoint at all", never "may this caregiver see this patient" —
// that second question is a row-level ownership check performed in the handler
// against the database. Both are required; neither is sufficient alone.
func RequireRole(roles ...auth.Role) gin.HandlerFunc {
	// Build the lookup set once at wiring time rather than per request.
	allowed := make(map[auth.Role]struct{}, len(roles))
	for _, r := range roles {
		allowed[r] = struct{}{}
	}

	return func(c *gin.Context) {
		principal, ok := auth.PrincipalFrom(c)
		if !ok {
			// RequireRole was mounted without RequireAuth ahead of it. Deny
			// rather than fall through to the handler.
			httpx.Unauthorized(c, "authentication required")
			return
		}

		if _, permitted := allowed[principal.Role]; !permitted {
			// Same opaque 403 the ownership checks return, so a caregiver
			// probing clinician routes cannot map the API surface.
			httpx.Forbidden(c)
			return
		}

		c.Next()
	}
}
