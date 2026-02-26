package application

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	conversationv1 "github.com/SARVESHVARADKAR123/RealChat/contracts/gen/go/conversation/v1"
	messagev1 "github.com/SARVESHVARADKAR123/RealChat/contracts/gen/go/message/v1"
	sharedv1 "github.com/SARVESHVARADKAR123/RealChat/contracts/gen/go/shared/v1"
	"github.com/SARVESHVARADKAR123/RealChat/services/message/internal/domain"
	"github.com/google/uuid"
	"go.uber.org/zap"
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

	s.log.Info("SendMessage requested",
		zap.String("conversation_id", cmd.ConversationID),
		zap.String("user_id", cmd.UserID),
	)

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

		// Parallelize calls to conversation service
		resp, seq, err := s.parallelConvCalls(ctx, cmd.ConversationID)
		if err != nil {
			return err
		}

		isParticipant := false
		for _, pID := range resp.ParticipantUserIds {
			if pID == cmd.UserID {
				isParticipant = true
				break
			}
		}

		if !isParticipant {
			return domain.ErrNotParticipant
		}

		s.log.Info("Message sequence generated successfully", zap.Any("sequence", seq))

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

		s.log.Info("Message created successfully", zap.Any("message", msg))

		if err := s.repo.InsertMessage(ctx, tx, msg); err != nil {
			return fmt.Errorf("failed to save message: %w", err)
		}

		s.log.Info("Message inserted successfully", zap.Any("message", msg))

		payload, err := json.Marshal(msg)
		if err != nil {
			return fmt.Errorf("failed to marshal message for idempotency: %w", err)
		}
		s.log.Info("Message marshaled successfully", zap.Any("message", msg))

		// Emit Event
		pbMsg := &messagev1.Message{
			MessageId:      msg.ID,
			ConversationId: msg.ConversationID,
			SenderUserId:   msg.SenderID,
			Sequence:       msg.Sequence,
			MessageType:    msg.Type,
			Content:        msg.Content,
			MetadataJson:   msg.Metadata,
			SentAt:         timestamppb.New(msg.SentAt),
		}

		s.log.Info("Message created successfully", zap.Any("message", pbMsg))

		event := &messagev1.MessageSentEvent{
			Message: pbMsg,
		}
		eventPayload, err := proto.Marshal(event)
		if err != nil {
			return fmt.Errorf("failed to marshal event payload: %w", err)
		}

		env := &sharedv1.EventEnvelope{
			EventType:     sharedv1.EventType_EVENT_TYPE_MESSAGE_SENT,
			SchemaVersion: 1,
			OccurredAt:    timestamppb.Now(),
			Payload:       eventPayload,
		}
		envPayload, err := proto.Marshal(env)
		if err != nil {
			return fmt.Errorf("failed to marshal event envelope: %w", err)
		}
		s.log.Info("Event envelope marshaled successfully", zap.Any("event", env))

		if err := s.repo.InsertOutbox(
			ctx, tx,
			"message",
			msg.ConversationID,
			"MESSAGE_SENT",
			envPayload,
		); err != nil {
			return fmt.Errorf("failed to save outbox event: %w", err)
		}

		s.log.Info("Outbox event saved successfully", zap.Any("event", env))
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

func (s *Service) parallelConvCalls(ctx context.Context, convID string) (*conversationv1.GetConversationResponse, int64, error) {
	type convRes struct {
		resp *conversationv1.GetConversationResponse
		err  error
	}
	type seqRes struct {
		resp *conversationv1.NextSequenceResponse
		err  error
	}

	convChan := make(chan convRes, 1)
	seqChan := make(chan seqRes, 1)

	go func() {
		resp, err := s.convSvc.GetConversation(ctx, &conversationv1.GetConversationRequest{
			ConversationId: convID,
		})
		convChan <- convRes{resp, err}
	}()

	go func() {
		resp, err := s.convSvc.NextSequence(ctx, &conversationv1.NextSequenceRequest{
			ConversationId: convID,
		})
		seqChan <- seqRes{resp, err}
	}()

	cr := <-convChan
	if cr.err != nil {
		return nil, 0, fmt.Errorf("failed to fetch conversation via gRPC: %w", cr.err)
	}

	sr := <-seqChan
	if sr.err != nil {
		return nil, 0, fmt.Errorf("failed to generate message sequence: %w", sr.err)
	}

	return cr.resp, sr.resp.Sequence, nil
}
