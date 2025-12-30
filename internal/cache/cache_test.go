package cache

import (
	"testing"
	"time"
)

func TestCacheTTL(t *testing.T) {
	c := NewMemoryCache(100 * time.Millisecond)
	c.Set("k", "v")
	if _, ok := c.Get("k"); !ok {
		t.Fatal("expected to find key immediately")
	}
	time.Sleep(150 * time.Millisecond)
	if _, ok := c.Get("k"); ok {
		t.Fatal("expected key to expire")
	}
}
