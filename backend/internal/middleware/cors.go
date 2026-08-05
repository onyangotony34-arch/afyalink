package middleware

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// CORS allows exactly the configured frontend origins.
//
// There is no wildcard path through this function, and config.Load rejects "*"
// before the server starts. Credentials are allowed because the refresh cookie
// must reach /api/auth — and `Access-Control-Allow-Credentials: true` combined
// with a wildcard origin is both forbidden by the spec and a total bypass of
// the same-origin policy, which is precisely why the allowlist is exact-match.
func CORS(allowedOrigins []string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		allowed[strings.TrimRight(origin, "/")] = struct{}{}
	}

	const maxAge = 12 * time.Hour

	return func(c *gin.Context) {
		origin := strings.TrimRight(c.GetHeader("Origin"), "/")

		if _, ok := allowed[origin]; ok && origin != "" {
			header := c.Writer.Header()
			header.Set("Access-Control-Allow-Origin", origin)
			header.Set("Access-Control-Allow-Credentials", "true")
			header.Set("Access-Control-Allow-Methods", "GET, POST, PATCH, OPTIONS")
			header.Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			header.Set("Access-Control-Max-Age", strconv.Itoa(int(maxAge.Seconds())))

			// Responses differ by Origin, so caches must not serve one
			// origin's response to another.
			header.Add("Vary", "Origin")
		}

		if c.Request.Method == http.MethodOptions {
			// A disallowed origin reaches here without the allow headers, so
			// the browser blocks the real request. Answering 204 either way
			// keeps the preflight from doubling as an origin oracle.
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// SecurityHeaders sets defensive response headers.
//
// This is a JSON API with no HTML surface, so the CSP is the most restrictive
// one possible: nothing may be loaded or framed at all.
func SecurityHeaders(isProduction bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.Writer.Header()
		header.Set("X-Content-Type-Options", "nosniff")
		header.Set("X-Frame-Options", "DENY")
		header.Set("Referrer-Policy", "no-referrer")
		header.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")

		// Responses carry PHI; keep them out of shared and browser caches.
		header.Set("Cache-Control", "no-store")

		if isProduction {
			header.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		c.Next()
	}
}
