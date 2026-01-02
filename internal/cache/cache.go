// Package cache provides in-memory caching with TTL for the validator-dashboard.
package cache

import (
	"sync"
	"time"
)

// Cache is a thread-safe in-memory cache with TTL support.
// It caches API responses to reduce calls to the Beaconcha API.
type Cache struct {
	items map[string]*cacheItem
	mu    sync.RWMutex
	ttl   time.Duration
}

type cacheItem struct {
	value      interface{}
	expiration time.Time
}

// New creates a new cache with the specified TTL.
func New(ttl time.Duration) *Cache {
	c := &Cache{
		items: make(map[string]*cacheItem),
		ttl:   ttl,
	}

	// Start cleanup goroutine
	go c.cleanupLoop()

	return c
}

// Get retrieves a value from the cache.
// Returns the value and true if found and not expired, nil and false otherwise.
func (c *Cache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, exists := c.items[key]
	if !exists {
		return nil, false
	}

	if time.Now().After(item.expiration) {
		return nil, false
	}

	return item.value, true
}

// Set stores a value in the cache with the default TTL.
func (c *Cache) Set(key string, value interface{}) {
	c.SetWithTTL(key, value, c.ttl)
}

// SetWithTTL stores a value in the cache with a custom TTL.
func (c *Cache) SetWithTTL(key string, value interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[key] = &cacheItem{
		value:      value,
		expiration: time.Now().Add(ttl),
	}
}

// Delete removes a value from the cache.
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.items, key)
}

// Clear removes all items from the cache.
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]*cacheItem)
}

// Size returns the number of items in the cache (including expired ones).
func (c *Cache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.items)
}

// GetOrSet returns the cached value if present, otherwise calls the loader function
// to compute the value, stores it in the cache, and returns it.
func (c *Cache) GetOrSet(key string, loader func() (interface{}, error)) (interface{}, error) {
	// First, try to get from cache
	if value, ok := c.Get(key); ok {
		return value, nil
	}

	// Not in cache, need to load
	// Use a lock to prevent thundering herd
	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring lock
	if item, exists := c.items[key]; exists && time.Now().Before(item.expiration) {
		return item.value, nil
	}

	// Load the value
	value, err := loader()
	if err != nil {
		return nil, err
	}

	// Store in cache
	c.items[key] = &cacheItem{
		value:      value,
		expiration: time.Now().Add(c.ttl),
	}

	return value, nil
}

// cleanupLoop periodically removes expired items from the cache.
func (c *Cache) cleanupLoop() {
	ticker := time.NewTicker(c.ttl / 2) // Cleanup at half the TTL interval
	defer ticker.Stop()

	for range ticker.C {
		c.cleanup()
	}
}

// cleanup removes all expired items from the cache.
func (c *Cache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, item := range c.items {
		if now.After(item.expiration) {
			delete(c.items, key)
		}
	}
}
