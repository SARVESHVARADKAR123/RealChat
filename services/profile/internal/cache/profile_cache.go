package cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/SARVESHVARADKAR123/RealChat/services/profile/internal/model"

	"github.com/redis/go-redis/v9"
)

type ProfileCache struct{ R *redis.Client }

func key(id string) string { return "profile:" + id }

func (c *ProfileCache) Get(ctx context.Context, id string) (*model.Profile, error) {
	b, err := c.R.Get(ctx, key(id)).Bytes()
	if err != nil {
		return nil, err
	}
	var p model.Profile
	return &p, json.Unmarshal(b, &p)
}

func (c *ProfileCache) Set(ctx context.Context, p *model.Profile) error {
	b, err := json.Marshal(p)
	if err != nil {
		return err
	}
	return c.R.Set(ctx, key(p.UserID), b, time.Hour).Err()
}

func (c *ProfileCache) Delete(ctx context.Context, id string) error {
	return c.R.Del(ctx, key(id)).Err()
}
