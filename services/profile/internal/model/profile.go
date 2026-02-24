package model

import "time"

type Profile struct {
	UserID      string
	Username    string
	DisplayName string
	Bio         string
	AvatarURL   string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
