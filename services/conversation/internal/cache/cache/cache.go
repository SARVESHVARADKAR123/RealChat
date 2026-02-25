package cache

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/SARVESHVARADKAR123/RealChat/services/conversation/internal/domain"
	"github.com/redis/go-redis/v9"
)

type Cache struct {
	Client *redis.Client
}

func New(addr string) *Cache {
	return &Cache{
		Client: redis.NewClient(&redis.Options{
			Addr: addr,
		}),
	}
}

func (c *Cache) GetConversation(ctx context.Context, id string) (*domain.Conversation, error) {
	val, err := c.Client.Get(ctx, "conv:"+id).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil // Miss
		}
		return nil, err
	}

	var conv domain.Conversation
	if err := json.Unmarshal(val, &conv); err != nil {
		return nil, err
	}
	return &conv, nil
}

func (c *Cache) SetConversation(ctx context.Context, conv *domain.Conversation) error {
	val, err := json.Marshal(conv)
	if err != nil {
		return err
	}
	return c.Client.Set(ctx, "conv:"+conv.ID, val, 10*time.Minute).Err()
}

func (c *Cache) DeleteConversation(ctx context.Context, id string) error {
	return c.Client.Del(ctx, "conv:"+id).Err()
}
