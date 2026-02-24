package dispatcher

import (
	"context"
	"errors"

	messagingv1 "github.com/SARVESHVARADKAR123/RealChat/contracts/gen/go/messaging/v1"
	"github.com/SARVESHVARADKAR123/RealChat/services/delivery/internal/membership"
	"github.com/SARVESHVARADKAR123/RealChat/services/delivery/internal/observability"
	"github.com/SARVESHVARADKAR123/RealChat/services/delivery/internal/presence"
	"github.com/SARVESHVARADKAR123/RealChat/services/delivery/internal/router"
	"github.com/SARVESHVARADKAR123/RealChat/services/delivery/internal/websocket"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

type Dispatcher struct {
	registry   *websocket.Registry
	membership *membership.Cache
	presence   *presence.Presence
	router     *router.Router
	instanceID string
	mSvc       messagingv1.MessagingApiClient
}

func New(registry *websocket.Registry, membership *membership.Cache,
	presence *presence.Presence, router *router.Router, instanceID string, mSvc messagingv1.MessagingApiClient) *Dispatcher {
	return &Dispatcher{
		registry:   registry,
		membership: membership,
		presence:   presence,
		router:     router,
		instanceID: instanceID,
		mSvc:       mSvc,
	}
}

func (d *Dispatcher) Handle(ctx context.Context, record []byte) {
	log := observability.GetLogger(ctx)
	var env messagingv1.MessagingEventEnvelope
	if err := proto.Unmarshal(record, &env); err != nil {
		log.Error("dispatcher: error unmarshaling event", zap.Error(err))
		return
	}

	switch env.GetEventType() {
	case messagingv1.MessagingEventType_MESSAGING_EVENT_TYPE_MESSAGE_SENT,
		messagingv1.MessagingEventType_MESSAGING_EVENT_TYPE_MESSAGE_DELETED,
		messagingv1.MessagingEventType_MESSAGING_EVENT_TYPE_READ_RECEIPT_UPDATED,
		messagingv1.MessagingEventType_MESSAGING_EVENT_TYPE_MEMBERSHIP_CHANGED,
		messagingv1.MessagingEventType_MESSAGING_EVENT_TYPE_CONVERSATION_CREATED:
		d.handleEvent(ctx, &env, record)
	}
}

func (d *Dispatcher) getConversationID(env *messagingv1.MessagingEventEnvelope) (string, error) {
	switch env.GetEventType() {
	case messagingv1.MessagingEventType_MESSAGING_EVENT_TYPE_MESSAGE_SENT:
		var event messagingv1.MessageSentEvent
		if err := proto.Unmarshal(env.GetPayload(), &event); err != nil {
			return "", err
		}
		return event.GetMessage().GetConversationId(), nil

	case messagingv1.MessagingEventType_MESSAGING_EVENT_TYPE_MESSAGE_DELETED:
		var event messagingv1.MessageDeletedEvent
		if err := proto.Unmarshal(env.GetPayload(), &event); err != nil {
			return "", err
		}
		return event.GetConversationId(), nil

	case messagingv1.MessagingEventType_MESSAGING_EVENT_TYPE_READ_RECEIPT_UPDATED:
		var event messagingv1.ReadReceiptUpdatedEvent
		if err := proto.Unmarshal(env.GetPayload(), &event); err != nil {
			return "", err
		}
		return event.GetConversationId(), nil

	case messagingv1.MessagingEventType_MESSAGING_EVENT_TYPE_MEMBERSHIP_CHANGED:
		var event messagingv1.MembershipChangedEvent
		if err := proto.Unmarshal(env.GetPayload(), &event); err != nil {
			return "", err
		}
		return event.GetConversationId(), nil

	case messagingv1.MessagingEventType_MESSAGING_EVENT_TYPE_CONVERSATION_CREATED:
		var event messagingv1.ConversationCreatedEvent
		if err := proto.Unmarshal(env.GetPayload(), &event); err != nil {
			return "", err
		}
		return event.GetConversation().GetConversationId(), nil

	default:
		return "", errors.New("unsupported event type")
	}
}

func (d *Dispatcher) handleEvent(ctx context.Context, env *messagingv1.MessagingEventEnvelope, rawPayload []byte) {
	log := observability.GetLogger(ctx)
	// If membership ADDED -> update cache BEFORE routing so new member gets the event
	if env.GetEventType() == messagingv1.MessagingEventType_MESSAGING_EVENT_TYPE_CONVERSATION_CREATED {
		d.handleConversationCreated(env)
	}

	if env.GetEventType() == messagingv1.MessagingEventType_MESSAGING_EVENT_TYPE_MEMBERSHIP_CHANGED {
		d.handleMembershipPreRoute(env)
	}

	conversationID, err := d.getConversationID(env)
	if err != nil {
		log.Error("dispatcher: fail to get convID", zap.Error(err))
		return
	}

	members := d.membership.Members(conversationID)
	if len(members) == 0 {
		// On-demand fetch if cache is empty (likely service restarted)
		log.Info("dispatcher: cache miss, fetching from message service", zap.String("conversation_id", conversationID))
		resp, err := d.mSvc.GetConversation(ctx, &messagingv1.GetConversationRequest{
			ConversationId: conversationID,
		})
		if err != nil {
			log.Error("dispatcher: failed to fetch missing conversation", zap.String("conversation_id", conversationID), zap.Error(err))
			return
		}
		d.membership.SetMembers(conversationID, resp.ParticipantUserIds)
		members = resp.ParticipantUserIds
	}

	remoteInstances := make(map[string]struct{})

	for _, userID := range members {
		devices, err := d.presence.GetUserDevices(ctx, userID)
		if err != nil {
			log.Error("dispatcher: presence lookup failed", zap.String("user_id", userID), zap.Error(err))
			continue
		}

		for deviceID, instance := range devices {
			log.Info("routing", zap.String("user_id", userID), zap.String("target_instance", instance), zap.String("current_instance", d.instanceID))
			if instance == d.instanceID {
				// Deliver to local session
				sessions := d.registry.GetUserSessions(userID)
				for _, s := range sessions {
					if s.DeviceID == deviceID {
						if !s.Buffer(env, rawPayload) {
							if s.TrySend(rawPayload) {
								log.Info("dispatcher: local delivery success", zap.String("user_id", userID), zap.String("device_id", deviceID))
							}
						}
					}
				}
			} else {
				remoteInstances[instance] = struct{}{}
			}
		}
	}

	// Publish to each remote instance ONLY ONCE
	for instance := range remoteInstances {
		if err := d.router.Publish(ctx, instance, rawPayload); err != nil {
			log.Error("dispatcher: remote routing failed", zap.String("instance", instance), zap.Error(err))
		} else {
			log.Info("dispatcher: remote routing success", zap.String("instance", instance))
		}
	}

	// If membership REMOVED -> update cache AFTER routing so removed member got the last event
	if env.GetEventType() == messagingv1.MessagingEventType_MESSAGING_EVENT_TYPE_MEMBERSHIP_CHANGED {
		d.handleMembershipPostRoute(env)
	}
}

func (d *Dispatcher) handleConversationCreated(env *messagingv1.MessagingEventEnvelope) {
	var event messagingv1.ConversationCreatedEvent
	if err := proto.Unmarshal(env.GetPayload(), &event); err != nil {
		return
	}
	d.membership.SetMembers(event.GetConversation().GetConversationId(), event.GetParticipantUserIds())
}

func (d *Dispatcher) handleMembershipPreRoute(env *messagingv1.MessagingEventEnvelope) {
	var event messagingv1.MembershipChangedEvent
	if err := proto.Unmarshal(env.GetPayload(), &event); err != nil {
		return
	}
	if event.GetAdded() {
		d.membership.Add(event.GetConversationId(), event.GetUserId())
	}
}

func (d *Dispatcher) handleMembershipPostRoute(env *messagingv1.MessagingEventEnvelope) {
	var event messagingv1.MembershipChangedEvent
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
	var env messagingv1.MessagingEventEnvelope
	if err := proto.Unmarshal(payload, &env); err != nil {
		log.Error("dispatcher: error unmarshaling remote event", zap.Error(err))
		return
	}

	conversationID, err := d.getConversationID(&env)
	if err != nil {
		return
	}
	members := d.membership.Members(conversationID)
	if len(members) == 0 {
		log.Info("dispatcher: remote (pubsub) cache miss, fetching from message service", zap.String("conversation_id", conversationID))
		resp, err := d.mSvc.GetConversation(ctx, &messagingv1.GetConversationRequest{
			ConversationId: conversationID,
		})
		if err != nil {
			log.Error("dispatcher: remote (pubsub) failed to fetch missing conversation", zap.String("conversation_id", conversationID), zap.Error(err))
			return
		}
		d.membership.SetMembers(conversationID, resp.ParticipantUserIds)
		members = resp.ParticipantUserIds
	}

	for _, userID := range members {
		sessions := d.registry.GetUserSessions(userID)
		for _, s := range sessions {
			// Double check device is actually on this instance
			// We skip the presence.GetUserDevices check here for performance,
			// relying on the registry being accurate for this replica.
			if !s.Buffer(&env, payload) {
				if s.TrySend(payload) {
					log.Info("dispatcher: remote delivery (pubsub) success", zap.String("user_id", userID), zap.String("conversation_id", conversationID))
				}
			} else {
				log.Info("dispatcher: remote delivery (pubsub) buffered", zap.String("user_id", userID), zap.String("conversation_id", conversationID))
			}
		}
	}
}
