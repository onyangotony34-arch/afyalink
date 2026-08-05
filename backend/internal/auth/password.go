package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Argon2id parameters, per the OWASP Password Storage Cheat Sheet's current
// recommendation: 19 MiB of memory, 2 iterations, 1 degree of parallelism.
//
// OWASP lists m=47104/t=1/p=1 as an equally strong alternative — the two trade
// RAM against CPU. The 19 MiB / t=2 configuration is chosen here because the
// deployment target is a small managed container where memory is the scarcer
// resource.
//
// https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html
const (
	argonMemoryKiB  uint32 = 19 * 1024 // 19 MiB
	argonIterations uint32 = 2
	argonThreads    uint8  = 1
	argonSaltBytes  uint32 = 16
	argonKeyBytes   uint32 = 32
)

// ErrInvalidPassword is returned when a password does not match its hash.
var ErrInvalidPassword = errors.New("invalid password")

// errMalformedHash indicates a stored hash that cannot be parsed. It is
// treated as a verification failure, never surfaced to the caller.
var errMalformedHash = errors.New("malformed password hash")

// HashPassword derives an Argon2id hash and encodes it in PHC string format:
//
//	$argon2id$v=19$m=19456,t=2,p=1$<salt>$<hash>
//
// Encoding the parameters alongside the digest means the cost can be raised
// later without invalidating existing credentials: old hashes still verify
// under their own recorded parameters.
func HashPassword(password string) (string, error) {
	salt := make([]byte, argonSaltBytes)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return "", fmt.Errorf("auth: generating salt: %w", err)
	}

	digest := argon2.IDKey([]byte(password), salt, argonIterations, argonMemoryKiB, argonThreads, argonKeyBytes)

	return fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		argonMemoryKiB, argonIterations, argonThreads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(digest),
	), nil
}

// VerifyPassword checks a password against a PHC-encoded Argon2id hash using a
// constant-time comparison.
func VerifyPassword(password, encodedHash string) error {
	params, salt, want, err := decodeHash(encodedHash)
	if err != nil {
		return ErrInvalidPassword
	}

	got := argon2.IDKey([]byte(password), salt, params.iterations, params.memoryKiB, params.threads, uint32(len(want)))

	if subtle.ConstantTimeCompare(got, want) != 1 {
		return ErrInvalidPassword
	}
	return nil
}

// dummyHash is verified against when a login names an unknown email, so that
// "no such user" costs the same wall-clock time as "wrong password". Without
// it, response latency alone reveals which addresses are registered.
var dummyHash = mustHash("argon2id-timing-equalisation-placeholder")

// EqualiseTiming performs a throwaway Argon2id verification. Call it on the
// user-not-found branch of login.
func EqualiseTiming() {
	_ = VerifyPassword("not-the-password", dummyHash)
}

func mustHash(password string) string {
	h, err := HashPassword(password)
	if err != nil {
		panic("auth: hashing timing placeholder: " + err.Error())
	}
	return h
}

type argonParams struct {
	memoryKiB  uint32
	iterations uint32
	threads    uint8
}

// decodeHash parses a PHC-format Argon2id string.
func decodeHash(encoded string) (argonParams, []byte, []byte, error) {
	var params argonParams

	parts := strings.Split(encoded, "$")
	// ["", "argon2id", "v=19", "m=...,t=...,p=...", salt, hash]
	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" {
		return params, nil, nil, errMalformedHash
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return params, nil, nil, errMalformedHash
	}
	if version != argon2.Version {
		return params, nil, nil, errMalformedHash
	}

	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &params.memoryKiB, &params.iterations, &params.threads); err != nil {
		return params, nil, nil, errMalformedHash
	}
	if params.memoryKiB == 0 || params.iterations == 0 || params.threads == 0 {
		return params, nil, nil, errMalformedHash
	}

	salt, err := base64.RawStdEncoding.Strict().DecodeString(parts[4])
	if err != nil {
		return params, nil, nil, errMalformedHash
	}

	digest, err := base64.RawStdEncoding.Strict().DecodeString(parts[5])
	if err != nil || len(digest) == 0 {
		return params, nil, nil, errMalformedHash
	}

	return params, salt, digest, nil
}
