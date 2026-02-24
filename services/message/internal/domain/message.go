package domain

import "time"

const MaxMessageSize = 5000

// Message Invariants:
// 1. Ordering: Sequence must be strictly increasing, gapless, and unique per conversation_id.
// 2. Immutability: Standard fields are immutable. Only DeletedAt can change.
// 3. Event Consistency: Creation must emit a MessageSentEvent.
type Message struct {
	ID             string
	ConversationID string
	SenderID       string
	Sequence       int64 // Monotonic, gapless, unique per ConversationID
	Type           string
	Content        string
	Metadata       string
	SentAt         time.Time
	DeletedAt      *time.Time
}

func NewMessage(
	id string,
	conversationID string,
	senderID string,
	sequence int64,
	msgType string,
	content string,
	metadata string,
	now time.Time,
) (*Message, error) {

	if id == "" || conversationID == "" || senderID == "" {
		return nil, ErrInvalidMessage
	}

	if sequence <= 0 {
		return nil, ErrInvalidSequence
	}

	if len(content) > MaxMessageSize {
		return nil, ErrMessageTooLarge
	}

	return &Message{
		ID:             id,
		ConversationID: conversationID,
		SenderID:       senderID,
		Sequence:       sequence,
		Type:           msgType,
		Content:        content,
		Metadata:       metadata,
		SentAt:         now,
	}, nil
}
