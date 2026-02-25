package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/SARVESHVARADKAR123/RealChat/services/profile/internal/cache"
	"github.com/SARVESHVARADKAR123/RealChat/services/profile/internal/model"
	"github.com/SARVESHVARADKAR123/RealChat/services/profile/internal/outbox"
	"github.com/SARVESHVARADKAR123/RealChat/services/profile/internal/repository"
)

// ProfileService handles profile business logic.
type ProfileService struct {
	Repo   *repository.ProfileRepo
	Cache  *cache.ProfileCache
	Outbox *outbox.Repository
}

// Get returns a profile by user ID, checking cache first.
func (s *ProfileService) Get(ctx context.Context, id string) (*model.Profile, error) {
	if p, err := s.Cache.Get(ctx, id); err == nil {
		return p, nil
	}
	p, err := s.Repo.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch profile: %w", err)
	}
	_ = s.Cache.Set(ctx, p)
	return p, nil
}

// BatchGet returns multiple profiles by user IDs.
func (s *ProfileService) BatchGet(ctx context.Context, ids []string) ([]*model.Profile, error) {
	// For simplicity, we skip cache for batch get now, or we could implement it
	profiles, err := s.Repo.BatchGet(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("failed to batch fetch profiles: %w", err)
	}
	return profiles, nil
}

// Update modifies a profile, invalidates the cache, and writes an outbox event.
func (s *ProfileService) Update(ctx context.Context, p *model.Profile) error {
	if err := s.Repo.Update(ctx, p); err != nil {
		return fmt.Errorf("failed to update profile in repo: %w", err)
	}

	if err := s.Cache.Delete(ctx, p.UserID); err != nil {
		// Log but don't fail update (cache miss is fine)
	}

	payload, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("failed to marshal profile: %w", err)
	}
	if err := s.Outbox.Add(ctx, "profile.updated", payload); err != nil {
		return fmt.Errorf("failed to save outbox event: %w", err)
	}
	return nil
}
