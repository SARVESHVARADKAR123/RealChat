package repository

import (
	"context"
	"database/sql"
)

// BlockRepo handles CRUD operations on the blocks table.
type BlockRepo struct{ DB *sql.DB }

// Add inserts a block relationship (idempotent via ON CONFLICT DO NOTHING).
func (r *BlockRepo) Add(ctx context.Context, userID, blockedUserID string) error {
	_, err := r.DB.ExecContext(ctx,
		`INSERT INTO blocks (user_id, blocked_user_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		userID, blockedUserID)
	return err
}

// Remove deletes a block relationship.
func (r *BlockRepo) Remove(ctx context.Context, userID, blockedUserID string) error {
	_, err := r.DB.ExecContext(ctx,
		`DELETE FROM blocks WHERE user_id = $1 AND blocked_user_id = $2`,
		userID, blockedUserID)
	return err
}

// Exists checks whether a block relationship exists between two users.
func (r *BlockRepo) Exists(ctx context.Context, userID, blockedUserID string) (bool, error) {
	var ok bool
	err := r.DB.QueryRowContext(ctx,
		`SELECT TRUE FROM blocks WHERE user_id = $1 AND blocked_user_id = $2`,
		userID, blockedUserID).Scan(&ok)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return ok, err
}
