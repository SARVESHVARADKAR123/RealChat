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
	DB         *sql.DB
	Producer   *kafka.Producer
	BatchSize  int
	PollDelay  time.Duration
	MaxRetries int
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
		SELECT id, aggregate_type, aggregate_id, event_type, payload, created_at, retry_count
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
		id            int64
		aggregateType string
		aggregateID   string
		eventType     string
		payload       []byte
		createdAt     time.Time
		retryCount    int
	}

	var events []event

	for rows.Next() {
		var e event
		if err := rows.Scan(&e.id, &e.aggregateType, &e.aggregateID, &e.eventType, &e.payload, &e.createdAt, &e.retryCount); err != nil {
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

	maxRetries := w.MaxRetries
	if maxRetries == 0 {
		maxRetries = 3 // default
	}

	var batchErr error

	// Publish each event
	for _, e := range events {
		if err := w.Producer.Publish(ctx, e.aggregateID, e.payload); err != nil {
			observability.OutboxPublishFailuresTotal.WithLabelValues("messaging", "messages").Inc()

			if e.retryCount >= maxRetries {
				_, dbErr := tx.ExecContext(ctx, `
					INSERT INTO outbox_dlq (id, aggregate_type, aggregate_id, event_type, payload, created_at, failed_at, error, retry_count)
					VALUES ($1, $2, $3, $4, $5, $6, now(), $7, $8)
				`, e.id, e.aggregateType, e.aggregateID, e.eventType, e.payload, e.createdAt, err.Error(), e.retryCount+1)
				if dbErr != nil {
					tx.Rollback()
					return dbErr
				}

				_, dbErr = tx.ExecContext(ctx, `
					DELETE FROM outbox_events WHERE id = $1
				`, e.id)
				if dbErr != nil {
					tx.Rollback()
					return dbErr
				}
			} else {
				_, dbErr := tx.ExecContext(ctx, `
					UPDATE outbox_events
					SET retry_count = retry_count + 1, error = $2
					WHERE id = $1
				`, e.id, err.Error())
				if dbErr != nil {
					tx.Rollback()
					return dbErr
				}
			}

			batchErr = err
			break
		} else {
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
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	return batchErr
}
