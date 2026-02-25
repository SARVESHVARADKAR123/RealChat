package cache

import (
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
