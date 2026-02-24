package outbox

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/SARVESHVARADKAR123/RealChat/services/auth/internal/kafka"
)

type Worker struct {
	DB        *sql.DB
	Producer  *kafka.Producer
	BatchSize int
	PollDelay time.Duration
}

func NewWorker(db *sql.DB, p *kafka.Producer, batchSize int, delay time.Duration) *Worker {
	return &Worker{
		DB:        db,
		Producer:  p,
		BatchSize: batchSize,
		PollDelay: delay,
	}
}

func (w *Worker) Start(ctx context.Context) {
	log.Println("auth outbox worker started")
	for {
		select {
		case <-ctx.Done():
			log.Println("auth outbox worker stopping")
			return
		default:
			if err := w.processBatch(ctx); err != nil {
				log.Printf("auth outbox worker error: %v", err)
				time.Sleep(time.Second)
			}
		}
	}
}

func (w *Worker) processBatch(ctx context.Context) error {
	tx, err := w.DB.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelReadCommitted,
	})
	if err != nil {
		return err
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT id, aggregate_id, event_type, payload
		FROM outbox_events
		WHERE processed_at IS NULL
		ORDER BY id
		FOR UPDATE SKIP LOCKED
		LIMIT $1
	`, w.BatchSize)

	if err != nil {
		tx.Rollback()
		return err
	}
	defer rows.Close()

	type event struct {
		id          int64
		aggregateID string
		eventType   string
		payload     []byte
	}

	var events []event
	for rows.Next() {
		var e event
		if err := rows.Scan(&e.id, &e.aggregateID, &e.eventType, &e.payload); err != nil {
			tx.Rollback()
			return err
		}
		events = append(events, e)
	}

	if len(events) == 0 {
		tx.Rollback()
		time.Sleep(w.PollDelay)
		return nil
	}

	for _, e := range events {
		topic := ""
		switch e.eventType {
		case "USER_CREATED":
			topic = "auth.user.created"
		default:
			log.Printf("unknown event type in outbox: %s", e.eventType)
			continue
		}

		if err := w.Producer.Publish(ctx, topic, []byte(e.aggregateID), e.payload); err != nil {
			tx.Rollback()
			return err
		}

		_, err := tx.ExecContext(ctx, `
			UPDATE outbox_events
			SET processed_at = NOW()
			WHERE id = $1
		`, e.id)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}
