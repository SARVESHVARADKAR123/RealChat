package router

import (
	"context"

	"github.com/SARVESHVARADKAR123/RealChat/services/delivery/internal/observability"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type Router struct {
	client     *redis.Client
	instanceID string
}

func New(client *redis.Client, instanceID string) *Router {
	return &Router{client: client, instanceID: instanceID}
}

func (r *Router) channel(id string) string {
	return "delivery:" + id
}

func (r *Router) Publish(ctx context.Context, target string, payload []byte) error {
	log := observability.GetLogger(ctx)
	log.Debug("publishing to instance", zap.String("target", target))
	return r.client.Publish(ctx, r.channel(target), payload).Err()
}

func (r *Router) Subscribe(ctx context.Context, handler func([]byte)) {
	channelName := r.channel(r.instanceID)
	pubsub := r.client.Subscribe(ctx, channelName)

	go func() {
		log := observability.GetLogger(ctx)
		log.Info("router: subscribed to channel", zap.String("channel", channelName))
		defer pubsub.Close()

		ch := pubsub.Channel()
		for {
			select {
			case <-ctx.Done():
				log.Info("router: subscription loop stopping: context canceled")
				return
			case msg, ok := <-ch:
				if !ok {
					log.Warn("router: pubsub channel closed")
					return
				}
				log.Debug("router: received message from channel", zap.String("channel", channelName))
				handler([]byte(msg.Payload))
			}
		}
	}()
}
