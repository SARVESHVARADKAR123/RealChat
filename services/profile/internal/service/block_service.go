package service

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/SARVESHVARADKAR123/RealChat/services/profile/internal/outbox"
	"github.com/SARVESHVARADKAR123/RealChat/services/profile/internal/repository"
)

// BlockService handles block/unblock business logic.
type BlockService struct {
	Repo        *repository.BlockRepo
	ContactRepo *repository.ContactRepo
	Outbox      *outbox.Repository
}

// Block creates a block, removes the contact, and writes an outbox event.
func (s *BlockService) Block(ctx context.Context, user, other string) error {
	if user == other {
		return errors.New("cannot block self")
	}

	if err := s.Repo.Add(ctx, user, other); err != nil {
		return err
	}

	if err := s.ContactRepo.Remove(ctx, user, other); err != nil {
		return err
	}

	b, err := json.Marshal(map[string]string{
		"user_id": user,
		"blocked": other,
	})
	if err != nil {
		return err
	}

	return s.Outbox.Add(ctx, "user.blocked", b)
}

// Unblock removes a block and writes an outbox event.
func (s *BlockService) Unblock(ctx context.Context, user, other string) error {
	if err := s.Repo.Remove(ctx, user, other); err != nil {
		return err
	}

	b, err := json.Marshal(map[string]string{
		"user_id": user,
		"blocked": other,
	})
	if err != nil {
		return err
	}

	return s.Outbox.Add(ctx, "user.unblocked", b)
}
