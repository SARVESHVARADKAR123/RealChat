package outbox

import (
	"context"
	"log"
	"time"

	"github.com/SARVESHVARADKAR123/RealChat/services/profile/internal/kafka"
)

// Publisher polls the outbox table and publishes unpublished events to Kafka.
type Publisher struct {
	repo     *Repository
	producer *kafka.Producer
}

// NewPublisher creates a new outbox publisher.
func NewPublisher(repo *Repository, producer *kafka.Producer) *Publisher {
	return &Publisher{repo: repo, producer: producer}
}

// Start begins the polling loop. It blocks until the context is cancelled.
func (p *Publisher) Start(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.publishBatch(ctx)
		}
	}
}

func (p *Publisher) publishBatch(ctx context.Context) {
	rows, err := p.repo.Fetch(ctx, 50)
	if err != nil {
		log.Println("outbox query error:", err)
		return
	}

	for _, row := range rows {
		if err := p.producer.Publish(ctx, row.Topic, []byte(row.Key), row.Payload); err != nil {
			log.Println("kafka publish failed:", err)
			continue
		}

		if err := p.repo.MarkPublished(ctx, row.ID); err != nil {
			log.Println("outbox mark published error:", err)
		}
	}
}
