package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// ErrInvalidToken covers every access-token failure: bad signature, wrong
// algorithm, expired, malformed, unknown role. Callers get one undifferentiated
// error so a probing client learns nothing about why a token was rejected.
var ErrInvalidToken = errors.New("invalid or expired access token")

const (
	tokenIssuer   = "afyalink"
	tokenAudience = "afyalink-api"
)

// Claims is the access token payload. The role is carried in the token so
// route-level RBAC needs no database round trip, while ownership checks — which
// do hit the database — remain the authority for row-level access.
type Claims struct {
	Role Role `json:"role"`
	jwt.RegisteredClaims
}

// JWTManager issues and verifies HS256 access tokens.
type JWTManager struct {
	secret []byte
	ttl    time.Duration
}

// NewJWTManager builds a manager. The secret length is enforced in config.
func NewJWTManager(secret []byte, ttl time.Duration) *JWTManager {
	return &JWTManager{secret: secret, ttl: ttl}
}

// TTL exposes the access token lifetime so handlers can tell the client when
// to refresh.
func (m *JWTManager) TTL() time.Duration { return m.ttl }

// Issue mints a short-lived access token for a user.
func (m *JWTManager) Issue(userID uuid.UUID, role Role) (string, error) {
	if !role.Valid() {
		return "", fmt.Errorf("auth: refusing to issue a token for unknown role %q", role)
	}

	now := time.Now()
	claims := Claims{
		Role: role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			Issuer:    tokenIssuer,
			Audience:  jwt.ClaimStrings{tokenAudience},
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.ttl)),
			ID:        uuid.NewString(),
		},
	}

	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
	if err != nil {
		return "", fmt.Errorf("auth: signing access token: %w", err)
	}
	return signed, nil
}

// Verify parses and validates a token, returning the caller it identifies.
func (m *JWTManager) Verify(tokenString string) (Principal, error) {
	// The parser is pinned to HS256. Without this, a token with alg "none" or
	// an RS256 token whose "public key" is our HMAC secret would be accepted —
	// the classic JWT algorithm-confusion attack.
	parsed, err := jwt.ParseWithClaims(
		tokenString,
		&Claims{},
		func(token *jwt.Token) (any, error) { return m.secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(tokenIssuer),
		jwt.WithAudience(tokenAudience),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return Principal{}, ErrInvalidToken
	}

	claims, ok := parsed.Claims.(*Claims)
	if !ok || !parsed.Valid {
		return Principal{}, ErrInvalidToken
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return Principal{}, ErrInvalidToken
	}

	// A token whose role is not one of the three known roles is rejected
	// rather than defaulted, so a future role rename cannot silently widen
	// access.
	if !claims.Role.Valid() {
		return Principal{}, ErrInvalidToken
	}

	return Principal{UserID: userID, Role: claims.Role}, nil
}
