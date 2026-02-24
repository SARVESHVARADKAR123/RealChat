package application

import (
	"context"

	"github.com/SARVESHVARADKAR123/RealChat/services/message/internal/domain"
)

func (s *Service) ListConversations(
	ctx context.Context,
	userID string,
) ([]*domain.Conversation, error) {
	return s.repo.ListConversationsByUser(ctx, userID)
}
