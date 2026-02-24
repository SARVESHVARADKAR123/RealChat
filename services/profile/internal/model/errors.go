package model

import "errors"

var (
	ErrProfileNotFound = errors.New("profile not found")
	ErrInvalidUpdate   = errors.New("invalid profile update")
)
