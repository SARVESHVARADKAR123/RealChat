package domain

import "errors"

var (
	ErrInvalidMessage  = errors.New("invalid message")
	ErrInvalidSequence = errors.New("invalid sequence")
	ErrMessageTooLarge = errors.New("message too large")
	ErrNotParticipant  = errors.New("user not participant")
	ErrMessageNotFound = errors.New("message not found")
	ErrInvalidInput    = errors.New("invalid input")
)
