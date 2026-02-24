package presencewatcher

import (
	"context"
	"time"

	messagingv1 "github.com/SARVESHVARADKAR123/RealChat/contracts/gen/go/messaging/v1"
	presencev1 "github.com/SARVESHVARADKAR123/RealChat/contracts/gen/go/presence/v1"
	"github.com/SARVESHVARADKAR123/RealChat/services/delivery/internal/membership"
	"github.com/SARVESHVARADKAR123/RealChat/services/delivery/internal/observability"
	"github.com/SARVESHVARADKAR123/RealChat/services/delivery/internal/presence"
	"github.com/SARVESHVARADKAR123/RealChat/services/delivery/internal/websocket"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Watcher struct {
	client     *redis.Client
	registry   *websocket.Registry
	membership *membership.Cache
}

func NewWatcher(client *redis.Client, registry *websocket.Registry, membership *membership.Cache) *Watcher {
	return &Watcher{
		client:     client,
		registry:   registry,
		membership: membership,
	}
}

func (w *Watcher) Start(ctx context.Context) {
	go func() {
		pubsub := w.client.Subscribe(ctx, presence.PresenceUpdate)
		defer pubsub.Close()

		ch := pubsub.Channel()
		for msg := range ch {
			var event presencev1.PresenceUpdateEvent
			if err := proto.Unmarshal([]byte(msg.Payload), &event); err != nil {
				observability.GetLogger(ctx).Error("presence watcher: error unmarshaling event", zap.Error(err))
				continue
			}

			w.handlePresenceUpdate(ctx, &event)
		}
	}()
}

func (w *Watcher) handlePresenceUpdate(ctx context.Context, event *presencev1.PresenceUpdateEvent) {
	log := observability.GetLogger(ctx)

	convIDs := w.membership.UserConvs(event.UserId)
	if len(convIDs) == 0 {
		return
	}

	eventPayload, err := proto.Marshal(event)
	if err != nil {
		log.Error("presence watcher: error marshaling event", zap.Error(err))
		return
	}

	// Wrap in MessagingEventEnvelope
	env := &messagingv1.MessagingEventEnvelope{
		EventType:     messagingv1.MessagingEventType_MESSAGING_EVENT_TYPE_PRESENCE_UPDATED,
		SchemaVersion: 1,
		OccurredAt:    timestamppb.New(time.Unix(event.OccurredAt, 0)),
		Payload:       eventPayload,
	}

	envPayload, err := proto.Marshal(env)
	if err != nil {
		log.Error("presence watcher: error marshaling envelope", zap.Error(err))
		return
	}

	w.notifySubscribers(event.UserId, convIDs, envPayload)
	log.Debug("presence watcher: processed update", zap.String("user_id", event.UserId), zap.String("status", event.Status.String()))
}

func (w *Watcher) notifySubscribers(userID string, convIDs []string, envPayload []byte) {
	notifiedUsers := make(map[string]struct{})
	notifiedUsers[userID] = struct{}{}

	for _, convID := range convIDs {
		members := w.membership.Members(convID)
		for _, memberID := range members {
			if _, ok := notifiedUsers[memberID]; ok {
				continue
			}

			sessions := w.registry.GetUserSessions(memberID)
			for _, s := range sessions {
				s.TrySend(envPayload)
			}
			if len(sessions) > 0 {
				notifiedUsers[memberID] = struct{}{}
			}
		}
	}
}
