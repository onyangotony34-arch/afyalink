package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/OderoCeasar/afyalink/backend/internal/db"
)

// ErrEmailTaken is returned when registration collides with an existing user.
var ErrEmailTaken = errors.New("a user with that email already exists")

// User is a row of the users table. PasswordHash never leaves this package's
// callers and is never serialised to JSON.
type User struct {
	ID           uuid.UUID
	Email        string
	Phone        *string
	PasswordHash string
	Role         Role
	CreatedAt    time.Time
}

// RefreshToken is a row of refresh_tokens. Only the hash is ever stored.
type RefreshToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	TokenHash string
	ExpiresAt time.Time
	RevokedAt *time.Time
	CreatedAt time.Time
}

// Active reports whether the token may still be exchanged.
func (t *RefreshToken) Active(now time.Time) bool {
	return t.RevokedAt == nil && now.Before(t.ExpiresAt)
}

// Store holds the auth-related queries. Every statement uses bind parameters.
type Store struct {
	pool *db.Pool
}

// NewStore builds a Store over the shared pool.
func NewStore(pool *db.Pool) *Store { return &Store{pool: pool} }

// CreateUser inserts a user. Email is normalised to lower case so the
// case-insensitive unique index and the login lookup agree.
func (s *Store) CreateUser(ctx context.Context, email string, phone *string, passwordHash string, role Role) (User, error) {
	const query = `
		INSERT INTO users (email, phone, password_hash, role)
		VALUES (lower($1), $2, $3, $4)
		RETURNING id, email, phone, password_hash, role, created_at`

	var u User
	err := s.pool.QueryRow(ctx, query, email, phone, passwordHash, string(role)).
		Scan(&u.ID, &u.Email, &u.Phone, &u.PasswordHash, &u.Role, &u.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return User{}, ErrEmailTaken
		}
		return User{}, fmt.Errorf("auth: inserting user: %w", err)
	}
	return u, nil
}

// GetUserByEmail looks a user up for login.
func (s *Store) GetUserByEmail(ctx context.Context, email string) (User, error) {
	const query = `
		SELECT id, email, phone, password_hash, role, created_at
		FROM users
		WHERE lower(email) = lower($1)`

	var u User
	err := s.pool.QueryRow(ctx, query, email).
		Scan(&u.ID, &u.Email, &u.Phone, &u.PasswordHash, &u.Role, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, db.ErrNotFound
	}
	if err != nil {
		return User{}, fmt.Errorf("auth: selecting user by email: %w", err)
	}
	return u, nil
}

// GetUserByID looks a user up by primary key.
func (s *Store) GetUserByID(ctx context.Context, id uuid.UUID) (User, error) {
	const query = `
		SELECT id, email, phone, password_hash, role, created_at
		FROM users
		WHERE id = $1`

	var u User
	err := s.pool.QueryRow(ctx, query, id).
		Scan(&u.ID, &u.Email, &u.Phone, &u.PasswordHash, &u.Role, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, db.ErrNotFound
	}
	if err != nil {
		return User{}, fmt.Errorf("auth: selecting user by id: %w", err)
	}
	return u, nil
}

// InsertRefreshToken records the hash of a newly issued refresh token.
func (s *Store) InsertRefreshToken(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) error {
	const query = `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)`

	if _, err := s.pool.Exec(ctx, query, userID, tokenHash, expiresAt); err != nil {
		return fmt.Errorf("auth: inserting refresh token: %w", err)
	}
	return nil
}

// GetRefreshTokenByHash finds a refresh token row, including revoked ones —
// the caller needs to see a revoked row in order to detect replay.
func (s *Store) GetRefreshTokenByHash(ctx context.Context, tokenHash string) (RefreshToken, error) {
	const query = `
		SELECT id, user_id, token_hash, expires_at, revoked_at, created_at
		FROM refresh_tokens
		WHERE token_hash = $1`

	var t RefreshToken
	err := s.pool.QueryRow(ctx, query, tokenHash).
		Scan(&t.ID, &t.UserID, &t.TokenHash, &t.ExpiresAt, &t.RevokedAt, &t.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return RefreshToken{}, db.ErrNotFound
	}
	if err != nil {
		return RefreshToken{}, fmt.Errorf("auth: selecting refresh token: %w", err)
	}
	return t, nil
}

// RotateRefreshToken atomically revokes the presented token and stores its
// replacement.
//
// Both statements share one transaction so a crash between them cannot leave
// the user with either two live tokens or none at all. The UPDATE's
// `revoked_at IS NULL` guard means two concurrent refreshes with the same
// token produce exactly one winner; the loser sees zero rows affected and is
// treated as a replay.
func (s *Store) RotateRefreshToken(
	ctx context.Context,
	oldTokenID uuid.UUID,
	userID uuid.UUID,
	newTokenHash string,
	newExpiresAt time.Time,
) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("auth: beginning rotation transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	const revoke = `
		UPDATE refresh_tokens
		SET revoked_at = now()
		WHERE id = $1 AND revoked_at IS NULL`

	tag, err := tx.Exec(ctx, revoke, oldTokenID)
	if err != nil {
		return fmt.Errorf("auth: revoking rotated token: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrTokenReplay
	}

	const insert = `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)`

	if _, err := tx.Exec(ctx, insert, userID, newTokenHash, newExpiresAt); err != nil {
		return fmt.Errorf("auth: inserting rotated token: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("auth: committing rotation: %w", err)
	}
	return nil
}

// RevokeRefreshToken revokes a single token, used by logout.
func (s *Store) RevokeRefreshToken(ctx context.Context, tokenHash string) error {
	const query = `
		UPDATE refresh_tokens
		SET revoked_at = now()
		WHERE token_hash = $1 AND revoked_at IS NULL`

	if _, err := s.pool.Exec(ctx, query, tokenHash); err != nil {
		return fmt.Errorf("auth: revoking refresh token: %w", err)
	}
	return nil
}

// RevokeAllForUser revokes every live refresh token for a user. This is the
// response to a detected replay: if a stolen token is being reused, the
// legitimate session cannot be distinguished from the attacker's, so both are
// terminated and the user must log in again.
func (s *Store) RevokeAllForUser(ctx context.Context, userID uuid.UUID) error {
	const query = `
		UPDATE refresh_tokens
		SET revoked_at = now()
		WHERE user_id = $1 AND revoked_at IS NULL`

	if _, err := s.pool.Exec(ctx, query, userID); err != nil {
		return fmt.Errorf("auth: revoking user's refresh tokens: %w", err)
	}
	return nil
}
