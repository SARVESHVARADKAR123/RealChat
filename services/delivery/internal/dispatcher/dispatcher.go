package dispatcher

import (
	"context"
	"errors"

	"sync"

	conversationv1 "github.com/SARVESHVARADKAR123/RealChat/contracts/gen/go/conversation/v1"
	messagev1 "github.com/SARVESHVARADKAR123/RealChat/contracts/gen/go/message/v1"
	presencev1 "github.com/SARVESHVARADKAR123/RealChat/contracts/gen/go/presence/v1"
	sharedv1 "github.com/SARVESHVARADKAR123/RealChat/contracts/gen/go/shared/v1"
	"github.com/SARVESHVARADKAR123/RealChat/services/delivery/internal/membership"
	"github.com/SARVESHVARADKAR123/RealChat/services/delivery/internal/observability"
	"github.com/SARVESHVARADKAR123/RealChat/services/delivery/internal/router"
	"github.com/SARVESHVARADKAR123/RealChat/services/delivery/internal/websocket"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

type Dispatcher struct {
	registry       *websocket.Registry
	membership     *membership.Cache
	presenceClient presencev1.PresenceApiClient
	router         *router.Router
	instanceID     string
	convSvc        conversationv1.ConversationApiClient
}

func New(registry *websocket.Registry, membership *membership.Cache,
	presenceClient presencev1.PresenceApiClient, router *router.Router, instanceID string, convSvc conversationv1.ConversationApiClient) *Dispatcher {
	return &Dispatcher{
		registry:       registry,
		membership:     membership,
		presenceClient: presenceClient,
		router:         router,
		instanceID:     instanceID,
		convSvc:        convSvc,
	}
}

func (d *Dispatcher) Handle(ctx context.Context, record []byte) {
	log := observability.GetLogger(ctx)
	var env sharedv1.EventEnvelope
	if err := proto.Unmarshal(record, &env); err != nil {
		log.Error("dispatcher: error unmarshaling event", zap.Error(err))
		return
	}

	switch env.GetEventType() {
	case sharedv1.EventType_EVENT_TYPE_MESSAGE_SENT,
		sharedv1.EventType_EVENT_TYPE_MESSAGE_DELETED,
		sharedv1.EventType_EVENT_TYPE_READ_RECEIPT_UPDATED,
		sharedv1.EventType_EVENT_TYPE_MEMBERSHIP_CHANGED,
		sharedv1.EventType_EVENT_TYPE_CONVERSATION_CREATED:
		d.handleEvent(ctx, &env, record)
	}
}

func (d *Dispatcher) getConversationID(env *sharedv1.EventEnvelope) (string, error) {
	switch env.GetEventType() {
	case sharedv1.EventType_EVENT_TYPE_MESSAGE_SENT:
		var event messagev1.MessageSentEvent
		if err := proto.Unmarshal(env.GetPayload(), &event); err != nil {
			return "", err
		}
		return event.GetMessage().GetConversationId(), nil

	case sharedv1.EventType_EVENT_TYPE_MESSAGE_DELETED:
		var event messagev1.MessageDeletedEvent
		if err := proto.Unmarshal(env.GetPayload(), &event); err != nil {
			return "", err
		}
		return event.GetConversationId(), nil

	case sharedv1.EventType_EVENT_TYPE_READ_RECEIPT_UPDATED:
		var event conversationv1.ReadReceiptUpdatedEvent
		if err := proto.Unmarshal(env.GetPayload(), &event); err != nil {
			return "", err
		}
		return event.GetConversationId(), nil

	case sharedv1.EventType_EVENT_TYPE_MEMBERSHIP_CHANGED:
		var event conversationv1.MembershipChangedEvent
		if err := proto.Unmarshal(env.GetPayload(), &event); err != nil {
			return "", err
		}
		return event.GetConversationId(), nil

	case sharedv1.EventType_EVENT_TYPE_CONVERSATION_CREATED:
		var event conversationv1.ConversationCreatedEvent
		if err := proto.Unmarshal(env.GetPayload(), &event); err != nil {
			return "", err
		}
		return event.GetConversation().GetConversationId(), nil

	default:
		return "", errors.New("unsupported event type")
	}
}

func (d *Dispatcher) handleEvent(ctx context.Context, env *sharedv1.EventEnvelope, rawPayload []byte) {
	log := observability.GetLogger(ctx)
	// If membership ADDED -> update cache BEFORE routing so new member gets the event
	if env.GetEventType() == sharedv1.EventType_EVENT_TYPE_CONVERSATION_CREATED {
		d.handleConversationCreated(env)
	}

	if env.GetEventType() == sharedv1.EventType_EVENT_TYPE_MEMBERSHIP_CHANGED {
		d.handleMembershipPreRoute(env)
	}

	conversationID, err := d.getConversationID(env)
	if err != nil {
		log.Error("dispatcher: fail to get convID", zap.Error(err))
		return
	}

	members, err := d.resolveMembers(ctx, conversationID)
	if err != nil {
		return
	}

	remoteInstances := sync.Map{} // map[string]struct{}
	var wg sync.WaitGroup

	for _, userID := range members {
		wg.Add(1)
		go func(uid string) {
			defer wg.Done()
			d.deliverToUser(ctx, uid, env, rawPayload, &remoteInstances)
		}(userID)
	}
	wg.Wait()

	// Publish to each remote instance ONLY ONCE
	remoteInstances.Range(func(key, value interface{}) bool {
		instance := key.(string)
		if err := d.router.Publish(ctx, instance, rawPayload); err != nil {
			log.Error("dispatcher: remote routing failed", zap.String("instance", instance), zap.Error(err))
		} else {
			log.Info("dispatcher: remote routing success", zap.String("instance", instance))
		}
		return true
	})

	// If membership REMOVED -> update cache AFTER routing so removed member got the last event
	if env.GetEventType() == sharedv1.EventType_EVENT_TYPE_MEMBERSHIP_CHANGED {
		d.handleMembershipPostRoute(env)
	}
}

func (d *Dispatcher) deliverToUser(ctx context.Context, userID string, env *sharedv1.EventEnvelope, rawPayload []byte, remoteInstances *sync.Map) {
	log := observability.GetLogger(ctx)
	devResp, err := d.presenceClient.GetUserDevices(ctx, &presencev1.GetUserDevicesRequest{
		UserId: userID,
	})
	if err != nil {
		log.Error("dispatcher: presence lookup failed", zap.String("user_id", userID), zap.Error(err))
		return
	}

	for _, device := range devResp.GetDevices() {
		if device.InstanceId == d.instanceID {
			// Deliver to local session
			sessions := d.registry.GetUserSessions(userID)
			for _, s := range sessions {
				if s.DeviceID == device.DeviceId {
					if !s.Buffer(env, rawPayload) {
						if s.TrySend(rawPayload) {
							log.Info("dispatcher: local delivery success", zap.String("user_id", userID), zap.String("device_id", device.DeviceId))
						}
					}
				}
			}
		} else {
			remoteInstances.Store(device.InstanceId, struct{}{})
		}
	}
}

func (d *Dispatcher) handleConversationCreated(env *sharedv1.EventEnvelope) {
	var event conversationv1.ConversationCreatedEvent
	if err := proto.Unmarshal(env.GetPayload(), &event); err != nil {
		return
	}
	d.membership.SetMembers(event.GetConversation().GetConversationId(), event.GetParticipantUserIds())
}

func (d *Dispatcher) handleMembershipPreRoute(env *sharedv1.EventEnvelope) {
	var event conversationv1.MembershipChangedEvent
	if err := proto.Unmarshal(env.GetPayload(), &event); err != nil {
		return
	}
	if event.GetAdded() {
		d.membership.Add(event.GetConversationId(), event.GetUserId())
	}
}

func (d *Dispatcher) handleMembershipPostRoute(env *sharedv1.EventEnvelope) {
	var event conversationv1.MembershipChangedEvent
	if err := proto.Unmarshal(env.GetPayload(), &event); err != nil {
		return
	}
	if !event.GetAdded() {
		d.membership.Remove(event.GetConversationId(), event.GetUserId())
	}
}

func (d *Dispatcher) DeliverRemote(payload []byte) {
	ctx := context.Background()
	log := observability.GetLogger(ctx)
	var env sharedv1.EventEnvelope
	if err := proto.Unmarshal(payload, &env); err != nil {
		log.Error("dispatcher: error unmarshaling remote event", zap.Error(err))
		return
	}

	conversationID, err := d.getConversationID(&env)
	if err != nil {
		return
	}
	members, err := d.resolveMembers(ctx, conversationID)
	if err != nil {
		return
	}

	var wg sync.WaitGroup
	for _, userID := range members {
		wg.Add(1)
		go func(uid string) {
			defer wg.Done()
			sessions := d.registry.GetUserSessions(uid)
			for _, s := range sessions {
				if !s.Buffer(&env, payload) {
					if s.TrySend(payload) {
						log.Info("dispatcher: remote delivery (pubsub) success", zap.String("user_id", uid), zap.String("conversation_id", conversationID))
					}
				} else {
					log.Info("dispatcher: remote delivery (pubsub) buffered", zap.String("user_id", uid), zap.String("conversation_id", conversationID))
				}
			}
		}(userID)
	}
	wg.Wait()
}

func (d *Dispatcher) resolveMembers(ctx context.Context, conversationID string) ([]string, error) {
	log := observability.GetLogger(ctx)
	members := d.membership.Members(conversationID)
	if len(members) > 0 {
		return members, nil
	}

	// On-demand fetch if cache is empty (likely service restarted)
	log.Info("dispatcher: cache miss, fetching from conversation service", zap.String("conversation_id", conversationID))
	resp, err := d.convSvc.GetConversation(ctx, &conversationv1.GetConversationRequest{
		ConversationId: conversationID,
	})
	if err != nil {
		log.Error("dispatcher: failed to fetch members", zap.String("conversation_id", conversationID), zap.Error(err))
		return nil, err
	}

	d.membership.SetMembers(conversationID, resp.ParticipantUserIds)
	return resp.ParticipantUserIds, nil
}
