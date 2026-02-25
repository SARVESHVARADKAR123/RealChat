package outbox

import (
	"context"
	"database/sql"
	"time"

	"github.com/SARVESHVARADKAR123/RealChat/services/conversation/internal/kafka"
	"github.com/SARVESHVARADKAR123/RealChat/services/conversation/internal/observability"
	"go.uber.org/zap"
)

type Worker struct {
	DB        *sql.DB
	Producer  *kafka.Producer
	BatchSize int
	PollDelay time.Duration
}

// Start Worker
func (w *Worker) Start(ctx context.Context) {

	log := observability.GetLogger(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		default:
			if err := w.processBatch(ctx); err != nil {
				log.Error("outbox error", zap.Error(err))
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
		SELECT id, aggregate_id, payload
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
		payload     []byte
	}

	var events []event

	for rows.Next() {
		var e event
		if err := rows.Scan(&e.id, &e.aggregateID, &e.payload); err != nil {
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

	// Publish each event
	for _, e := range events {
		if err := w.Producer.Publish(ctx, e.aggregateID, e.payload); err != nil {
			observability.OutboxPublishFailuresTotal.WithLabelValues("messaging", "messages").Inc()
			tx.Rollback()
			return err
		}

		_, err := tx.ExecContext(ctx, `
			UPDATE outbox_events
			SET processed_at = now()
			WHERE id = $1
		`, e.id)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}
