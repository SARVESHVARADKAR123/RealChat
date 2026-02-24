package application

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/SARVESHVARADKAR123/RealChat/services/message/internal/domain"
)

type createConvRepo struct {
	mockSyncRepo
	convs map[string]*domain.Conversation
}

func (m *createConvRepo) GetConversationByLookupKey(ctx context.Context, tx *sql.Tx, key string) (*domain.Conversation, error) {
	for _, c := range m.convs {
		// Mock lookup key logic
		p1, p2 := "", ""
		participants := []string{}
		for uid := range c.Participants {
			participants = append(participants, uid)
		}
		if len(participants) == 2 {
			p1, p2 = participants[0], participants[1]
			if p1 > p2 {
				p1, p2 = p2, p1
			}
			if fmt.Sprintf("direct:%s:%s", p1, p2) == key {
				return c, nil
			}
		}
	}
	return nil, domain.ErrConversationNotFound
}

func (m *createConvRepo) InsertConversation(ctx context.Context, tx *sql.Tx, id string, convType domain.ConversationType, name, avatar string, lookupKey *string) error {
	m.convs[id] = &domain.Conversation{
		ID:           id,
		Type:         convType,
		Participants: make(map[string]domain.Participant),
	}
	return nil
}

func (m *createConvRepo) GetConversationLocked(ctx context.Context, tx *sql.Tx, id string) (*domain.Conversation, error) {
	return m.convs[id], nil
}

func (m *createConvRepo) ListConversationsByUser(ctx context.Context, userID string) ([]*domain.Conversation, error) {
	return nil, nil
}

func (m *createConvRepo) InsertParticipant(ctx context.Context, tx *sql.Tx, convID, userID string, role domain.Role) error {
	m.convs[convID].Participants[userID] = domain.Participant{UserID: userID, Role: role}
	return nil
}

func TestService_CreateConversation_Idempotency(t *testing.T) {
	ctx := context.Background()
	repo := &createConvRepo{
		convs: make(map[string]*domain.Conversation),
	}
	s := &Service{repo: repo, tx: &mockTransactor{}}

	cmd := CreateConversationCommand{
		ID:           "conv-1",
		Type:         domain.ConversationDirect,
		Participants: []string{"user-a", "user-b"},
	}

	// 1. Create for the first time
	c1, err := s.CreateConversation(ctx, cmd)
	if err != nil {
		t.Fatalf("failed to create: %v", err)
	}

	// 2. Try creating again with same participants but DIFFERENT ID
	cmdDuplicate := CreateConversationCommand{
		ID:           "conv-2",
		Type:         domain.ConversationDirect,
		Participants: []string{"user-b", "user-a"}, // reverse order
	}

	c2, err := s.CreateConversation(ctx, cmdDuplicate)
	if err != nil {
		t.Fatalf("failed duplicate: %v", err)
	}

	if c1.ID != c2.ID {
		t.Errorf("expected same ID for duplicate direct chat, got %s != %s", c1.ID, c2.ID)
	}
}
