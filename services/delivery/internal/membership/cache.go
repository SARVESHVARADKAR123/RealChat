package membership

import "sync"

type Cache struct {
	mu          sync.RWMutex
	data        map[string]map[string]struct{}
	userToConvs map[string]map[string]struct{}
}

// New creates a new in-memory cache for conversation memberships.
func New() *Cache {
	return &Cache{
		data:        make(map[string]map[string]struct{}),
		userToConvs: make(map[string]map[string]struct{}),
	}
}

// Add adds a user to the cache for a specific conversation.
func (c *Cache) Add(conv, user string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.data[conv] == nil {
		c.data[conv] = make(map[string]struct{})
	}
	c.data[conv][user] = struct{}{}

	if c.userToConvs[user] == nil {
		c.userToConvs[user] = make(map[string]struct{})
	}
	c.userToConvs[user][conv] = struct{}{}
}

// Remove removes a user from the cache for a specific conversation.
func (c *Cache) Remove(conv, user string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.data[conv] != nil {
		delete(c.data[conv], user)
	}
	if c.userToConvs[user] != nil {
		delete(c.userToConvs[user], conv)
	}
}

func (c *Cache) Members(conv string) []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	var out []string
	for u := range c.data[conv] {
		out = append(out, u)
	}
	return out
}

func (c *Cache) UserConvs(userID string) []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	var out []string
	for conv := range c.userToConvs[userID] {
		out = append(out, conv)
	}
	return out
}

func (c *Cache) SetMembers(conv string, users []string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Cleanup old membership for these users in this conversation?
	// Easier to just rebuild everything carefully if needed,
	// but SetMembers is usually for a full sync.

	// Remove this conv from all users currently in this conv
	if oldMembers, ok := c.data[conv]; ok {
		for userID := range oldMembers {
			if convs, ok := c.userToConvs[userID]; ok {
				delete(convs, conv)
			}
		}
	}

	m := make(map[string]struct{})
	for _, u := range users {
		m[u] = struct{}{}
		if c.userToConvs[u] == nil {
			c.userToConvs[u] = make(map[string]struct{})
		}
		c.userToConvs[u][conv] = struct{}{}
	}
	c.data[conv] = m
}
