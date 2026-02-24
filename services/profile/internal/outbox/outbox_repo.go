package outbox

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
)

// OutboxRow represents a single unpublished outbox entry.
type OutboxRow struct {
	ID      string
	Topic   string
	Key     string
	Payload []byte
}

// Repository manages the transactional outbox table.
type Repository struct{ DB *sql.DB }

// NewRepository returns an outbox repository backed by the given DB.
func NewRepository(db *sql.DB) *Repository { return &Repository{DB: db} }

// InsertTx inserts a new outbox event inside an existing transaction.
func (r *Repository) InsertTx(ctx context.Context, tx *sql.Tx, topic, key string, payload []byte) error {
	_, err := tx.ExecContext(ctx,
		`INSERT INTO outbox (id, topic, key, payload) VALUES ($1,$2,$3,$4)`,
		uuid.NewString(), topic, key, payload)
	return err
}



// Add inserts a new outbox event outside a transaction.  Services that do not
// need transactional guarantees can use this convenience method.
func (r *Repository) Add(ctx context.Context, eventType string, payload []byte) error {
	_, err := r.DB.ExecContext(ctx,
		`INSERT INTO outbox (id, topic, key, payload) VALUES ($1,$2,$3,$4)`,
		uuid.NewString(), eventType, uuid.NewString(), payload)
	return err
}

// Fetch returns up to `limit` unpublished outbox rows ordered by creation time.
func (r *Repository) Fetch(ctx context.Context, limit int) ([]OutboxRow, error) {
	rows, err := r.DB.QueryContext(ctx,
		`SELECT id, topic, key, payload FROM outbox WHERE published_at IS NULL ORDER BY created_at LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []OutboxRow
	for rows.Next() {
		var row OutboxRow
		if err := rows.Scan(&row.ID, &row.Topic, &row.Key, &row.Payload); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

// MarkPublished sets the published_at timestamp for the given outbox row.
func (r *Repository) MarkPublished(ctx context.Context, id string) error {
	_, err := r.DB.ExecContext(ctx,
		`UPDATE outbox SET published_at = NOW() WHERE id = $1`, id)
	return err
}
