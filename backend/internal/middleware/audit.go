package middleware

import (
	"context"
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/OderoCeasar/afyalink/backend/internal/audit"
	"github.com/OderoCeasar/afyalink/backend/internal/auth"
)

// Audit flushes the entries handlers staged during the request.
//
// Two deliberate choices:
//
//   - Entries are written with a fresh background context, not the request's.
//     A client that disconnects mid-response would otherwise cancel the write
//     and erase the record of an access that did happen.
//   - Only successful responses are audited. A 403 records an attempted access
//     rather than an actual one, and handlers do not stage entries on their
//     error paths.
func Audit(store *audit.Store, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		entries := audit.Pending(c)
		if len(entries) == 0 {
			return
		}

		status := c.Writer.Status()
		if status < 200 || status >= 300 {
			return
		}

		var userID *string
		principal, ok := auth.PrincipalFrom(c)
		if ok {
			id := principal.UserID.String()
			userID = &id
		}

		ip := c.ClientIP()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		for _, entry := range entries {
			if ok {
				id := principal.UserID
				entry.UserID = &id
			}
			entry.IPAddress = ip

			if err := store.Write(ctx, entry); err != nil {
				// The response is already written, so the request cannot be
				// failed retroactively. Log loudly instead: a persistent
				// failure here means the audit trail has gaps and must be
				// alerted on in production.
				logger.Error("audit write failed",
					"action", entry.Action,
					"resource", entry.Resource,
					"user_id", derefString(userID),
					"err", err,
				)
			}
		}
	}
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
