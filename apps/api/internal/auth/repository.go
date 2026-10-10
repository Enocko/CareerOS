package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles user persistence.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new auth Repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// CreateUser inserts a new user and returns the created record.
func (r *Repository) CreateUser(ctx context.Context, email, passwordHash string) (*User, error) {
	var user User
	err := r.pool.QueryRow(ctx, `
		INSERT INTO users (email, password_hash)
		VALUES ($1, $2)
		RETURNING id, email, password_hash, created_at, updated_at
	`, email, passwordHash).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrEmailTaken
		}
		return nil, fmt.Errorf("insert user: %w", err)
	}
	return &user, nil
}

// GetUserByEmail retrieves a user by email address.
func (r *Repository) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	var user User
	err := r.pool.QueryRow(ctx, `
		SELECT id, email, password_hash, created_at, updated_at
		FROM users
		WHERE email = $1
	`, email).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	return &user, nil
}

// GetUserByID retrieves a user by ID.
func (r *Repository) GetUserByID(ctx context.Context, id uuid.UUID) (*User, error) {
	var user User
	err := r.pool.QueryRow(ctx, `
		SELECT id, email, password_hash, created_at, updated_at
		FROM users
		WHERE id = $1
	`, id).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return &user, nil
}

// RevokeToken records a logged-out token ID until the token would have expired.
// Expired revocations are pruned on the way, keeping the table small.
func (r *Repository) RevokeToken(ctx context.Context, tokenID string, expiresAt time.Time) error {
	if _, err := r.pool.Exec(ctx, `DELETE FROM revoked_tokens WHERE expires_at < now()`); err != nil {
		return fmt.Errorf("prune revoked tokens: %w", err)
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO revoked_tokens (token_id, expires_at)
		VALUES ($1, $2)
		ON CONFLICT (token_id) DO NOTHING
	`, tokenID, expiresAt)
	if err != nil {
		return fmt.Errorf("revoke token: %w", err)
	}
	return nil
}

// IsTokenRevoked reports whether a token ID was revoked by logout.
func (r *Repository) IsTokenRevoked(ctx context.Context, tokenID string) (bool, error) {
	var revoked bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM revoked_tokens WHERE token_id = $1)
	`, tokenID).Scan(&revoked)
	if err != nil {
		return false, fmt.Errorf("check revoked token: %w", err)
	}
	return revoked, nil
}
