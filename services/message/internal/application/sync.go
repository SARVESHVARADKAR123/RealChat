package application

import (
	"context"

	conversationv1 "github.com/SARVESHVARADKAR123/RealChat/contracts/gen/go/conversation/v1"
	"github.com/SARVESHVARADKAR123/RealChat/services/message/internal/domain"
)

func (s *Service) SyncMessages(
	ctx context.Context,
	conversationID string,
	userID string,
	afterSequence int64,
	pageSize int,
) ([]*domain.Message, error) {

	if pageSize <= 0 || pageSize > 500 {
		pageSize = 100
	}

	// Verify membership
	resp, err := s.convSvc.GetConversation(ctx, &conversationv1.GetConversationRequest{
		ConversationId: conversationID,
	})
	if err != nil {
		return nil, err
	}

	isParticipant := false
	for _, pID := range resp.ParticipantUserIds {
		if pID == userID {
			isParticipant = true
			break
		}
	}

	if !isParticipant {
		return nil, domain.ErrNotParticipant
	}

	return s.repo.FetchMessages(
		ctx,
		conversationID,
		afterSequence,
		pageSize,
	)
}
