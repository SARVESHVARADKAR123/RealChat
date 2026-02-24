package application

import (
	"context"
	"database/sql"

	messagingv1 "github.com/SARVESHVARADKAR123/RealChat/contracts/gen/go/messaging/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *Service) UpdateReadReceipt(
	ctx context.Context,
	convID, userID string,
	readSequence int64,
) error {

	return s.tx.WithTx(ctx, func(ctx context.Context, tx *sql.Tx) error {

		maxSeq, err := s.repo.GetCurrentMaxSequence(ctx, tx, convID)
		if err != nil {
			return err
		}

		if readSequence > maxSeq {
			readSequence = maxSeq
		}

		if err := s.repo.UpdateLastReadSequence(
			ctx, tx,
			convID,
			userID,
			readSequence,
		); err != nil {
			return err
		}

		// Emit Event
		event := &messagingv1.ReadReceiptUpdatedEvent{
			ConversationId: convID,
			UserId:         userID,
			ReadSequence:   readSequence,
		}
		eventPayload, err := proto.Marshal(event)
		if err != nil {
			return err
		}

		env := &messagingv1.MessagingEventEnvelope{
			EventType:     messagingv1.MessagingEventType_MESSAGING_EVENT_TYPE_READ_RECEIPT_UPDATED,
			SchemaVersion: 1,
			OccurredAt:    timestamppb.Now(),
			Payload:       eventPayload,
		}
		envPayload, err := proto.Marshal(env)
		if err != nil {
			return err
		}

		return s.repo.InsertOutbox(
			ctx, tx,
			"message",
			convID,
			"READ_RECEIPT_UPDATED",
			envPayload,
		)
	})
}
