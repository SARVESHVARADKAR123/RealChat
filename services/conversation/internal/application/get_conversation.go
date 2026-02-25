package application

import (
	"context"

	"github.com/SARVESHVARADKAR123/RealChat/services/conversation/internal/domain"
)

func (s *Service) GetConversation(
	ctx context.Context,
	conversationID string,
) (*domain.Conversation, error) {
	return s.repo.GetConversation(ctx, nil, conversationID)
}
