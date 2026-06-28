package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/krishnamadhavan/expense-tracker/internal/domain"
	"github.com/krishnamadhavan/expense-tracker/internal/ports"
)

// UserRepo implements ports.UserRepository.
type UserRepo struct{ Pool *pgxpool.Pool }

func (r *UserRepo) Count(ctx context.Context) (int, error) {
	var n int
	err := r.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&n)
	return n, err
}

func (r *UserRepo) Create(ctx context.Context, householdID domain.HouseholdID, email, passwordHash string, role domain.UserRole) (ports.UserRecord, error) {
	const q = `
		INSERT INTO users (household_id, email, password_hash, role)
		VALUES ($1, $2, $3, $4)
		RETURNING id, household_id, email, password_hash, role`
	var u ports.UserRecord
	var roleStr string
	err := r.Pool.QueryRow(ctx, q, householdID, email, passwordHash, string(role)).Scan(
		&u.ID, &u.HouseholdID, &u.Email, &u.PasswordHash, &roleStr,
	)
	u.Role = domain.UserRole(roleStr)
	return u, err
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (ports.UserRecord, error) {
	const q = `SELECT id, household_id, email, password_hash, role FROM users WHERE email = $1`
	var u ports.UserRecord
	var roleStr string
	err := r.Pool.QueryRow(ctx, q, email).Scan(&u.ID, &u.HouseholdID, &u.Email, &u.PasswordHash, &roleStr)
	if errors.Is(err, pgx.ErrNoRows) {
		return ports.UserRecord{}, domain.ErrNotFound
	}
	u.Role = domain.UserRole(roleStr)
	return u, err
}

func (r *UserRepo) GetByID(ctx context.Context, id domain.UserID) (ports.UserRecord, error) {
	const q = `SELECT id, household_id, email, password_hash, role FROM users WHERE id = $1`
	var u ports.UserRecord
	var roleStr string
	err := r.Pool.QueryRow(ctx, q, id).Scan(&u.ID, &u.HouseholdID, &u.Email, &u.PasswordHash, &roleStr)
	if errors.Is(err, pgx.ErrNoRows) {
		return ports.UserRecord{}, domain.ErrNotFound
	}
	u.Role = domain.UserRole(roleStr)
	return u, err
}

// SessionRepo implements ports.SessionRepository.
type SessionRepo struct{ Pool *pgxpool.Pool }

func (r *SessionRepo) Create(ctx context.Context, s ports.SessionRecord) error {
	const q = `
		INSERT INTO sessions (id, user_id, token_hash, csrf_secret, expires_at, idle_expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)`
	id := s.ID
	if id == uuid.Nil {
		id = uuid.New()
	}
	_, err := r.Pool.Exec(ctx, q, id, s.UserID, s.TokenHash, s.CSRFSecret, s.ExpiresAt, s.IdleExpiresAt)
	return err
}

func (r *SessionRepo) GetByTokenHash(ctx context.Context, tokenHash string) (ports.SessionRecord, error) {
	const q = `
		SELECT id, user_id, token_hash, csrf_secret, expires_at, idle_expires_at
		FROM sessions WHERE token_hash = $1`
	var s ports.SessionRecord
	err := r.Pool.QueryRow(ctx, q, tokenHash).Scan(
		&s.ID, &s.UserID, &s.TokenHash, &s.CSRFSecret, &s.ExpiresAt, &s.IdleExpiresAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return ports.SessionRecord{}, domain.ErrNotFound
	}
	return s, err
}

func (r *SessionRepo) TouchIdle(ctx context.Context, id domain.SessionID, idleExpiresAt time.Time) error {
	_, err := r.Pool.Exec(ctx, `UPDATE sessions SET idle_expires_at = $2 WHERE id = $1`, id, idleExpiresAt)
	return err
}

func (r *SessionRepo) Delete(ctx context.Context, id domain.SessionID) error {
	_, err := r.Pool.Exec(ctx, `DELETE FROM sessions WHERE id = $1`, id)
	return err
}

// APITokenRepo implements ports.APITokenRepository.
type APITokenRepo struct{ Pool *pgxpool.Pool }

func (r *APITokenRepo) Create(ctx context.Context, t ports.APITokenRecord) error {
	id := t.ID
	if id == uuid.Nil {
		id = uuid.New()
	}
	const q = `
		INSERT INTO api_tokens (id, user_id, name, token_prefix, token_hash, scopes, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := r.Pool.Exec(ctx, q, id, t.UserID, t.Name, t.Prefix, t.TokenHash, string(t.Scopes), t.ExpiresAt)
	return err
}

func (r *APITokenRepo) GetByTokenHash(ctx context.Context, tokenHash string) (ports.APITokenRecord, error) {
	const q = `
		SELECT id, user_id, name, token_prefix, token_hash, scopes, expires_at, revoked_at
		FROM api_tokens WHERE token_hash = $1`
	var t ports.APITokenRecord
	var scope string
	err := r.Pool.QueryRow(ctx, q, tokenHash).Scan(
		&t.ID, &t.UserID, &t.Name, &t.Prefix, &t.TokenHash, &scope, &t.ExpiresAt, &t.RevokedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return ports.APITokenRecord{}, domain.ErrNotFound
	}
	t.Scopes = domain.TokenScope(scope)
	return t, err
}

func (r *APITokenRepo) ListByUser(ctx context.Context, userID domain.UserID) ([]ports.APITokenRecord, error) {
	const q = `
		SELECT id, user_id, name, token_prefix, token_hash, scopes, expires_at, revoked_at
		FROM api_tokens WHERE user_id = $1 AND revoked_at IS NULL ORDER BY created_at DESC`
	rows, err := r.Pool.Query(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ports.APITokenRecord
	for rows.Next() {
		var t ports.APITokenRecord
		var scope string
		if err := rows.Scan(&t.ID, &t.UserID, &t.Name, &t.Prefix, &t.TokenHash, &scope, &t.ExpiresAt, &t.RevokedAt); err != nil {
			return nil, err
		}
		t.Scopes = domain.TokenScope(scope)
		out = append(out, t)
	}
	return out, rows.Err()
}

func (r *APITokenRepo) Revoke(ctx context.Context, userID domain.UserID, id domain.APITokenID) error {
	tag, err := r.Pool.Exec(ctx, `
		UPDATE api_tokens SET revoked_at = now() WHERE id = $1 AND user_id = $2 AND revoked_at IS NULL`, id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
