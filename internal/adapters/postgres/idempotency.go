package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/krishnamadhavan/expense-tracker/internal/domain"
)

// IdempotencyRepo stores POST /transactions idempotency keys.
type IdempotencyRepo struct{ Pool *pgxpool.Pool }

// StoredResponse is a cached successful (or error) response for a key.
type StoredResponse struct {
	RequestHash  string
	ResponseCode int
	ResponseBody []byte
}

func (r *IdempotencyRepo) Get(ctx context.Context, userID domain.UserID, key string) (StoredResponse, error) {
	const q = `SELECT request_hash, response_code, response_body FROM idempotency_keys WHERE user_id = $1 AND key = $2`
	var s StoredResponse
	err := r.Pool.QueryRow(ctx, q, userID, key).Scan(&s.RequestHash, &s.ResponseCode, &s.ResponseBody)
	if errors.Is(err, pgx.ErrNoRows) {
		return StoredResponse{}, domain.ErrNotFound
	}
	return s, err
}

func (r *IdempotencyRepo) Put(ctx context.Context, userID domain.UserID, key, requestHash string, code int, body []byte) error {
	const q = `
		INSERT INTO idempotency_keys (user_id, key, request_hash, response_code, response_body)
		VALUES ($1, $2, $3, $4, $5::jsonb)
		ON CONFLICT (user_id, key) DO NOTHING`
	_, err := r.Pool.Exec(ctx, q, userID, key, requestHash, code, body)
	return err
}
