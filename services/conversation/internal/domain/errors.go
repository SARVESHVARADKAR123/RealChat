package domain

import "errors"

var (
	ErrInvalidMessage       = errors.New("invalid message")
	ErrInvalidSequence      = errors.New("invalid sequence")
	ErrMessageTooLarge      = errors.New("message too large")
	ErrNotParticipant       = errors.New("user not participant")
	ErrNotAdmin             = errors.New("admin privileges required")
	ErrDirectModification   = errors.New("cannot modify direct conversation")
	ErrConversationNotFound = errors.New("conversation not found")
	ErrMessageNotFound      = errors.New("message not found")
	ErrInvalidInput         = errors.New("invalid input")
	ErrLastAdmin            = errors.New("cannot remove last admin")
)
