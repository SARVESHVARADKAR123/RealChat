package repository

import (
	"context"
	"database/sql"

	_ "github.com/lib/pq"
)

func NewDB(ctx context.Context, url string) (*sql.DB, error) {
	db, err := sql.Open("postgres", url)
	if err != nil {
		return nil, err
	}
	return db, db.PingContext(ctx)
}
