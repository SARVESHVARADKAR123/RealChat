package tx

import (
	"context"
	"database/sql"
	"errors"
	"strings"
)

type Manager struct {
	DB *sql.DB
}

const maxRetries = 5

func (m *Manager) WithTx(
	ctx context.Context,
	fn func(ctx context.Context, tx *sql.Tx) error,
) error {

	for i := 0; i < maxRetries; i++ {

		tx, err := m.DB.BeginTx(ctx, &sql.TxOptions{
			Isolation: sql.LevelReadCommitted, // For auth, ReadCommitted is usually enough unless we have complex races
		})
		if err != nil {
			return err
		}

		err = fn(ctx, tx)
		if err != nil {
			tx.Rollback()
			if isSerializationError(err) {
				continue
			}
			return err
		}

		if err := tx.Commit(); err != nil {
			if isSerializationError(err) {
				continue
			}
			return err
		}

		return nil
	}

	return errors.New("transaction retry exhausted")
}

func isSerializationError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "could not serialize")
}
