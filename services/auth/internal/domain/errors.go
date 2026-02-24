package domain

import "errors"

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserNotFound       = errors.New("user not found")
	ErrEmailConflict      = errors.New("email already in use")
	ErrInvalidToken       = errors.New("invalid or expired token")
)
