package middleware

import (
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"

	"github.com/OderoCeasar/afyalink/backend/internal/httpx"
)

// RateLimiter is a per-client-IP token bucket.
//
// In-process state is a deliberate MVP choice: with more than one API replica
// each would enforce its own bucket, so the effective limit multiplies by the
// replica count. Redis is explicitly out of scope for this pass (spec §7), so
// the shared-store version is a Phase 2 item — noted in the README.
type RateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitor

	limit rate.Limit
	burst int
	ttl   time.Duration
}

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// NewRateLimiter builds a limiter allowing perMinute sustained requests with
// the given burst, per IP.
func NewRateLimiter(perMinute, burst int) *RateLimiter {
	rl := &RateLimiter{
		visitors: make(map[string]*visitor),
		limit:    rate.Limit(float64(perMinute) / 60.0),
		burst:    burst,
		ttl:      10 * time.Minute,
	}

	go rl.reapLoop()
	return rl
}

// Middleware rejects requests from an IP that has exceeded its budget.
//
// Mounted on /auth/login and /auth/refresh, this is what turns credential
// stuffing and refresh-token brute forcing from cheap into impractical.
func (rl *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// ClientIP honours trusted proxy configuration; see main.go, where the
		// trusted proxy list is set explicitly so a client cannot spoof
		// X-Forwarded-For to get a fresh bucket.
		if !rl.limiterFor(c.ClientIP()).Allow() {
			httpx.TooManyRequests(c)
			return
		}
		c.Next()
	}
}

func (rl *RateLimiter) limiterFor(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	v, ok := rl.visitors[ip]
	if !ok {
		v = &visitor{limiter: rate.NewLimiter(rl.limit, rl.burst)}
		rl.visitors[ip] = v
	}
	v.lastSeen = time.Now()
	return v.limiter
}

// reapLoop evicts idle buckets so the map cannot grow without bound as an
// attacker rotates source addresses.
func (rl *RateLimiter) reapLoop() {
	ticker := time.NewTicker(rl.ttl)
	defer ticker.Stop()

	for range ticker.C {
		cutoff := time.Now().Add(-rl.ttl)

		rl.mu.Lock()
		for ip, v := range rl.visitors {
			if v.lastSeen.Before(cutoff) {
				delete(rl.visitors, ip)
			}
		}
		rl.mu.Unlock()
	}
}
