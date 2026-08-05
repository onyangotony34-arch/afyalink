// Package config loads and validates process configuration from the
// environment. Every secret is validated at startup so a misconfigured deploy
// fails loudly at boot rather than silently serving PHI with a weak key.
package config

import (
	"encoding/base64"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all runtime configuration. It is read once at startup and
// treated as immutable thereafter.
type Config struct {
	Env             string
	Port            string
	DatabaseURL     string
	JWTSecret       []byte
	PHIKey          []byte
	CORSOrigins     []string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration

	// LoginRateLimit is the sustained requests-per-minute allowed against the
	// auth endpoints, per client IP.
	LoginRateLimit  int
	LoginRateBurst  int
	SchedulerPeriod time.Duration

	// TrustedProxies lists proxy CIDRs whose X-Forwarded-For header may be
	// believed. Empty means trust none, so ClientIP falls back to the socket
	// peer address. That default matters: if any client could set
	// X-Forwarded-For, the per-IP login rate limit would be trivially bypassed
	// by varying the header.
	TrustedProxies []string
}

// IsProduction reports whether the process is running with production
// hardening expectations (no dev fallbacks, secure cookies).
func (c *Config) IsProduction() bool { return c.Env == "production" }

// Load reads configuration from the environment, falling back to a .env file
// when one is present. Missing or malformed required values are returned as an
// error rather than defaulted, because every one of them is security-relevant.
func Load() (*Config, error) {
	// A missing .env is not an error: in production the values come from the
	// platform's secret store, not a file on disk.
	_ = godotenv.Load()

	cfg := &Config{
		Env:  getEnvDefault("APP_ENV", "development"),
		Port: getEnvDefault("PORT", "8080"),
	}

	var err error

	if cfg.DatabaseURL = os.Getenv("DATABASE_URL"); cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	if cfg.JWTSecret, err = decodeSecret("JWT_SECRET", 32); err != nil {
		return nil, err
	}

	// AES-256 requires exactly 32 bytes; anything else is a configuration bug.
	if cfg.PHIKey, err = decodeSecret("PHI_ENCRYPTION_KEY", 32); err != nil {
		return nil, err
	}
	if len(cfg.PHIKey) != 32 {
		return nil, fmt.Errorf("PHI_ENCRYPTION_KEY must decode to exactly 32 bytes for AES-256, got %d", len(cfg.PHIKey))
	}

	// CORS is an explicit allowlist. A wildcard is rejected outright rather
	// than silently accepted, per the security checklist.
	origins := getEnvDefault("CORS_ORIGINS", "http://localhost:5173")
	for _, o := range strings.Split(origins, ",") {
		o = strings.TrimSpace(o)
		if o == "" {
			continue
		}
		if o == "*" {
			return nil, fmt.Errorf("CORS_ORIGINS must not contain a wildcard; list the frontend origin explicitly")
		}
		cfg.CORSOrigins = append(cfg.CORSOrigins, o)
	}
	if len(cfg.CORSOrigins) == 0 {
		return nil, fmt.Errorf("CORS_ORIGINS must list at least one origin")
	}

	if cfg.AccessTokenTTL, err = getDuration("ACCESS_TOKEN_TTL", 15*time.Minute); err != nil {
		return nil, err
	}
	if cfg.RefreshTokenTTL, err = getDuration("REFRESH_TOKEN_TTL", 7*24*time.Hour); err != nil {
		return nil, err
	}
	if cfg.SchedulerPeriod, err = getDuration("SCHEDULER_PERIOD", time.Minute); err != nil {
		return nil, err
	}
	if cfg.LoginRateLimit, err = getInt("LOGIN_RATE_LIMIT_PER_MIN", 10); err != nil {
		return nil, err
	}
	if cfg.LoginRateBurst, err = getInt("LOGIN_RATE_BURST", 5); err != nil {
		return nil, err
	}

	for _, proxy := range strings.Split(os.Getenv("TRUSTED_PROXIES"), ",") {
		if proxy = strings.TrimSpace(proxy); proxy != "" {
			cfg.TrustedProxies = append(cfg.TrustedProxies, proxy)
		}
	}

	return cfg, nil
}

// decodeSecret reads a base64 (std or raw, padded or not) secret and enforces a
// minimum decoded length.
func decodeSecret(key string, minBytes int) ([]byte, error) {
	raw := os.Getenv(key)
	if raw == "" {
		return nil, fmt.Errorf("%s is required (generate with: openssl rand -base64 32)", key)
	}

	decoded, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		if decoded, err = base64.RawStdEncoding.DecodeString(raw); err != nil {
			return nil, fmt.Errorf("%s must be valid base64: %w", key, err)
		}
	}
	if len(decoded) < minBytes {
		return nil, fmt.Errorf("%s must decode to at least %d bytes, got %d", key, minBytes, len(decoded))
	}
	return decoded, nil
}

func getEnvDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getDuration(key string, fallback time.Duration) (time.Duration, error) {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback, nil
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("%s must be a duration such as 15m or 168h: %w", key, err)
	}
	if d <= 0 {
		return 0, fmt.Errorf("%s must be positive", key)
	}
	return d, nil
}

func getInt(key string, fallback int) (int, error) {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer: %w", key, err)
	}
	if n <= 0 {
		return 0, fmt.Errorf("%s must be positive", key)
	}
	return n, nil
}
