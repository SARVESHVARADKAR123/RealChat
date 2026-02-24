package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/SARVESHVARADKAR123/RealChat/services/auth/internal/domain"
	"github.com/lib/pq"
)

type AuthRepository struct {
	db *sql.DB
}

func NewAuthRepository(db *sql.DB) *AuthRepository {
	return &AuthRepository{db: db}
}

func (r *AuthRepository) CreateUser(ctx context.Context, tx *sql.Tx, id, email, hash string) error {
	var err error
	if tx != nil {
		_, err = tx.ExecContext(ctx,
			`INSERT INTO users (id, email, password_hash) VALUES ($1,$2,$3)`,
			id, email, hash,
		)
	} else {
		_, err = r.db.ExecContext(ctx,
			`INSERT INTO users (id, email, password_hash) VALUES ($1,$2,$3)`,
			id, email, hash,
		)
	}
	if err != nil {
		if pgErr, ok := err.(*pq.Error); ok && pgErr.Code == "23505" {
			return domain.ErrEmailConflict
		}
		return err
	}
	return nil
}

func (r *AuthRepository) GetUserByEmail(ctx context.Context, email string) (string, string, error) {
	var id, hash string
	err := r.db.QueryRowContext(ctx,
		`SELECT id, password_hash FROM users WHERE email=$1`, email,
	).Scan(&id, &hash)
	if err == sql.ErrNoRows {
		return "", "", domain.ErrUserNotFound
	}
	return id, hash, err
}

func (r *AuthRepository) SaveRefresh(ctx context.Context, id, userID, tokenHash string, exp time.Time) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at) VALUES ($1,$2,$3,$4)`,
		id, userID, tokenHash, exp,
	)
	return err
}

func (r *AuthRepository) GetRefresh(ctx context.Context, tokenHash string) (string, time.Time, error) {
	var uid string
	var exp time.Time
	err := r.db.QueryRowContext(ctx,
		`SELECT user_id, expires_at FROM refresh_tokens WHERE token_hash=$1 AND revoked_at IS NULL`,
		tokenHash,
	).Scan(&uid, &exp)
	if err == sql.ErrNoRows {
		return "", time.Time{}, domain.ErrInvalidToken
	}
	return uid, exp, err
}

func (r *AuthRepository) RevokeRefresh(ctx context.Context, tokenHash string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE refresh_tokens SET revoked_at=NOW() WHERE token_hash=$1`,
		tokenHash,
	)
	return err
}

func (r *AuthRepository) InsertOutbox(ctx context.Context, tx *sql.Tx, aggregateType, aggregateID, eventType string, payload []byte) error {
	query := `INSERT INTO outbox_events (aggregate_type, aggregate_id, event_type, payload) VALUES ($1, $2, $3, $4)`
	var err error
	if tx != nil {
		_, err = tx.ExecContext(ctx, query, aggregateType, aggregateID, eventType, payload)
	} else {
		_, err = r.db.ExecContext(ctx, query, aggregateType, aggregateID, eventType, payload)
	}
	return err
}
