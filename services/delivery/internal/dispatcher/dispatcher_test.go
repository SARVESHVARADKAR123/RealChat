package dispatcher

import (
	"context"
	"testing"

	messagingv1 "github.com/SARVESHVARADKAR123/RealChat/contracts/gen/go/messaging/v1"
	"github.com/SARVESHVARADKAR123/RealChat/services/delivery/internal/membership"
	"github.com/SARVESHVARADKAR123/RealChat/services/delivery/internal/websocket"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

type mockMessagingClient struct {
	messagingv1.MessagingApiClient
	getConvFunc func(ctx context.Context, in *messagingv1.GetConversationRequest) (*messagingv1.GetConversationResponse, error)
}

func (m *mockMessagingClient) GetConversation(ctx context.Context, in *messagingv1.GetConversationRequest, opts ...grpc.CallOption) (*messagingv1.GetConversationResponse, error) {
	return m.getConvFunc(ctx, in)
}

func TestDeliverRemote_CacheMiss(t *testing.T) {
	reg := websocket.NewRegistry()
	mem := membership.New()

	// mock client
	client := &mockMessagingClient{
		getConvFunc: func(ctx context.Context, in *messagingv1.GetConversationRequest) (*messagingv1.GetConversationResponse, error) {
			return &messagingv1.GetConversationResponse{
				ParticipantUserIds: []string{"user1"},
			}, nil
		},
	}

	d := New(reg, mem, nil, nil, "inst1", client)

	// Add local session for user1
	s := websocket.NewSession("s1", "user1", "d1", nil)
	s.SetReady()
	reg.Add(s)

	// Remote event
	event := &messagingv1.MessageSentEvent{
		Message: &messagingv1.Message{
			ConversationId: "conv1",
			SenderUserId:   "user2",
			Content:        "hello",
		},
	}
	payload, _ := proto.Marshal(event)
	env := &messagingv1.MessagingEventEnvelope{
		EventType:     messagingv1.MessagingEventType_MESSAGING_EVENT_TYPE_MESSAGE_SENT,
		Payload:       payload,
		SchemaVersion: 1,
	}
	raw, _ := proto.Marshal(env)

	// Cache is empty initially
	if len(mem.Members("conv1")) != 0 {
		t.Fatal("Cache should be empty")
	}

	// Trigger DeliverRemote
	d.DeliverRemote(raw)

	// Cache should now be populated
	members := mem.Members("conv1")
	if len(members) != 1 || members[0] != "user1" {
		t.Errorf("Expected 1 member (user1), got %v", members)
	}

	// Session should have message in queue
	if len(s.SendQueue) != 1 {
		t.Error("Message should have been delivered to session")
	}
}
