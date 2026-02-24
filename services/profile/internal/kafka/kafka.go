package kafka

import (
	"context"
	"encoding/json"
	"log"

	"github.com/segmentio/kafka-go"
)

// --------------- Producer ---------------

// Producer wraps a kafka.Writer for publishing messages.
type Producer struct {
	w *kafka.Writer
}

// NewProducer creates a Kafka writer that routes messages by the topic set on
// each kafka.Message.  Balancer is round-robin by default.
func NewProducer(brokers string) *Producer {
	return &Producer{
		w: &kafka.Writer{
			Addr:     kafka.TCP(brokers),
			Balancer: &kafka.LeastBytes{},
		},
	}
}

// Publish sends a single message to the pre-configured topic.
func (p *Producer) Publish(ctx context.Context, topic string, key, value []byte) error {
	return p.w.WriteMessages(ctx, kafka.Message{
		Topic: topic,
		Key:   key,
		Value: value,
	})
}

// Close flushes and closes the underlying writer.
func (p *Producer) Close() error { return p.w.Close() }

// --------------- Consumer ---------------

type userCreated struct {
	UserID string `json:"user_id"`
}

// StartUserCreatedConsumer listens on "auth.user.created" and creates a
// profile row for every new user (idempotent via ON CONFLICT DO NOTHING).
func StartUserCreatedConsumer(ctx context.Context, brokers string, repo interface {
	CreateIfNotExists(context.Context, string) error
}) {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{brokers},
		Topic:   "auth.user.created",
		GroupID: "profile",
	})
	defer r.Close()

	for {
		m, err := r.ReadMessage(ctx)
		if err != nil {
			return
		}

		var e userCreated
		if err := json.Unmarshal(m.Value, &e); err != nil {
			log.Println("bad user.created payload:", err)
			continue
		}

		if err := repo.CreateIfNotExists(ctx, e.UserID); err != nil {
			log.Println("idempotent create failed:", err)
		}
	}
}
