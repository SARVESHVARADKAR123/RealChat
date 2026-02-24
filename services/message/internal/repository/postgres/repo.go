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

func (r *Repository) InsertParticipant(
	ctx context.Context,
	tx *sql.Tx,
	convID, userID string,
	role domain.Role,
) error {
	q := r.getter(tx)
	_, err := q.ExecContext(ctx, `
		INSERT INTO conversation_participants (conversation_id, user_id, role)
		VALUES ($1, $2, $3)
	`, convID, userID, role)
	return err
}

func (r *Repository) DeleteParticipant(
	ctx context.Context,
	tx *sql.Tx,
	convID, userID string,
) error {
	q := r.getter(tx)
	_, err := q.ExecContext(ctx, `
		DELETE FROM conversation_participants
		WHERE conversation_id = $1 AND user_id = $2
	`, convID, userID)
	return err
}

func (r *Repository) NextSequence(
	ctx context.Context,
	tx *sql.Tx,
	convID string,
) (int64, error) {

	var next int64

	q := r.getter(tx)
	err := q.QueryRowContext(ctx, `
		UPDATE conversation_sequences
		SET next_sequence = next_sequence + 1
		WHERE conversation_id = $1
		RETURNING next_sequence
	`, convID).Scan(&next)

	if err != nil {
		return 0, err
	}

	// next_sequence starts at 0 and we return the post-increment value as the message sequence.
	return next, nil
}

// Idempotency
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

func (r *Repository) UpdateLastReadSequence(
	ctx context.Context,
	tx *sql.Tx,
	convID string,
	userID string,
	sequence int64,
) error {

	q := r.getter(tx)
	_, err := q.ExecContext(ctx, `
		UPDATE conversation_participants
		SET last_read_sequence = GREATEST(last_read_sequence, $3)
		WHERE conversation_id = $1
		  AND user_id = $2
	`, convID, userID, sequence)

	return err
}

func (r *Repository) GetCurrentMaxSequence(
	ctx context.Context,
	tx *sql.Tx,
	convID string,
) (int64, error) {

	var maxSeq sql.NullInt64

	q := r.getter(tx)
	err := q.QueryRowContext(ctx, `
		SELECT next_sequence - 1
		FROM conversation_sequences
		WHERE conversation_id = $1
	`, convID).Scan(&maxSeq)

	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, err
	}

	if !maxSeq.Valid {
		return 0, nil
	}

	return maxSeq.Int64, nil
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

func (r *Repository) GetConversationLocked(
	ctx context.Context,
	tx *sql.Tx,
	convID string,
) (*domain.Conversation, error) {
	// 1. Get Conversation (FOR UPDATE)
	return r.fetchConversation(ctx, tx, convID, true)
}

func (r *Repository) GetConversation(
	ctx context.Context,
	tx *sql.Tx,
	convID string,
) (*domain.Conversation, error) {
	// 1. Try Cache
	if r.Cache != nil {
		conv, err := r.Cache.GetConversation(ctx, convID)
		if err == nil && conv != nil {
			return conv, nil
		}
	}

	// 2. Fallback to DB (No Lock)
	conv, err := r.fetchConversation(ctx, tx, convID, false)
	if err != nil {
		return nil, err
	}

	// 3. Populate Cache
	if r.Cache != nil {
		_ = r.Cache.SetConversation(ctx, conv)
	}

	return conv, nil
}

func (r *Repository) ListConversationsByUser(
	ctx context.Context,
	userID string,
) ([]*domain.Conversation, error) {
	rows, err := r.DB.QueryContext(ctx, `
		SELECT c.id, c.display_name, c.avatar_url, c.type, c.created_at
		FROM conversations c
		JOIN conversation_participants cp ON c.id = cp.conversation_id
		WHERE cp.user_id = $1
		ORDER BY c.updated_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var conversations []*domain.Conversation
	for rows.Next() {
		var c domain.Conversation
		var displayName, avatarURL sql.NullString
		if err := rows.Scan(
			&c.ID,
			&displayName,
			&avatarURL,
			&c.Type,
			&c.CreatedAt,
		); err != nil {
			return nil, err
		}
		c.DisplayName = displayName.String
		c.AvatarURL = avatarURL.String
		conversations = append(conversations, &c)
	}

	return conversations, nil
}

func (r *Repository) InvalidateConversation(
	ctx context.Context,
	convID string,
) error {
	if r.Cache != nil {
		return r.Cache.DeleteConversation(ctx, convID)
	}
	return nil
}

func (r *Repository) fetchConversation(
	ctx context.Context,
	tx *sql.Tx,
	convID string,
	forUpdate bool,
) (*domain.Conversation, error) {
	query := `
		SELECT id, type, display_name, avatar_url, created_at
		FROM conversations
		WHERE id = $1
	`
	if forUpdate {
		query += " FOR UPDATE"
	}

	q := r.getter(tx)

	// 1. Get Conversation
	var conv domain.Conversation
	var displayName, avatarURL sql.NullString
	err := q.QueryRowContext(ctx, query, convID).Scan(
		&conv.ID,
		&conv.Type,
		&displayName,
		&avatarURL,
		&conv.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrConversationNotFound
		}
		return nil, err
	}
	conv.DisplayName = displayName.String
	conv.AvatarURL = avatarURL.String

	// 2. Get Participants
	rows, err := q.QueryContext(ctx, `
		SELECT user_id, role
		FROM conversation_participants
		WHERE conversation_id = $1
	`, convID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	conv.Participants = make(map[string]domain.Participant)
	for rows.Next() {
		var p domain.Participant
		if err := rows.Scan(&p.UserID, &p.Role); err != nil {
			return nil, err
		}
		conv.Participants[p.UserID] = p
	}

	return &conv, nil
}

func (r *Repository) GetConversationByLookupKey(
	ctx context.Context,
	tx *sql.Tx,
	key string,
) (*domain.Conversation, error) {
	q := r.getter(tx)
	var id string
	err := q.QueryRowContext(ctx, `
		SELECT id FROM conversations WHERE lookup_key = $1
	`, key).Scan(&id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrConversationNotFound
		}
		return nil, err
	}
	return r.fetchConversation(ctx, tx, id, false)
}

func (r *Repository) InsertConversation(
	ctx context.Context,
	tx *sql.Tx,
	id string,
	convType domain.ConversationType,
	name, avatar string,
	lookupKey *string,
) error {
	q := r.getter(tx)
	_, err := q.ExecContext(ctx, `
		INSERT INTO conversations (id, type, display_name, avatar_url, lookup_key)
		VALUES ($1, $2, $3, $4, $5)
	`, id, convType, name, avatar, lookupKey)
	return err
}

func (r *Repository) InitSequence(
	ctx context.Context,
	tx *sql.Tx,
	id string,
) error {
	// Starts at 0; NextSequence increments first then returns the new value,
	// so the first message gets sequence number 1.
	q := r.getter(tx)
	_, err := q.ExecContext(ctx, `
		INSERT INTO conversation_sequences (conversation_id, next_sequence)
		VALUES ($1, 0)
	`, id)
	return err
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
	// Returns true (owned) only when THIS call inserted the row.
	// ON CONFLICT DO NOTHING means a duplicate returns 0 RowsAffected.
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
			return nil, nil // Not found is nil, nil? Or error? Application expects nil payload if key exists but processing?
			// Actually TryInsert should be called first. If Get is called, it means we found the key.
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
