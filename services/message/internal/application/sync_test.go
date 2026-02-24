package application

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/SARVESHVARADKAR123/RealChat/services/message/internal/domain"
)

type mockSyncRepo struct {
	conv *domain.Conversation
}

func (m *mockSyncRepo) InsertConversation(ctx context.Context, tx *sql.Tx, id string, convType domain.ConversationType, name, avatar string, lookupKey *string) error {
	return nil
}
func (m *mockSyncRepo) GetConversationByLookupKey(ctx context.Context, tx *sql.Tx, key string) (*domain.Conversation, error) {
	return nil, errors.New("not found")
}
func (m *mockSyncRepo) GetConversation(ctx context.Context, tx *sql.Tx, convID string) (*domain.Conversation, error) {
	if m.conv == nil {
		return nil, errors.New("not found")
	}
	if m.conv.ID != convID {
		return nil, errors.New("not found")
	}
	return m.conv, nil
}
func (m *mockSyncRepo) GetConversationLocked(ctx context.Context, tx *sql.Tx, convID string) (*domain.Conversation, error) {
	return nil, nil
}
func (m *mockSyncRepo) InvalidateConversation(ctx context.Context, convID string) error {
	return nil
}
func (m *mockSyncRepo) InitSequence(ctx context.Context, tx *sql.Tx, id string) error {
	return nil
}
func (m *mockSyncRepo) GetMessageForUpdate(ctx context.Context, tx *sql.Tx, messageID string) (*domain.Message, error) {
	return nil, nil
}
func (m *mockSyncRepo) ListConversationsByUser(ctx context.Context, userID string) ([]*domain.Conversation, error) {
	return nil, nil
}

func (m *mockSyncRepo) InsertParticipant(ctx context.Context, tx *sql.Tx, convID, userID string, role domain.Role) error {
	return nil
}
func (m *mockSyncRepo) DeleteParticipant(ctx context.Context, tx *sql.Tx, convID, userID string) error {
	return nil
}
func (m *mockSyncRepo) NextSequence(ctx context.Context, tx *sql.Tx, convID string) (int64, error) {
	return 0, nil
}
func (m *mockSyncRepo) InsertMessage(ctx context.Context, tx *sql.Tx, msg *domain.Message) error {
	return nil
}
func (m *mockSyncRepo) MarkMessageDeleted(ctx context.Context, tx *sql.Tx, msgID string) error {
	return nil
}
func (m *mockSyncRepo) UpdateLastReadSequence(ctx context.Context, tx *sql.Tx, convID, userID string, seq int64) error {
	return nil
}
func (m *mockSyncRepo) GetCurrentMaxSequence(ctx context.Context, tx *sql.Tx, convID string) (int64, error) {
	return 0, nil
}
func (m *mockSyncRepo) TryInsertIdempotency(ctx context.Context, tx *sql.Tx, key, userID, conversationID string, expiresAt time.Time) (bool, error) {
	return true, nil
}
func (m *mockSyncRepo) GetIdempotencyForUpdate(ctx context.Context, tx *sql.Tx, key, userID, conversationID string) ([]byte, error) {
	return nil, nil
}
func (m *mockSyncRepo) UpdateIdempotencyResponse(ctx context.Context, tx *sql.Tx, key, userID, conversationID string, payload []byte) error {
	return nil
}
func (m *mockSyncRepo) InsertOutbox(ctx context.Context, tx *sql.Tx, aggregateType, aggregateID, eventType string, payload []byte) error {
	return nil
}
func (m *mockSyncRepo) FetchMessages(ctx context.Context, convID string, lastSeq int64, limit int) ([]*domain.Message, error) {
	return []*domain.Message{}, nil
}

func TestSyncMessages(t *testing.T) {
	ctx := context.Background()

	t.Run("success_when_member", func(t *testing.T) {
		repo := &mockSyncRepo{
			conv: &domain.Conversation{
				ID: "conv-1",
				Participants: map[string]domain.Participant{
					"user-1": {UserID: "user-1", Role: domain.RoleMember},
				},
			},
		}
		s := &Service{repo: repo}

		_, err := s.SyncMessages(ctx, "conv-1", "user-1", 0, 10)
		if err != nil {
			t.Errorf("expected success, got %v", err)
		}
	})

	t.Run("failure_when_not_member", func(t *testing.T) {
		repo := &mockSyncRepo{
			conv: &domain.Conversation{
				ID: "conv-1",
				Participants: map[string]domain.Participant{
					"user-1": {UserID: "user-1", Role: domain.RoleMember},
				},
			},
		}
		s := &Service{repo: repo}

		_, err := s.SyncMessages(ctx, "conv-1", "user-2", 0, 10)
		if !errors.Is(err, domain.ErrNotParticipant) {
			t.Errorf("expected ErrNotParticipant, got %v", err)
		}
	})

	t.Run("failure_when_conv_not_found", func(t *testing.T) {
		repo := &mockSyncRepo{conv: nil}
		s := &Service{repo: repo}

		_, err := s.SyncMessages(ctx, "conv-ghost", "user-1", 0, 10)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})
}
