// Package cache provides tests for the cache.
package cache

import (
	"sync"
	"testing"
	"time"
)

func TestCache_SetAndGet(t *testing.T) {
	c := New(time.Minute)

	c.Set("key1", "value1")

	value, ok := c.Get("key1")
	if !ok {
		t.Fatal("expected to find key1")
	}
	if value != "value1" {
		t.Errorf("expected 'value1', got '%v'", value)
	}
}

func TestCache_GetMissing(t *testing.T) {
	c := New(time.Minute)

	_, ok := c.Get("nonexistent")
	if ok {
		t.Error("expected key not to be found")
	}
}

func TestCache_Expiration(t *testing.T) {
	c := New(50 * time.Millisecond)

	c.Set("key1", "value1")

	// Should be present immediately
	if _, ok := c.Get("key1"); !ok {
		t.Error("key should be present immediately")
	}

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Should be expired now
	if _, ok := c.Get("key1"); ok {
		t.Error("key should be expired")
	}
}

func TestCache_SetWithCustomTTL(t *testing.T) {
	c := New(time.Minute)

	// Set with short TTL
	c.SetWithTTL("key1", "value1", 50*time.Millisecond)

	// Should be present immediately
	if _, ok := c.Get("key1"); !ok {
		t.Error("key should be present immediately")
	}

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Should be expired
	if _, ok := c.Get("key1"); ok {
		t.Error("key should be expired")
	}
}

func TestCache_Delete(t *testing.T) {
	c := New(time.Minute)

	c.Set("key1", "value1")
	c.Delete("key1")

	if _, ok := c.Get("key1"); ok {
		t.Error("key should be deleted")
	}
}

func TestCache_Clear(t *testing.T) {
	c := New(time.Minute)

	c.Set("key1", "value1")
	c.Set("key2", "value2")
	c.Clear()

	if c.Size() != 0 {
		t.Errorf("cache should be empty, has %d items", c.Size())
	}
}

func TestCache_GetOrSet(t *testing.T) {
	c := New(time.Minute)

	callCount := 0
	loader := func() (interface{}, error) {
		callCount++
		return "loaded-value", nil
	}

	// First call should load
	value, err := c.GetOrSet("key1", loader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if value != "loaded-value" {
		t.Errorf("expected 'loaded-value', got '%v'", value)
	}
	if callCount != 1 {
		t.Errorf("loader should be called once, called %d times", callCount)
	}

	// Second call should use cache
	value, err = c.GetOrSet("key1", loader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if value != "loaded-value" {
		t.Errorf("expected 'loaded-value', got '%v'", value)
	}
	if callCount != 1 {
		t.Errorf("loader should not be called again, called %d times", callCount)
	}
}

func TestCache_Concurrent(t *testing.T) {
	c := New(time.Minute)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			key := "key" + string(rune(n%10))
			c.Set(key, n)
			c.Get(key)
		}(i)
	}

	wg.Wait()
}
