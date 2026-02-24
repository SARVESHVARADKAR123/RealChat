package application

import (
	"context"
	"database/sql"

	messagingv1 "github.com/SARVESHVARADKAR123/RealChat/contracts/gen/go/messaging/v1"
	"github.com/SARVESHVARADKAR123/RealChat/services/message/internal/domain"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type DeleteMessageCommand struct {
	ConversationID string
	MessageID      string
	RequesterID    string
}

func (s *Service) DeleteMessage(
	ctx context.Context,
	cmd DeleteMessageCommand,
) error {

	return s.tx.WithTx(ctx, func(ctx context.Context, tx *sql.Tx) error {

		// 1️⃣ Verify message exists and requester is sender
		msg, err := s.repo.GetMessageForUpdate(
			ctx,
			tx,
			cmd.MessageID,
		)
		if err != nil {
			return err
		}

		if msg.ConversationID != cmd.ConversationID {
			return domain.ErrInvalidInput
		}

		if msg.SenderID != cmd.RequesterID {
			return domain.ErrNotParticipant
		}

		// Already deleted? idempotent no-op
		if msg.DeletedAt != nil {
			return nil
		}

		// 2️⃣ Soft delete
		if err := s.repo.MarkMessageDeleted(
			ctx,
			tx,
			cmd.MessageID,
		); err != nil {
			return err
		}

		// 3️⃣ Emit outbox event
		event := &messagingv1.MessageDeletedEvent{
			ConversationId: cmd.ConversationID,
			MessageId:      cmd.MessageID,
		}
		eventPayload, err := proto.Marshal(event)
		if err != nil {
			return err
		}

		env := &messagingv1.MessagingEventEnvelope{
			EventType:     messagingv1.MessagingEventType_MESSAGING_EVENT_TYPE_MESSAGE_DELETED,
			SchemaVersion: 1,
			OccurredAt:    timestamppb.Now(),
			Payload:       eventPayload,
		}
		payload, err := proto.Marshal(env)
		if err != nil {
			return err
		}

		return s.repo.InsertOutbox(
			ctx,
			tx,
			"message",
			cmd.ConversationID,
			"MESSAGE_DELETED",
			payload,
		)
	})
}
