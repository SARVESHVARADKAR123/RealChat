package model

import "time"

type Outbox struct {
	ID        int64
	EventType string
	Payload   []byte
	CreatedAt time.Time
	Processed bool
}
