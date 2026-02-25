package application

import (
	"context"
	"database/sql"
	"fmt"
)

// NextSequence atomically increments and returns the next message sequence
// number for the given conversation. Called by the message service via gRPC.
func (s *Service) NextSequence(ctx context.Context, conversationID string) (int64, error) {
	var seq int64
	err := s.tx.WithTx(ctx, func(ctx context.Context, tx *sql.Tx) error {
		var err error
		seq, err = s.repo.NextSequence(ctx, tx, conversationID)
		if err != nil {
			return fmt.Errorf("failed to increment sequence for conversation %s: %w", conversationID, err)
		}
		return nil
	})
	return seq, err
}
