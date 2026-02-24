package model

import "time"

type Block struct {
	UserID    string
	BlockedID string
	CreatedAt time.Time
}
