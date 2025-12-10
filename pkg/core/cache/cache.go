package cache

import (
	"sync"
	"time"
)

// Entry represents a cached item with expiration
type Entry struct {
	Value      interface{}
	Expiration time.Time
}

// IsExpired checks if the entry has expired
func (e *Entry) IsExpired() bool {
	if e.Expiration.IsZero() {
		return false // Never expires
	}
	return time.Now().After(e.Expiration)
}

// Cache is a thread-safe in-memory cache with TTL support
type Cache struct {
	mu       sync.RWMutex
	items    map[string]*Entry
	maxItems int
	ttl      time.Duration

	// Metrics
	hits   int64
	misses int64
}

// Config holds cache configuration
type Config struct {
	MaxItems int
	TTL      time.Duration
}

// DefaultConfig returns default cache configuration
func DefaultConfig() Config {
	return Config{
		MaxItems: 10000,
		TTL:      5 * time.Minute,
	}
}

// New creates a new cache instance
func New(cfg Config) *Cache {
	if cfg.MaxItems <= 0 {
		cfg.MaxItems = 10000
	}
	if cfg.TTL <= 0 {
		cfg.TTL = 5 * time.Minute
	}

	c := &Cache{
		items:    make(map[string]*Entry),
		maxItems: cfg.MaxItems,
		ttl:      cfg.TTL,
	}

	// Start cleanup goroutine
	go c.cleanupLoop()

	return c
}

// Get retrieves a value from the cache
func (c *Cache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	entry, exists := c.items[key]
	c.mu.RUnlock()

	if !exists {
		c.mu.Lock()
		c.misses++
		c.mu.Unlock()
		return nil, false
	}

	if entry.IsExpired() {
		c.mu.Lock()
		delete(c.items, key)
		c.misses++
		c.mu.Unlock()
		return nil, false
	}

	c.mu.Lock()
	c.hits++
	c.mu.Unlock()
	return entry.Value, true
}

// Set stores a value in the cache with the default TTL
func (c *Cache) Set(key string, value interface{}) {
	c.SetWithTTL(key, value, c.ttl)
}

// SetWithTTL stores a value with a custom TTL
func (c *Cache) SetWithTTL(key string, value interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Evict if at capacity (simple LRU: remove oldest)
	if len(c.items) >= c.maxItems {
		c.evictOldest()
	}

	var exp time.Time
	if ttl > 0 {
		exp = time.Now().Add(ttl)
	}

	c.items[key] = &Entry{
		Value:      value,
		Expiration: exp,
	}
}

// Delete removes a value from the cache
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, key)
}

// Clear removes all items from the cache
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = make(map[string]*Entry)
}

// Size returns the number of items in the cache
func (c *Cache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}

// Stats returns cache statistics
func (c *Cache) Stats() (hits, misses int64, hitRate float64) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	hits = c.hits
	misses = c.misses
	total := hits + misses
	if total > 0 {
		hitRate = float64(hits) / float64(total) * 100
	}
	return
}

// evictOldest removes the oldest entry (must be called with lock held)
func (c *Cache) evictOldest() {
	var oldestKey string
	var oldestTime time.Time

	for key, entry := range c.items {
		if oldestKey == "" || entry.Expiration.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.Expiration
		}
	}

	if oldestKey != "" {
		delete(c.items, oldestKey)
	}
}

// cleanupLoop periodically removes expired entries
func (c *Cache) cleanupLoop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.cleanup()
	}
}

// cleanup removes all expired entries
func (c *Cache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	for key, entry := range c.items {
		if entry.IsExpired() {
			delete(c.items, key)
		}
	}
}

// GetOrSet atomically gets a value or sets it if not present
func (c *Cache) GetOrSet(key string, fn func() (interface{}, error)) (interface{}, error) {
	// Try to get first
	if val, ok := c.Get(key); ok {
		return val, nil
	}

	// Compute the value
	val, err := fn()
	if err != nil {
		return nil, err
	}

	// Store and return
	c.Set(key, val)
	return val, nil
}

// GetOrSetWithTTL is like GetOrSet but with custom TTL
func (c *Cache) GetOrSetWithTTL(key string, ttl time.Duration, fn func() (interface{}, error)) (interface{}, error) {
	if val, ok := c.Get(key); ok {
		return val, nil
	}

	val, err := fn()
	if err != nil {
		return nil, err
	}

	c.SetWithTTL(key, val, ttl)
	return val, nil
}
