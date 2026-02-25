package domain

import "time"

type ConversationType string

const (
	ConversationDirect ConversationType = "direct"
	ConversationGroup  ConversationType = "group"
)

type Role string

const (
	RoleMember Role = "member"
	RoleAdmin  Role = "admin"
)

type Participant struct {
	UserID string
	Role   Role
}

// Conversation Invariants:
// 1. Membership (Direct): Exactly 2 participants.
// 2. Membership (Group): At least 1 admin. Cannot remove last admin.
// 3. Modification: Only admins can Add/Remove participants.
type Conversation struct {
	ID           string
	Type         ConversationType
	DisplayName  string
	AvatarURL    string
	CreatedAt    time.Time
	Participants map[string]Participant
}

func (c *Conversation) CanSend(userID string) error {
	if _, ok := c.Participants[userID]; !ok {
		return ErrNotParticipant
	}
	return nil
}

func (c *Conversation) AddParticipant(requesterID, userID string) error {
	if c.Type != ConversationGroup {
		return ErrDirectModification
	}

	req, ok := c.Participants[requesterID]
	if !ok || req.Role != RoleAdmin {
		return ErrNotAdmin
	}

	c.Participants[userID] = Participant{
		UserID: userID,
		Role:   RoleMember,
	}
	return nil
}

func (c *Conversation) RemoveParticipant(requesterID, targetID string) error {
	if c.Type != ConversationGroup {
		return ErrDirectModification
	}

	req, ok := c.Participants[requesterID]
	if !ok || req.Role != RoleAdmin {
		return ErrNotAdmin
	}

	delete(c.Participants, targetID)
	return nil
}
