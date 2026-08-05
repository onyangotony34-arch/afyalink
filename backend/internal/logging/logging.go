// Package logging builds the application's structured logger.
//
// PHI discipline: nothing in this codebase logs a decrypted field, a password,
// or a token. Rather than trusting every call site to remember that, the
// handler installed here scrubs values whose keys look sensitive, so an
// accidental logger.Info("...", "password", pw) still cannot write a secret to
// disk. Treat that as a backstop, not a licence — do not pass PHI to the
// logger in the first place.
package logging

import (
	"context"
	"io"
	"log/slog"
	"strings"
)

// redactedKeys are attribute names whose values are replaced with a placeholder
// wherever they appear, at any nesting depth.
var redactedKeys = []string{
	"password", "password_hash", "passwordhash",
	"token", "access_token", "refresh_token", "authorization", "cookie",
	"diagnosis", "allergies", "summary", "note", "body",
	"jwt_secret", "phi_encryption_key", "secret", "key",
	"answer", "full_name", "fullname", "dob", "emergency_contact", "phone",
}

const redactedPlaceholder = "[REDACTED]"

// New returns a logger writing JSON to w. In development the level is Debug
// and output stays JSON so local logs match production exactly.
func New(w io.Writer, env string) *slog.Logger {
	level := slog.LevelInfo
	if env != "production" {
		level = slog.LevelDebug
	}

	handler := slog.NewJSONHandler(w, &slog.HandlerOptions{Level: level})
	return slog.New(&redactingHandler{inner: handler})
}

// redactingHandler wraps another handler and scrubs sensitive attributes.
type redactingHandler struct {
	inner slog.Handler
}

func (h *redactingHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

func (h *redactingHandler) Handle(ctx context.Context, record slog.Record) error {
	// Records are copied rather than mutated in place: slog.Record shares
	// backing storage between clones, so editing attrs directly can corrupt a
	// record another handler is still reading.
	scrubbed := slog.NewRecord(record.Time, record.Level, record.Message, record.PC)
	record.Attrs(func(attr slog.Attr) bool {
		scrubbed.AddAttrs(redact(attr))
		return true
	})
	return h.inner.Handle(ctx, scrubbed)
}

func (h *redactingHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	redacted := make([]slog.Attr, 0, len(attrs))
	for _, attr := range attrs {
		redacted = append(redacted, redact(attr))
	}
	return &redactingHandler{inner: h.inner.WithAttrs(redacted)}
}

func (h *redactingHandler) WithGroup(name string) slog.Handler {
	return &redactingHandler{inner: h.inner.WithGroup(name)}
}

func redact(attr slog.Attr) slog.Attr {
	if isSensitive(attr.Key) {
		return slog.String(attr.Key, redactedPlaceholder)
	}

	// Recurse into groups so a nested {"user": {"password": ...}} is caught.
	if attr.Value.Kind() == slog.KindGroup {
		children := attr.Value.Group()
		scrubbed := make([]any, 0, len(children))
		for _, child := range children {
			scrubbed = append(scrubbed, redact(child))
		}
		return slog.Group(attr.Key, scrubbed...)
	}

	return attr
}

func isSensitive(key string) bool {
	lower := strings.ToLower(key)
	for _, candidate := range redactedKeys {
		if strings.Contains(lower, candidate) {
			return true
		}
	}
	return false
}
