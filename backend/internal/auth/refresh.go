package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
)

// refreshTokenBytes is the entropy of an opaque refresh token. 32 bytes is far
// beyond guessable and keeps the encoded token compact enough for a cookie.
const refreshTokenBytes = 32

// generateRefreshToken returns the opaque token handed to the client together
// with the SHA-256 hash that is all the database ever stores.
//
// The token is a random string rather than a JWT on purpose: refresh tokens
// must be revocable, and revocation requires server-side state regardless, so
// there is nothing to gain from making them self-describing.
func generateRefreshToken() (token string, hash string, err error) {
	raw := make([]byte, refreshTokenBytes)
	if _, err := io.ReadFull(rand.Reader, raw); err != nil {
		return "", "", fmt.Errorf("auth: generating refresh token: %w", err)
	}

	token = base64.RawURLEncoding.EncodeToString(raw)
	return token, hashRefreshToken(token), nil
}

// hashRefreshToken is a plain SHA-256, not a password KDF.
//
// That is the correct choice here and not an oversight: the input is 256 bits
// of uniform randomness, so there is no dictionary to attack and no work
// factor worth paying. Argon2 on this path would only add latency to every
// refresh. The same reasoning does NOT apply to user passwords, which are low
// entropy and are hashed with Argon2id in password.go.
func hashRefreshToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
