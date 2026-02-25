package application

import (
	"context"
	"database/sql"

	conversationv1 "github.com/SARVESHVARADKAR123/RealChat/contracts/gen/go/conversation/v1"
	sharedv1 "github.com/SARVESHVARADKAR123/RealChat/contracts/gen/go/shared/v1"
	"github.com/SARVESHVARADKAR123/RealChat/services/conversation/internal/domain"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type AddParticipantCommand struct {
	ConversationID string
	ActorID        string
	TargetID       string
}

func (s *Service) AddParticipant(
	ctx context.Context,
	cmd AddParticipantCommand,
) error {

	return s.tx.WithTx(ctx, func(ctx context.Context, tx *sql.Tx) error {

		conv, err := s.repo.GetConversationLocked(
			ctx,
			tx,
			cmd.ConversationID,
		)
		if err != nil {
			return err
		}

		if conv.Type != domain.ConversationGroup {
			return domain.ErrDirectModification
		}

		role, ok := conv.Participants[cmd.ActorID]
		if !ok || role.Role != domain.RoleAdmin {
			return domain.ErrNotAdmin
		}

		// Already exists? no-op
		if _, exists := conv.Participants[cmd.TargetID]; exists {
			return nil
		}

		if err := s.repo.InsertParticipant(
			ctx,
			tx,
			cmd.ConversationID,
			cmd.TargetID,
			domain.RoleMember,
		); err != nil {
			return err
		}

		// Emit Event
		event := &conversationv1.MembershipChangedEvent{
			ConversationId: cmd.ConversationID,
			UserId:         cmd.TargetID,
			Added:          true,
		}
		eventPayload, err := proto.Marshal(event)
		if err != nil {
			return err
		}

		env := &sharedv1.EventEnvelope{
			EventType:     sharedv1.EventType_EVENT_TYPE_MEMBERSHIP_CHANGED,
			SchemaVersion: 1,
			OccurredAt:    timestamppb.Now(),
			Payload:       eventPayload,
		}
		envPayload, err := proto.Marshal(env)
		if err != nil {
			return err
		}

		if err := s.repo.InsertOutbox(
			ctx, tx,
			"message",
			cmd.ConversationID,
			"MEMBERSHIP_CHANGED",
			envPayload,
		); err != nil {
			return err
		}

		return s.repo.InvalidateConversation(ctx, cmd.ConversationID)
	})
}
