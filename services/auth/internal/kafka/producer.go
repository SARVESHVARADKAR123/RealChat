package kafka

import (
	"context"

	"github.com/segmentio/kafka-go"
)

// Producer wraps a kafka.Writer for publishing messages.
type Producer struct {
	w *kafka.Writer
}

// NewProducer creates a Kafka writer that routes messages by the topic set on
// each kafka.Message.
func NewProducer(brokers []string) *Producer {
	return &Producer{
		w: &kafka.Writer{
			Addr:     kafka.TCP(brokers...),
			Balancer: &kafka.LeastBytes{},
		},
	}
}

// Publish sends a single message to the given topic.
func (p *Producer) Publish(ctx context.Context, topic string, key, value []byte) error {
	return p.w.WriteMessages(ctx, kafka.Message{
		Topic: topic,
		Key:   key,
		Value: value,
	})
}

// Close flushes and closes the underlying writer.
func (p *Producer) Close() error { return p.w.Close() }
