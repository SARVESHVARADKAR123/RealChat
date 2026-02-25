package repository

import (
	"context"
	"database/sql"

	"github.com/SARVESHVARADKAR123/RealChat/services/conversation/internal/domain"
)

type Repository interface {
	// Conversation Lifecycle
	InsertConversation(ctx context.Context, tx *sql.Tx, id string, convType domain.ConversationType, name, avatar string, lookupKey *string) error

	GetConversationByLookupKey(ctx context.Context, tx *sql.Tx, key string) (*domain.Conversation, error)

	// GetConversation (ReadOnly/Cached) - Best Effort consistency
	GetConversation(ctx context.Context, tx *sql.Tx, convID string) (*domain.Conversation, error)

	// GetConversationLocked (Write/Strict) - SELECT ... FOR UPDATE
	GetConversationLocked(ctx context.Context, tx *sql.Tx, convID string) (*domain.Conversation, error)

	// InvalidateConversation (Cache)
	InvalidateConversation(ctx context.Context, convID string) error

	InitSequence(ctx context.Context, tx *sql.Tx, id string) error
	NextSequence(ctx context.Context, tx *sql.Tx, convID string) (int64, error)

	ListConversationsByUser(ctx context.Context, userID string) ([]*domain.Conversation, error)
	InsertParticipant(ctx context.Context, tx *sql.Tx, convID, userID string, role domain.Role) error
	DeleteParticipant(ctx context.Context, tx *sql.Tx, convID, userID string) error

	UpdateLastReadSequence(ctx context.Context, tx *sql.Tx, convID, userID string, seq int64) error
	GetCurrentMaxSequence(ctx context.Context, tx *sql.Tx, convID string) (int64, error)

	InsertOutbox(ctx context.Context, tx *sql.Tx, aggregateType, aggregateID, eventType string, payload []byte) error
}
