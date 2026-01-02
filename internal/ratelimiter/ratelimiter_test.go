package ratelimiter

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestGlobalRateLimiter_Wait(t *testing.T) {
	limiter := NewGlobalRateLimiter(100 * time.Millisecond)

	ctx := context.Background()

	// First request should wait (burst token is consumed at creation)
	start := time.Now()
	if err := limiter.Wait(ctx); err != nil {
		t.Fatalf("first wait failed: %v", err)
	}
	elapsed := time.Since(start)
	if elapsed < 50*time.Millisecond {
		t.Errorf("first request should have waited: %v", elapsed)
	}

	// Second request should also wait
	start = time.Now()
	if err := limiter.Wait(ctx); err != nil {
		t.Fatalf("second wait failed: %v", err)
	}
	elapsed = time.Since(start)
	if elapsed < 50*time.Millisecond {
		t.Errorf("second request should have waited: %v", elapsed)
	}
}

func TestGlobalRateLimiter_ContextCancellation(t *testing.T) {
	limiter := NewGlobalRateLimiter(time.Second)

	ctx := context.Background()
	_ = limiter.Wait(ctx)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := limiter.Wait(ctx)
	if err == nil {
		t.Error("expected error due to context cancellation")
	}
}

func TestIPRateLimiter_Allow(t *testing.T) {
	limiter := NewIPRateLimiter(5, time.Second)

	ip := "192.168.1.1"

	for i := 0; i < 5; i++ {
		if !limiter.Allow(ip) {
			t.Errorf("request %d should be allowed", i+1)
		}
	}

	if limiter.Allow(ip) {
		t.Error("6th request should be denied")
	}
}

func TestIPRateLimiter_DifferentIPs(t *testing.T) {
	limiter := NewIPRateLimiter(2, time.Second)

	ip1 := "192.168.1.1"
	ip2 := "192.168.1.2"

	if !limiter.Allow(ip1) {
		t.Error("ip1 request 1 should be allowed")
	}
	if !limiter.Allow(ip1) {
		t.Error("ip1 request 2 should be allowed")
	}
	if limiter.Allow(ip1) {
		t.Error("ip1 request 3 should be denied")
	}

	if !limiter.Allow(ip2) {
		t.Error("ip2 request 1 should be allowed")
	}
	if !limiter.Allow(ip2) {
		t.Error("ip2 request 2 should be allowed")
	}
}

func TestIPRateLimiter_Concurrent(t *testing.T) {
	limiter := NewIPRateLimiter(100, time.Second)

	var wg sync.WaitGroup
	allowed := make(chan bool, 200)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				allowed <- limiter.Allow("test-ip")
			}
		}()
	}

	wg.Wait()
	close(allowed)

	count := 0
	for a := range allowed {
		if a {
			count++
		}
	}

	if count != 100 {
		t.Errorf("expected 100 allowed requests, got %d", count)
	}
}
