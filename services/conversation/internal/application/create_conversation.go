package application

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	conversationv1 "github.com/SARVESHVARADKAR123/RealChat/contracts/gen/go/conversation/v1"
	sharedv1 "github.com/SARVESHVARADKAR123/RealChat/contracts/gen/go/shared/v1"
	"github.com/SARVESHVARADKAR123/RealChat/services/conversation/internal/domain"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type CreateConversationCommand struct {
	ID           string
	Type         domain.ConversationType
	Name         string
	AvatarURL    string
	Participants []string // for group: first is creator (admin)
}

func (s *Service) CreateConversation(
	ctx context.Context,
	cmd CreateConversationCommand,
) (*domain.Conversation, error) {

	if cmd.ID == "" {
		return nil, domain.ErrInvalidInput
	}

	if cmd.Type == domain.ConversationDirect && len(cmd.Participants) != 2 {
		return nil, domain.ErrInvalidInput
	}
	if cmd.Type == domain.ConversationGroup && len(cmd.Participants) == 0 {
		return nil, domain.ErrInvalidInput
	}

	lookupKey := s.getLookupKey(cmd)
	if cmd.Type == domain.ConversationDirect && lookupKey == nil {
		return nil, domain.ErrInvalidInput
	}

	// 1. Initial best-effort lookups before transaction in parallel
	if existing := s.parallelLookup(ctx, cmd.ID, lookupKey); existing != nil {
		return existing, nil
	}

	var result *domain.Conversation
	txErr := s.tx.WithTx(ctx, func(ctx context.Context, tx *sql.Tx) error {
		// 2. Double-check inside transaction (for both ID and lookup key)
		if cmd.ID != "" {
			if existing, err := s.repo.GetConversation(ctx, tx, cmd.ID); err == nil && existing != nil {
				result = existing
				return nil
			}
		}

		if lookupKey != nil {
			if existing, err := s.repo.GetConversationByLookupKey(ctx, tx, *lookupKey); err == nil {
				result = existing
				return nil
			}
		}

		// 3. Create new conversation
		conv, err := s.doCreateConversation(ctx, tx, cmd, lookupKey)
		if err != nil {
			// 4. Handle race condition violation
			if cmd.ID != "" {
				if existing, errRefetch := s.repo.GetConversation(ctx, tx, cmd.ID); errRefetch == nil {
					result = existing
					return nil
				}
			}
			if lookupKey != nil {
				if existing, errRefetch := s.repo.GetConversationByLookupKey(ctx, tx, *lookupKey); errRefetch == nil {
					result = existing
					return nil
				}
			}
			return err
		}

		result = conv
		return nil
	})

	return result, txErr
}

func (s *Service) getLookupKey(cmd CreateConversationCommand) *string {
	if cmd.Type != domain.ConversationDirect || len(cmd.Participants) != 2 {
		return nil
	}
	p1, p2 := cmd.Participants[0], cmd.Participants[1]
	if p1 > p2 {
		p1, p2 = p2, p1
	}
	key := fmt.Sprintf("direct:%s:%s", p1, p2)
	return &key
}

func (s *Service) doCreateConversation(
	ctx context.Context,
	tx *sql.Tx,
	cmd CreateConversationCommand,
	lookupKey *string,
) (*domain.Conversation, error) {
	// 1️⃣ Insert conversation row
	if err := s.repo.InsertConversation(
		ctx, tx, cmd.ID, cmd.Type, cmd.Name, cmd.AvatarURL, lookupKey,
	); err != nil {
		return nil, fmt.Errorf("failed to create conversation: %w", err)
	}

	// 2️⃣ Initialize sequence row
	if err := s.repo.InitSequence(ctx, tx, cmd.ID); err != nil {
		return nil, fmt.Errorf("failed to initialize sequence: %w", err)
	}

	// 3️⃣ Insert participants
	if err := s.insertParticipants(ctx, tx, cmd); err != nil {
		return nil, err
	}

	// 4️⃣ Emit Event and return
	return s.emitConversationCreated(ctx, tx, cmd.ID)
}

func (s *Service) insertParticipants(ctx context.Context, tx *sql.Tx, cmd CreateConversationCommand) error {
	for i, userID := range cmd.Participants {
		role := domain.RoleMember
		if cmd.Type == domain.ConversationGroup && i == 0 {
			role = domain.RoleAdmin
		}
		if err := s.repo.InsertParticipant(ctx, tx, cmd.ID, userID, role); err != nil {
			return fmt.Errorf("failed to add participant %s: %w", userID, err)
		}
	}
	return nil
}

func (s *Service) emitConversationCreated(ctx context.Context, tx *sql.Tx, convID string) (*domain.Conversation, error) {
	conv, err := s.repo.GetConversationLocked(ctx, tx, convID)
	if err != nil {
		return nil, fmt.Errorf("failed to lock conversation after creation: %w", err)
	}

	pbParticipants := make([]string, 0, len(conv.Participants))
	for uid := range conv.Participants {
		pbParticipants = append(pbParticipants, uid)
	}

	var pbType conversationv1.ConversationType
	if conv.Type == domain.ConversationGroup {
		pbType = conversationv1.ConversationType_GROUP
	} else {
		pbType = conversationv1.ConversationType_DIRECT
	}

	pbConv := &conversationv1.Conversation{
		ConversationId: conv.ID,
		DisplayName:    conv.DisplayName,
		AvatarUrl:      conv.AvatarURL,
		Type:           pbType,
		CreatedAt:      timestamppb.New(conv.CreatedAt),
	}

	event := &conversationv1.ConversationCreatedEvent{
		Conversation:       pbConv,
		ParticipantUserIds: pbParticipants,
	}
	eventPayload, err := proto.Marshal(event)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal event payload: %w", err)
	}

	env := &sharedv1.EventEnvelope{
		EventType:     sharedv1.EventType_EVENT_TYPE_CONVERSATION_CREATED,
		SchemaVersion: 1,
		OccurredAt:    pbConv.CreatedAt, // Align with domain event timestamp
		Payload:       eventPayload,
	}
	envPayload, err := proto.Marshal(env)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal event envelope: %w", err)
	}

	if err := s.repo.InsertOutbox(ctx, tx, "message", conv.ID, "CONVERSATION_CREATED", envPayload); err != nil {
		return nil, fmt.Errorf("failed to save outbox event: %w", err)
	}

	return conv, nil
}

func (s *Service) parallelLookup(ctx context.Context, id string, lookupKey *string) *domain.Conversation {
	resChan := make(chan *domain.Conversation, 2)
	var wg sync.WaitGroup

	if id != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if existing, err := s.repo.GetConversation(ctx, nil, id); err == nil && existing != nil {
				resChan <- existing
			}
		}()
	}

	if lookupKey != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if existing, err := s.repo.GetConversationByLookupKey(ctx, nil, *lookupKey); err == nil && existing != nil {
				resChan <- existing
			}
		}()
	}

	// Wait for all to finish in a separate goroutine to close the channel
	go func() {
		wg.Wait()
		close(resChan)
	}()

	// Return the first one that succeeded
	for conv := range resChan {
		if conv != nil {
			return conv
		}
	}

	return nil
}
