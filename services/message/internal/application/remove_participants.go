package application

import (
	"context"
	"database/sql"

	messagingv1 "github.com/SARVESHVARADKAR123/RealChat/contracts/gen/go/messaging/v1"
	"github.com/SARVESHVARADKAR123/RealChat/services/message/internal/domain"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type RemoveParticipantCommand struct {
	ConversationID string
	ActorID        string
	TargetID       string
}

func (s *Service) RemoveParticipant(
	ctx context.Context,
	cmd RemoveParticipantCommand,
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

		reqRole, ok := conv.Participants[cmd.ActorID]
		if !ok || reqRole.Role != domain.RoleAdmin {
			return domain.ErrNotAdmin
		}

		targetRole, exists := conv.Participants[cmd.TargetID]
		if !exists {
			return nil // already removed
		}

		// If removing an admin → ensure not last admin
		if targetRole.Role == domain.RoleAdmin {

			adminCount := 0
			for _, r := range conv.Participants {
				if r.Role == domain.RoleAdmin {
					adminCount++
				}
			}

			if adminCount <= 1 {
				return domain.ErrLastAdmin
			}
		}

		if err := s.repo.DeleteParticipant(
			ctx,
			tx,
			cmd.ConversationID,
			cmd.TargetID,
		); err != nil {
			return err
		}

		//add admin to adming
		// Emit Event
		event := &messagingv1.MembershipChangedEvent{
			ConversationId: cmd.ConversationID,
			UserId:         cmd.TargetID,
			Added:          false,
		}
		eventPayload, err := proto.Marshal(event)
		if err != nil {
			return err
		}

		env := &messagingv1.MessagingEventEnvelope{
			EventType:     messagingv1.MessagingEventType_MESSAGING_EVENT_TYPE_MEMBERSHIP_CHANGED,
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
