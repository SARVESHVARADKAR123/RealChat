package model

import "time"

type Contact struct {
	UserID    string
	ContactID string
	CreatedAt time.Time
}
