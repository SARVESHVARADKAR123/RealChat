package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/SARVESHVARADKAR123/RealChat/services/message/internal/domain"
)

type Repository interface {
	// Messaging
	InsertMessage(ctx context.Context, tx *sql.Tx, msg *domain.Message) error
	MarkMessageDeleted(ctx context.Context, tx *sql.Tx, msgID string) error
	GetMessageForUpdate(ctx context.Context, tx *sql.Tx, messageID string) (*domain.Message, error)
	FetchMessages(ctx context.Context, convID string, lastSeq int64, limit int) ([]*domain.Message, error)

	// Idempotency
	TryInsertIdempotency(ctx context.Context, tx *sql.Tx, key, userID, conversationID string, expiresAt time.Time) (bool, error)
	GetIdempotencyForUpdate(ctx context.Context, tx *sql.Tx, key, userID, conversationID string) ([]byte, error)
	UpdateIdempotencyResponse(ctx context.Context, tx *sql.Tx, key, userID, conversationID string, payload []byte) error

	// Outbox
	InsertOutbox(ctx context.Context, tx *sql.Tx, aggregateType, aggregateID, eventType string, payload []byte) error
}
