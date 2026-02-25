package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/SARVESHVARADKAR123/RealChat/services/message/internal/cache"
	"github.com/SARVESHVARADKAR123/RealChat/services/message/internal/domain"
)

type Repository struct {
	DB    *sql.DB
	Cache *cache.Cache
}

type queryable interface {
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
}

func (r *Repository) getter(tx *sql.Tx) queryable {
	if tx != nil {
		return tx
	}
	return r.DB
}

func (r *Repository) InsertMessage(
	ctx context.Context,
	tx *sql.Tx,
	msg *domain.Message,
) error {
	var metadata interface{}
	if msg.Metadata == "" {
		metadata = nil
	} else {
		metadata = msg.Metadata
	}

	q := r.getter(tx)
	_, err := q.ExecContext(ctx, `
		INSERT INTO messages (
			id, conversation_id, sender_id,
			sequence, type, content, metadata, sent_at
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
	`,
		msg.ID,
		msg.ConversationID,
		msg.SenderID,
		msg.Sequence,
		msg.Type,
		msg.Content,
		metadata,
		msg.SentAt,
	)

	return err
}

func (r *Repository) FetchMessages(
	ctx context.Context,
	convID string,
	lastSeq int64,
	limit int,
) ([]*domain.Message, error) {

	rows, err := r.DB.QueryContext(ctx, `
		SELECT id, conversation_id, sender_id, sequence,
		       type, content, metadata, sent_at, deleted_at
		FROM messages
		WHERE conversation_id = $1
		  AND sequence > $2
		ORDER BY sequence ASC
		LIMIT $3
	`, convID, lastSeq, limit)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []*domain.Message

	for rows.Next() {
		var msg domain.Message
		var metadata sql.NullString
		var deletedAt sql.NullTime
		if err := rows.Scan(
			&msg.ID,
			&msg.ConversationID,
			&msg.SenderID,
			&msg.Sequence,
			&msg.Type,
			&msg.Content,
			&metadata,
			&msg.SentAt,
			&deletedAt,
		); err != nil {
			return nil, err
		}
		msg.Metadata = metadata.String
		if deletedAt.Valid {
			msg.DeletedAt = &deletedAt.Time
		}
		messages = append(messages, &msg)
	}

	return messages, nil
}

func (r *Repository) GetMessageForUpdate(
	ctx context.Context,
	tx *sql.Tx,
	messageID string,
) (*domain.Message, error) {

	q := r.getter(tx)
	row := q.QueryRowContext(ctx, `
		SELECT id, conversation_id, sender_id, sequence,
		       type, content, metadata, sent_at, deleted_at
		FROM messages
		WHERE id = $1
		FOR UPDATE
	`, messageID)

	var msg domain.Message
	var metadata sql.NullString
	var deletedAt sql.NullTime

	err := row.Scan(
		&msg.ID,
		&msg.ConversationID,
		&msg.SenderID,
		&msg.Sequence,
		&msg.Type,
		&msg.Content,
		&metadata,
		&msg.SentAt,
		&deletedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrMessageNotFound
		}
		return nil, err
	}
	msg.Metadata = metadata.String
	if deletedAt.Valid {
		msg.DeletedAt = &deletedAt.Time
	}

	return &msg, nil
}

func (r *Repository) MarkMessageDeleted(
	ctx context.Context,
	tx *sql.Tx,
	msgID string,
) error {
	q := r.getter(tx)
	_, err := q.ExecContext(ctx, `
		UPDATE messages
		SET deleted_at = NOW()
		WHERE id = $1
	`, msgID)
	return err
}

func (r *Repository) TryInsertIdempotency(
	ctx context.Context,
	tx *sql.Tx,
	key, userID, conversationID string,
	expiresAt time.Time,
) (bool, error) {
	q := r.getter(tx)
	result, err := q.ExecContext(ctx, `
		INSERT INTO idempotency_keys (key, user_id, conversation_id, expires_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (key, user_id, conversation_id) DO NOTHING
	`, key, userID, conversationID, expiresAt)
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return n == 1, nil
}

func (r *Repository) GetIdempotencyForUpdate(
	ctx context.Context,
	tx *sql.Tx,
	key, userID, conversationID string,
) ([]byte, error) {
	q := r.getter(tx)
	var payload []byte
	err := q.QueryRowContext(ctx, `
        SELECT payload
        FROM idempotency_keys
        WHERE key = $1 AND user_id = $2 AND conversation_id = $3
        FOR UPDATE
    `, key, userID, conversationID).Scan(&payload)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return payload, nil
}

func (r *Repository) UpdateIdempotencyResponse(
	ctx context.Context,
	tx *sql.Tx,
	key, userID, conversationID string,
	payload []byte,
) error {
	q := r.getter(tx)
	_, err := q.ExecContext(ctx, `
        UPDATE idempotency_keys
        SET payload = $4
        WHERE key = $1 AND user_id = $2 AND conversation_id = $3
    `, key, userID, conversationID, payload)
	return err
}

func (r *Repository) InsertOutbox(
	ctx context.Context,
	tx *sql.Tx,
	aggregateType, aggregateID, eventType string,
	payload []byte,
) error {
	q := r.getter(tx)
	_, err := q.ExecContext(ctx, `
        INSERT INTO outbox_events (aggregate_type, aggregate_id, event_type, payload)
        VALUES ($1, $2, $3, $4)
    `, aggregateType, aggregateID, eventType, payload)
	return err
}
