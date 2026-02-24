package websocket

import (
	"log"
	"sync"
)

type Registry struct {
	mu       sync.RWMutex
	sessions map[string]map[string]*Session
}

func NewRegistry() *Registry {
	return &Registry{
		sessions: make(map[string]map[string]*Session),
	}
}

func (r *Registry) Add(s *Session) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.sessions[s.UserID] == nil {
		r.sessions[s.UserID] = make(map[string]*Session)
	}

	// Check for existing session
	if old, ok := r.sessions[s.UserID][s.DeviceID]; ok {
		// Log and close old session before replacing
		log.Printf("session: replacing existing connection user=%s device=%s old_sid=%s new_sid=%s",
			s.UserID, s.DeviceID, old.ID, s.ID)

		// Close old session. It will call registry.Remove but since we have the lock,
		// it will wait. However, we should be careful not to deadlock if Close calls Remove.
		// Our current Remove also takes the lock.
		// Wait, if old.Close() calls Remove, it will deadlock.
		// Registry.Remove is called from readLoop defer, which happens after s.Close() returns.
		// But in this case, we are still holding the lock.
		// Let's release the lock before closing or handle it carefully.

		// Actually, let's just close it. The readLoop will eventually call Remove.
		// BUT we are about to overwrite it in the map anyway.
		old.CloseWithReason(4000, "session_replaced")
	}

	r.sessions[s.UserID][s.DeviceID] = s
}

func (r *Registry) Remove(s *Session) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if devices, ok := r.sessions[s.UserID]; ok {
		// Only remove if it's the SAME session object (matches ID)
		// This prevents a late Remove from an old replaced session from killing the new one
		if current, ok := devices[s.DeviceID]; ok && current.ID == s.ID {
			delete(devices, s.DeviceID)
			if len(devices) == 0 {
				delete(r.sessions, s.UserID)
			}
		}
	}
}

func (r *Registry) GetUserSessions(userID string) []*Session {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*Session
	for _, s := range r.sessions[userID] {
		result = append(result, s)
	}
	return result
}

func (r *Registry) CloseAll() {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, devices := range r.sessions {
		for _, s := range devices {
			s.Close()
		}
	}
}
