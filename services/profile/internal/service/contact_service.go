package service

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/SARVESHVARADKAR123/RealChat/services/profile/internal/model"
	"github.com/SARVESHVARADKAR123/RealChat/services/profile/internal/outbox"
	"github.com/SARVESHVARADKAR123/RealChat/services/profile/internal/repository"
)

// ContactService handles contact business logic.
type ContactService struct {
	Repo      *repository.ContactRepo
	BlockRepo *repository.BlockRepo
	Outbox    *outbox.Repository
}

// Add creates a contact relationship after checking for self-add and blocks.
func (s *ContactService) Add(ctx context.Context, user, contact string) error {
	if user == contact {
		return errors.New("cannot add self")
	}

	blocked, err := s.BlockRepo.Exists(ctx, user, contact)
	if err != nil {
		return err
	}
	if blocked {
		return errors.New("blocked")
	}

	if err := s.Repo.Add(ctx, user, contact); err != nil {
		return err
	}

	b, err := json.Marshal(map[string]string{
		"user_id":    user,
		"contact_id": contact,
	})
	if err != nil {
		return err
	}

	return s.Outbox.Add(ctx, "contact.added", b)
}

// Remove deletes a contact relationship and writes an outbox event.
func (s *ContactService) Remove(ctx context.Context, user, contact string) error {
	if err := s.Repo.Remove(ctx, user, contact); err != nil {
		return err
	}

	b, err := json.Marshal(map[string]string{
		"user_id":    user,
		"contact_id": contact,
	})
	if err != nil {
		return err
	}

	return s.Outbox.Add(ctx, "contact.removed", b)
}

// List returns a paginated list of contacts for a user.
func (s *ContactService) List(ctx context.Context, user string, limit, offset int) ([]model.Contact, error) {
	return s.Repo.List(ctx, user, limit, offset)
}
