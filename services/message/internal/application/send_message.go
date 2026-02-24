package application

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	messagingv1 "github.com/SARVESHVARADKAR123/RealChat/contracts/gen/go/messaging/v1"
	"github.com/SARVESHVARADKAR123/RealChat/services/message/internal/domain"
	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type SendMessageCommand struct {
	ConversationID string
	UserID         string
	ClientMsgID    string
	Type           string
	Content        string
	Metadata       string
}

func (s *Service) SendMessage(
	ctx context.Context,
	cmd SendMessageCommand,
) (*domain.Message, error) {

	var result *domain.Message

	err := s.tx.WithTx(ctx, func(ctx context.Context, tx *sql.Tx) error {

		owned, err := s.repo.TryInsertIdempotency(
			ctx, tx,
			cmd.ClientMsgID,
			cmd.UserID,
			cmd.ConversationID,
			time.Now().Add(24*time.Hour),
		)
		if err != nil {
			return fmt.Errorf("failed to check idempotency: %w", err)
		}

		if !owned {
			payload, err := s.repo.GetIdempotencyForUpdate(
				ctx, tx,
				cmd.ClientMsgID,
				cmd.UserID,
				cmd.ConversationID,
			)
			if err != nil {
				return fmt.Errorf("failed to fetch idempotency response: %w", err)
			}
			if payload != nil {
				var msg domain.Message
				if err := json.Unmarshal(payload, &msg); err != nil {
					return fmt.Errorf("failed to unmarshal cached message: %w", err)
				}
				result = &msg
				return nil
			}
		}

		conv, err := s.repo.GetConversation(ctx, tx, cmd.ConversationID)
		if err != nil {
			return fmt.Errorf("failed to fetch conversation: %w", err)
		}

		if err := conv.CanSend(cmd.UserID); err != nil {
			return fmt.Errorf("permission denied: %w", err)
		}

		seq, err := s.repo.NextSequence(ctx, tx, cmd.ConversationID)
		if err != nil {
			return fmt.Errorf("failed to generate message sequence: %w", err)
		}

		msg, err := domain.NewMessage(
			uuid.NewString(),
			cmd.ConversationID,
			cmd.UserID,
			seq,
			cmd.Type,
			cmd.Content,
			cmd.Metadata,
			time.Now().UTC(),
		)
		if err != nil {
			return fmt.Errorf("failed to create new message: %w", err)
		}

		if err := s.repo.InsertMessage(ctx, tx, msg); err != nil {
			return fmt.Errorf("failed to save message: %w", err)
		}

		payload, err := json.Marshal(msg)
		if err != nil {
			return fmt.Errorf("failed to marshal message for idempotency: %w", err)
		}

		// Emit Event
		pbMsg := &messagingv1.Message{
			MessageId:      msg.ID,
			ConversationId: msg.ConversationID,
			SenderUserId:   msg.SenderID,
			Sequence:       msg.Sequence,
			MessageType:    msg.Type,
			Content:        msg.Content,
			MetadataJson:   msg.Metadata,
			SentAt:         timestamppb.New(msg.SentAt),
		}

		event := &messagingv1.MessageSentEvent{
			Message: pbMsg,
		}
		eventPayload, err := proto.Marshal(event)
		if err != nil {
			return fmt.Errorf("failed to marshal event payload: %w", err)
		}

		env := &messagingv1.MessagingEventEnvelope{
			EventType:     messagingv1.MessagingEventType_MESSAGING_EVENT_TYPE_MESSAGE_SENT,
			SchemaVersion: 1,
			OccurredAt:    timestamppb.Now(),
			Payload:       eventPayload,
		}
		envPayload, err := proto.Marshal(env)
		if err != nil {
			return fmt.Errorf("failed to marshal event envelope: %w", err)
		}

		if err := s.repo.InsertOutbox(
			ctx, tx,
			"message",
			msg.ConversationID,
			"MESSAGE_SENT",
			envPayload,
		); err != nil {
			return fmt.Errorf("failed to save outbox event: %w", err)
		}

		// Idempotency: use the same payload for the response
		if err := s.repo.UpdateIdempotencyResponse(
			ctx, tx,
			cmd.ClientMsgID,
			cmd.UserID,
			cmd.ConversationID,
			payload, // result of json.Marshal(msg) for the direct response
		); err != nil {
			return fmt.Errorf("failed to update idempotency response: %w", err)
		}

		result = msg
		return nil
	})

	return result, err
}
