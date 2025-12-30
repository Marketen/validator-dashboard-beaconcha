package cache

import (
	"sync"
	"time"
)

type item struct {
	value     interface{}
	expiresAt time.Time
}

type MemoryCache struct {
	mu    sync.RWMutex
	store map[string]item
	ttl   time.Duration
}

func NewMemoryCache(ttl time.Duration) *MemoryCache {
	c := &MemoryCache{store: make(map[string]item), ttl: ttl}
	go c.janitor()
	return c
}

func (c *MemoryCache) Set(key string, v interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.store[key] = item{value: v, expiresAt: time.Now().Add(c.ttl)}
}

func (c *MemoryCache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	it, ok := c.store[key]
	c.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if time.Now().After(it.expiresAt) {
		c.mu.Lock()
		delete(c.store, key)
		c.mu.Unlock()
		return nil, false
	}
	return it.value, true
}

func (c *MemoryCache) janitor() {
	ticker := time.NewTicker(time.Minute)
	for range ticker.C {
		now := time.Now()
		c.mu.Lock()
		for k, it := range c.store {
			if now.After(it.expiresAt) {
				delete(c.store, k)
			}
		}
		c.mu.Unlock()
	}
}
