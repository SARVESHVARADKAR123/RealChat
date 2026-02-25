package presence

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	"github.com/SARVESHVARADKAR123/RealChat/services/presence/internal/observability"

	presencev1 "github.com/SARVESHVARADKAR123/RealChat/contracts/gen/go/presence/v1"
)

const (
	TTL            = 60 * time.Second
	PresenceUpdate = "presence:updates"
)

type Presence struct {
	client     *redis.Client
	instanceID string
}

func New(addr, instanceID string) *Presence {
	return &Presence{
		client: redis.NewClient(&redis.Options{
			Addr: addr,
		}),
		instanceID: instanceID,
	}
}

func sessionKey(userID, deviceID string) string {
	return "session:" + userID + ":" + deviceID
}

func userSessionsSetKey(userID string) string {
	return "presence:user:" + userID + ":devices"
}

func (p *Presence) Register(ctx context.Context, userID, deviceID, instanceID string) error {
	pipe := p.client.TxPipeline()

	pipe.Set(ctx, sessionKey(userID, deviceID), instanceID, TTL)
	pipe.SAdd(ctx, userSessionsSetKey(userID), deviceID)
	pipe.Expire(ctx, userSessionsSetKey(userID), TTL+time.Hour)

	if _, err := pipe.Exec(ctx); err != nil {
		return err
	}

	return p.PublishUpdate(ctx, userID, presencev1.PresenceStatus_PRESENCE_STATUS_ONLINE)
}

func (p *Presence) Unregister(ctx context.Context, userID, deviceID string) error {
	pipe := p.client.TxPipeline()

	pipe.Del(ctx, sessionKey(userID, deviceID))
	pipe.SRem(ctx, userSessionsSetKey(userID), deviceID)

	if _, err := pipe.Exec(ctx); err != nil {
		return err
	}

	devices, err := p.GetUserDevices(ctx, userID)
	if err != nil || len(devices) == 0 {
		return p.PublishUpdate(ctx, userID, presencev1.PresenceStatus_PRESENCE_STATUS_OFFLINE)
	}

	return nil
}

func (p *Presence) Refresh(ctx context.Context, userID, deviceID string) error {
	pipe := p.client.TxPipeline()
	pipe.Expire(ctx, sessionKey(userID, deviceID), TTL)
	pipe.Expire(ctx, userSessionsSetKey(userID), TTL+time.Hour)
	_, err := pipe.Exec(ctx)
	return err
}

func (p *Presence) PublishUpdate(ctx context.Context, userID string, status presencev1.PresenceStatus) error {
	event := &presencev1.PresenceUpdateEvent{
		UserId:     userID,
		Status:     status,
		OccurredAt: time.Now().Unix(),
	}

	payload, err := proto.Marshal(event)
	if err != nil {
		return err
	}

	return p.client.Publish(ctx, PresenceUpdate, payload).Err()
}

func (p *Presence) GetUserDevices(ctx context.Context, userID string) (map[string]string, error) {
	log := observability.GetLogger(ctx)

	deviceIDs, err := p.client.SMembers(ctx, userSessionsSetKey(userID)).Result()
	if err != nil {
		return nil, err
	}

	if len(deviceIDs) == 0 {
		return make(map[string]string), nil
	}

	keys := make([]string, len(deviceIDs))
	for i, dID := range deviceIDs {
		keys[i] = sessionKey(userID, dID)
	}

	instances, err := p.client.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, err
	}

	result := make(map[string]string)
	var staleDevices []any

	for i, instance := range instances {
		dID := deviceIDs[i]
		if instance == nil {
			staleDevices = append(staleDevices, dID)
			continue
		}
		if instStr, ok := instance.(string); ok {
			result[dID] = instStr
		}
	}

	// Async cleanup of stale devices
	if len(staleDevices) > 0 {
		go func() {
			err := p.client.SRem(context.Background(), userSessionsSetKey(userID), staleDevices...).Err()
			if err != nil {
				log.Error("presence: fail to cleanup stale devices", zap.String("user_id", userID), zap.Error(err))
			}
		}()
	}

	return result, nil
}
