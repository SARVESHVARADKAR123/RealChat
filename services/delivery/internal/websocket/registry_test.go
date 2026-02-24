package websocket

import (
	"testing"
)

func TestRegistry_SessionReplacement(t *testing.T) {
	r := NewRegistry()

	u1 := "user1"
	d1 := "device1"

	s1 := NewSession("s1", u1, d1, nil)
	r.Add(s1)

	// Verify s1 is in registry
	sessions := r.GetUserSessions(u1)
	if len(sessions) != 1 || sessions[0].ID != "s1" {
		t.Errorf("Expected session s1, got %v", sessions)
	}

	// Add s2 for same user/device
	s2 := NewSession("s2", u1, d1, nil)
	r.Add(s2)

	// Verify s1 is closed (done channel closed)
	select {
	case <-s1.Done():
		// OK
	default:
		t.Error("Old session s1 should have been closed")
	}

	// Verify only s2 is in registry
	sessions = r.GetUserSessions(u1)
	if len(sessions) != 1 || sessions[0].ID != "s2" {
		t.Errorf("Expected only session s2, got %v", sessions)
	}

	// Remove s1 (simulating late cleanup of old session)
	r.Remove(s1)

	// Verify s2 is STILL in registry
	sessions = r.GetUserSessions(u1)
	if len(sessions) != 1 || sessions[0].ID != "s2" {
		t.Errorf("Session s2 should still be in registry after late Remove(s1), got %v", sessions)
	}

	// Remove s2
	r.Remove(s2)

	// Verify registry is empty for u1
	sessions = r.GetUserSessions(u1)
	if len(sessions) != 0 {
		t.Errorf("Expected no sessions for u1, got %v", sessions)
	}
}
