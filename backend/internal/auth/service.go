package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/OderoCeasar/afyalink/backend/internal/db"
)

// ErrInvalidCredentials is the single error returned for every failed login,
// whether the email is unknown or the password is wrong. Distinguishing them
// would turn the login endpoint into an account-existence oracle.
var ErrInvalidCredentials = errors.New("invalid email or password")

// ErrTokenReplay indicates a refresh token was presented after it had already
// been rotated or revoked.
var ErrTokenReplay = errors.New("refresh token has already been used")

// ErrInvalidRefreshToken covers unknown, expired, or malformed refresh tokens.
var ErrInvalidRefreshToken = errors.New("invalid or expired refresh token")

// Service implements the authentication use cases.
type Service struct {
	store           *Store
	jwt             *JWTManager
	refreshTokenTTL time.Duration
}

// NewService wires the auth service.
func NewService(store *Store, jwtManager *JWTManager, refreshTTL time.Duration) *Service {
	return &Service{store: store, jwt: jwtManager, refreshTokenTTL: refreshTTL}
}

// TokenPair is what a successful login or refresh produces.
type TokenPair struct {
	AccessToken     string
	AccessExpiresIn time.Duration
	RefreshToken    string
	RefreshExpires  time.Time
	User            User
}

// Register creates a user. There is no public self-signup in this MVP: the
// only caller is a clinician bootstrapping a patient or caregiver account, and
// the route enforcing that lives in the handler layer.
func (s *Service) Register(ctx context.Context, email string, phone *string, password string, role Role) (User, error) {
	if !role.Valid() {
		return User{}, fmt.Errorf("auth: unknown role %q", role)
	}

	hash, err := HashPassword(password)
	if err != nil {
		return User{}, err
	}

	user, err := s.store.CreateUser(ctx, strings.TrimSpace(email), phone, hash, role)
	if err != nil {
		return User{}, err
	}
	return user, nil
}

// Login verifies credentials and issues a fresh token pair.
func (s *Service) Login(ctx context.Context, email, password string) (TokenPair, error) {
	user, err := s.store.GetUserByEmail(ctx, strings.TrimSpace(email))
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			// Spend the same Argon2id work as a real verification so response
			// time does not reveal whether the account exists.
			EqualiseTiming()
			return TokenPair{}, ErrInvalidCredentials
		}
		return TokenPair{}, err
	}

	if err := VerifyPassword(password, user.PasswordHash); err != nil {
		return TokenPair{}, ErrInvalidCredentials
	}

	return s.issuePair(ctx, user)
}

// Refresh validates and rotates a refresh token.
//
// Rotation is unconditional: the presented token is revoked whether or not the
// caller ever uses the replacement. Presenting an already-revoked token means
// either a stolen token is in play or a client replayed one, and both are
// handled by revoking the user's entire token family.
func (s *Service) Refresh(ctx context.Context, refreshToken string) (TokenPair, error) {
	if refreshToken == "" {
		return TokenPair{}, ErrInvalidRefreshToken
	}

	stored, err := s.store.GetRefreshTokenByHash(ctx, hashRefreshToken(refreshToken))
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return TokenPair{}, ErrInvalidRefreshToken
		}
		return TokenPair{}, err
	}

	now := time.Now()

	if stored.RevokedAt != nil {
		// Replay of a token we already rotated away. We cannot tell the
		// attacker's request from the legitimate client's, so we end every
		// session for this user.
		if err := s.store.RevokeAllForUser(ctx, stored.UserID); err != nil {
			return TokenPair{}, err
		}
		return TokenPair{}, ErrTokenReplay
	}

	if !now.Before(stored.ExpiresAt) {
		return TokenPair{}, ErrInvalidRefreshToken
	}

	user, err := s.store.GetUserByID(ctx, stored.UserID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return TokenPair{}, ErrInvalidRefreshToken
		}
		return TokenPair{}, err
	}

	newToken, newHash, err := generateRefreshToken()
	if err != nil {
		return TokenPair{}, err
	}
	newExpiry := now.Add(s.refreshTokenTTL)

	if err := s.store.RotateRefreshToken(ctx, stored.ID, user.ID, newHash, newExpiry); err != nil {
		if errors.Is(err, ErrTokenReplay) {
			// Lost a race against a concurrent refresh using the same token.
			if revokeErr := s.store.RevokeAllForUser(ctx, stored.UserID); revokeErr != nil {
				return TokenPair{}, revokeErr
			}
			return TokenPair{}, ErrTokenReplay
		}
		return TokenPair{}, err
	}

	accessToken, err := s.jwt.Issue(user.ID, user.Role)
	if err != nil {
		return TokenPair{}, err
	}

	return TokenPair{
		AccessToken:     accessToken,
		AccessExpiresIn: s.jwt.TTL(),
		RefreshToken:    newToken,
		RefreshExpires:  newExpiry,
		User:            user,
	}, nil
}

// Logout revokes the presented refresh token.
//
// It reports no error for an unknown token: logout is idempotent, and telling
// a caller their token was not found would leak token validity.
func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	if refreshToken == "" {
		return nil
	}
	return s.store.RevokeRefreshToken(ctx, hashRefreshToken(refreshToken))
}

// LogoutAll revokes every refresh token for a user.
func (s *Service) LogoutAll(ctx context.Context, userID uuid.UUID) error {
	return s.store.RevokeAllForUser(ctx, userID)
}

// issuePair mints a new access and refresh token for an authenticated user.
func (s *Service) issuePair(ctx context.Context, user User) (TokenPair, error) {
	accessToken, err := s.jwt.Issue(user.ID, user.Role)
	if err != nil {
		return TokenPair{}, err
	}

	refreshToken, refreshHash, err := generateRefreshToken()
	if err != nil {
		return TokenPair{}, err
	}

	expiresAt := time.Now().Add(s.refreshTokenTTL)
	if err := s.store.InsertRefreshToken(ctx, user.ID, refreshHash, expiresAt); err != nil {
		return TokenPair{}, err
	}

	return TokenPair{
		AccessToken:     accessToken,
		AccessExpiresIn: s.jwt.TTL(),
		RefreshToken:    refreshToken,
		RefreshExpires:  expiresAt,
		User:            user,
	}, nil
}
