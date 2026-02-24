package websocket

import (
	"testing"
	"time"

	messagingv1 "github.com/SARVESHVARADKAR123/RealChat/contracts/gen/go/messaging/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestSessionBuffering(t *testing.T) {
	s := NewSession("s1", "u1", "d1", nil)

	// Mock events
	events := []struct {
		seq int64
		ts  time.Time
	}{
		{seq: 3, ts: time.Now().Add(3 * time.Second)},
		{seq: 1, ts: time.Now().Add(1 * time.Second)},
		{seq: 2, ts: time.Now().Add(2 * time.Second)},
	}

	for _, e := range events {
		msg := &messagingv1.Message{
			Sequence: e.seq,
		}
		event := &messagingv1.MessageSentEvent{
			Message: msg,
		}
		eventPayload, _ := proto.Marshal(event)
		env := &messagingv1.MessagingEventEnvelope{
			EventType:  messagingv1.MessagingEventType_MESSAGING_EVENT_TYPE_MESSAGE_SENT,
			OccurredAt: timestamppb.New(e.ts),
			Payload:    eventPayload,
		}
		payload, _ := proto.Marshal(env)
		s.Buffer(env, payload)
	}

	s.FlushBufferSorted()

	if !s.IsReady() {
		t.Error("Session should be ready after flush")
	}

	// Should have 3 messages in queue
	if len(s.SendQueue) != 3 {
		t.Errorf("Expected 3 messages in queue, got %d", len(s.SendQueue))
	}

	// Verify order (sorted by sequence)
	for i := int64(1); i <= 3; i++ {
		payload := <-s.SendQueue
		var env messagingv1.MessagingEventEnvelope
		if err := proto.Unmarshal(payload, &env); err != nil {
			t.Fatalf("Failed to unmarshal env: %v", err)
		}
		var event messagingv1.MessageSentEvent
		if err := proto.Unmarshal(env.Payload, &event); err != nil {
			t.Fatalf("Failed to unmarshal event: %v", err)
		}
		if event.Message.Sequence != i {
			t.Errorf("Expected sequence %d, got %d", i, event.Message.Sequence)
		}
	}
}

func TestSessionBuffering_Mixed(t *testing.T) {
	s := NewSession("s1", "u1", "d1", nil)

	t1 := time.Now().Add(100 * time.Millisecond)
	t2 := time.Now().Add(200 * time.Millisecond)

	// Event 1: Message sequence 2, at t2
	msg1 := &messagingv1.Message{Sequence: 2}
	event1 := &messagingv1.MessageSentEvent{Message: msg1}
	eventPayload1, _ := proto.Marshal(event1)
	env1 := &messagingv1.MessagingEventEnvelope{
		EventType:  messagingv1.MessagingEventType_MESSAGING_EVENT_TYPE_MESSAGE_SENT,
		OccurredAt: timestamppb.New(t2),
		Payload:    eventPayload1,
	}
	s.Buffer(env1, []byte("payload1"))

	// Event 2: Generic event at t1
	env2 := &messagingv1.MessagingEventEnvelope{
		EventType:  messagingv1.MessagingEventType_MESSAGING_EVENT_TYPE_READ_RECEIPT_UPDATED,
		OccurredAt: timestamppb.New(t1),
	}
	s.Buffer(env2, []byte("payload2"))

	s.FlushBufferSorted()

	// env2 (seq 0) vs env1 (seq 2): t1 vs t2 -> env2 comes first
	p1 := <-s.SendQueue
	if string(p1) != "payload2" {
		t.Errorf("Expected payload2, got %s", string(p1))
	}

	p2 := <-s.SendQueue
	if string(p2) != "payload1" {
		t.Errorf("Expected payload1, got %s", string(p2))
	}
}
