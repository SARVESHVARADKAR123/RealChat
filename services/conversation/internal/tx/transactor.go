package tx

import (
	"context"
	"database/sql"
)

type Transactor interface {
	WithTx(ctx context.Context, fn func(ctx context.Context, tx *sql.Tx) error) error
}
