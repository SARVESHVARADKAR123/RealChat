package application

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/SARVESHVARADKAR123/RealChat/services/message/internal/domain"
)

func TestIdempotencyScoping(t *testing.T) {
	ctx := context.Background()

	// Setup mock repo that supports idempotency tracking
	repo := &idempotencyMockRepo{
		idempotency: make(map[string][]byte),
		conversations: map[string]*domain.Conversation{
			"conv-1": {
				ID: "conv-1",
				Participants: map[string]domain.Participant{
					"user-1": {UserID: "user-1", Role: domain.RoleMember},
				},
			},
			"conv-2": {
				ID: "conv-2",
				Participants: map[string]domain.Participant{
					"user-1": {UserID: "user-1", Role: domain.RoleMember},
				},
			},
		},
	}
	// We don't actually use the DB in the mock repo calls, but SendMessage requires a tx.Manager.
	// However, WithTx will try to call DB.BeginTx.
	// So we need a better way to mock or just use a simple mock that works with the interface if it were one.
	// Since tx.Manager.WithTx is NOT an interface, we must provide a real one.
	// Let's modify the test to NOT rely on tx.Manager if possible, or just mock the repo methods to not need tx.

	s := &Service{repo: repo, tx: &mockTransactor{}} // This will panic if WithTx is called.

	clientMsgID := "shared-id"
	userID := "user-1"

	// 1. Send to Conversation 1
	cmd1 := SendMessageCommand{
		ConversationID: "conv-1",
		UserID:         userID,
		ClientMsgID:    clientMsgID,
		Type:           "text",
		Content:        "Hello 1",
	}
	msg1, err := s.SendMessage(ctx, cmd1)
	if err != nil {
		t.Fatalf("first message failed: %v", err)
	}

	// 2. Send same ClientMsgID to Conversation 2
	cmd2 := SendMessageCommand{
		ConversationID: "conv-2",
		UserID:         userID,
		ClientMsgID:    clientMsgID,
		Type:           "text",
		Content:        "Hello 2",
	}
	msg2, err := s.SendMessage(ctx, cmd2)
	if err != nil {
		t.Fatalf("second message (different conversation) failed: %v", err)
	}

	if msg1.ID == msg2.ID {
		t.Errorf("expected different message IDs, but got same: %s", msg1.ID)
	}

	if msg2.Content != "Hello 2" {
		t.Errorf("expected second message content 'Hello 2', got %s", msg2.Content)
	}

	// 3. Send same ClientMsgID to Conversation 1 AGAIN (should be idempotent)
	msg1Retry, err := s.SendMessage(ctx, cmd1)
	if err != nil {
		t.Fatalf("retry of first message failed: %v", err)
	}
	if msg1Retry.ID != msg1.ID {
		t.Errorf("expected same message ID for retry, but got %s != %s", msg1Retry.ID, msg1.ID)
	}
}

type idempotencyMockRepo struct {
	mockSyncRepo
	idempotency   map[string][]byte
	conversations map[string]*domain.Conversation
}

func (m *idempotencyMockRepo) GetConversation(ctx context.Context, tx *sql.Tx, convID string) (*domain.Conversation, error) {
	c, ok := m.conversations[convID]
	if !ok {
		return nil, domain.ErrConversationNotFound
	}
	return c, nil
}

func (m *idempotencyMockRepo) TryInsertIdempotency(ctx context.Context, tx *sql.Tx, key, userID, conversationID string, expiresAt time.Time) (bool, error) {
	fullKey := key + ":" + userID + ":" + conversationID
	if _, ok := m.idempotency[fullKey]; ok {
		return false, nil
	}
	m.idempotency[fullKey] = nil
	return true, nil
}

func (m *idempotencyMockRepo) GetIdempotencyForUpdate(ctx context.Context, tx *sql.Tx, key, userID, conversationID string) ([]byte, error) {
	fullKey := key + ":" + userID + ":" + conversationID
	return m.idempotency[fullKey], nil
}

func (m *idempotencyMockRepo) UpdateIdempotencyResponse(ctx context.Context, tx *sql.Tx, key, userID, conversationID string, payload []byte) error {
	fullKey := key + ":" + userID + ":" + conversationID
	m.idempotency[fullKey] = payload
	return nil
}

func (m *idempotencyMockRepo) NextSequence(ctx context.Context, tx *sql.Tx, convID string) (int64, error) {
	return 1, nil
}

func (m *idempotencyMockRepo) InsertMessage(ctx context.Context, tx *sql.Tx, msg *domain.Message) error {
	return nil
}

func (m *idempotencyMockRepo) InsertOutbox(ctx context.Context, tx *sql.Tx, aggregateType, aggregateID, eventType string, payload []byte) error {
	return nil
}

func (m *idempotencyMockRepo) InsertConversation(ctx context.Context, tx *sql.Tx, id string, convType domain.ConversationType, name, avatar string, lookupKey *string) error {
	return nil
}

func (m *idempotencyMockRepo) GetConversationByLookupKey(ctx context.Context, tx *sql.Tx, key string) (*domain.Conversation, error) {
	return nil, domain.ErrConversationNotFound
}

type mockTransactor struct{}

func (m *mockTransactor) WithTx(ctx context.Context, fn func(context.Context, *sql.Tx) error) error {
	return fn(ctx, nil)
}
