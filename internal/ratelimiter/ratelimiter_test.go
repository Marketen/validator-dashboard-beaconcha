package ratelimiter

import (
	"context"
	"net/http"
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

func TestParseRateLimitHeaders(t *testing.T) {
	tests := []struct {
		name     string
		headers  map[string]string
		expected *RateLimitInfo
	}{
		{
			name:    "all headers present",
			headers: map[string]string{"ratelimit-remaining": "5", "ratelimit-reset": "10", "ratelimit-window": "60"},
			expected: &RateLimitInfo{
				Remaining: 5,
				Reset:     10 * time.Second,
				Window:    60 * time.Second,
			},
		},
		{
			name:    "only remaining",
			headers: map[string]string{"ratelimit-remaining": "3"},
			expected: &RateLimitInfo{
				Remaining: 3,
				Reset:     0,
				Window:    0,
			},
		},
		{
			name:    "zero remaining",
			headers: map[string]string{"ratelimit-remaining": "0", "ratelimit-reset": "5"},
			expected: &RateLimitInfo{
				Remaining: 0,
				Reset:     5 * time.Second,
				Window:    0,
			},
		},
		{
			name:     "no headers",
			headers:  map[string]string{},
			expected: &RateLimitInfo{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &http.Response{Header: make(http.Header)}
			for k, v := range tt.headers {
				resp.Header.Set(k, v)
			}

			info := ParseRateLimitHeaders(resp)

			if info.Remaining != tt.expected.Remaining {
				t.Errorf("Remaining: got %d, want %d", info.Remaining, tt.expected.Remaining)
			}
			if info.Reset != tt.expected.Reset {
				t.Errorf("Reset: got %v, want %v", info.Reset, tt.expected.Reset)
			}
			if info.Window != tt.expected.Window {
				t.Errorf("Window: got %v, want %v", info.Window, tt.expected.Window)
			}
		})
	}
}

func TestParseRateLimitHeaders_NilResponse(t *testing.T) {
	info := ParseRateLimitHeaders(nil)
	if info != nil {
		t.Errorf("expected nil for nil response, got %v", info)
	}
}

func TestGlobalRateLimiter_UpdateFromHeaders(t *testing.T) {
	limiter := NewGlobalRateLimiter(100 * time.Millisecond)

	// Update with zero remaining - should set nextAllowed
	info := &RateLimitInfo{
		Remaining: 0,
		Reset:     200 * time.Millisecond,
	}
	limiter.UpdateFromHeaders(info)

	// WaitAdaptive should now wait for the reset duration
	ctx := context.Background()
	start := time.Now()
	if err := limiter.WaitAdaptive(ctx); err != nil {
		t.Fatalf("WaitAdaptive failed: %v", err)
	}
	elapsed := time.Since(start)

	// Should have waited at least 150ms (with some tolerance)
	if elapsed < 150*time.Millisecond {
		t.Errorf("should have waited for reset, elapsed: %v", elapsed)
	}
}

func TestGlobalRateLimiter_UpdateFromHeaders_Nil(t *testing.T) {
	limiter := NewGlobalRateLimiter(50 * time.Millisecond)

	// Should not panic with nil
	limiter.UpdateFromHeaders(nil)

	// Should still work normally
	ctx := context.Background()
	if err := limiter.WaitAdaptive(ctx); err != nil {
		t.Fatalf("WaitAdaptive failed: %v", err)
	}
}
