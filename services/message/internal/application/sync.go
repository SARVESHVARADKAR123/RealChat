package application

import (
	"context"

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
	conv, err := s.repo.GetConversation(ctx, nil, conversationID)
	if err != nil {
		return nil, err
	}

	if _, ok := conv.Participants[userID]; !ok {
		return nil, domain.ErrNotParticipant
	}

	return s.repo.FetchMessages(
		ctx,
		conversationID,
		afterSequence,
		pageSize,
	)
}
