// Package ratelimiter provides rate limiting functionality for the validator-dashboard.
package ratelimiter

import (
	"context"
	"net/http"
	"strconv"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RateLimitInfo contains rate limit information parsed from response headers.
type RateLimitInfo struct {
	Remaining int           // Requests remaining in current window
	Reset     time.Duration // Time until rate limit resets
	Window    time.Duration // Size of the rate limit window
}

// ParseRateLimitHeaders extracts rate limit info from HTTP response headers.
func ParseRateLimitHeaders(resp *http.Response) *RateLimitInfo {
	if resp == nil {
		return nil
	}

	info := &RateLimitInfo{}

	// Parse ratelimit-remaining (requests left in current window)
	if remaining := resp.Header.Get("ratelimit-remaining"); remaining != "" {
		if val, err := strconv.Atoi(remaining); err == nil {
			info.Remaining = val
		}
	}

	// Parse ratelimit-reset (seconds until reset)
	if reset := resp.Header.Get("ratelimit-reset"); reset != "" {
		if val, err := strconv.Atoi(reset); err == nil {
			info.Reset = time.Duration(val) * time.Second
		}
	}

	// Parse ratelimit-window (window size in seconds)
	if window := resp.Header.Get("ratelimit-window"); window != "" {
		if val, err := strconv.Atoi(window); err == nil {
			info.Window = time.Duration(val) * time.Second
		}
	}

	return info
}

// GlobalRateLimiter provides a shared rate limiter for outbound API calls.
// This ensures we respect the Beaconcha API rate limit of 1 request per second.
type GlobalRateLimiter struct {
	limiter     *rate.Limiter
	mu          sync.Mutex
	lastCall    time.Time
	nextAllowed time.Time // Adaptive delay based on rate limit headers
}

// NewGlobalRateLimiter creates a new rate limiter with the specified interval between requests.
func NewGlobalRateLimiter(interval time.Duration) *GlobalRateLimiter {
	// Calculate rate: 1 request per interval
	// rate.Limit is requests per second, so we convert
	rateLimit := rate.Every(interval)

	// Create limiter with burst=1 but immediately consume the initial token
	// This ensures the first request also waits, providing consistent spacing
	limiter := rate.NewLimiter(rateLimit, 1)
	limiter.Allow() // Consume the initial burst token

	return &GlobalRateLimiter{
		limiter:  limiter,
		lastCall: time.Now(),
	}
}

// Wait blocks until the rate limiter allows an event to happen.
// It returns an error if the context is canceled.
func (g *GlobalRateLimiter) Wait(ctx context.Context) error {
	err := g.limiter.Wait(ctx)
	if err == nil {
		g.mu.Lock()
		g.lastCall = time.Now()
		g.mu.Unlock()
	}
	return err
}

// Allow reports whether an event may happen now.
// Use this for non-blocking rate limit checks.
func (g *GlobalRateLimiter) Allow() bool {
	return g.limiter.Allow()
}

// Reserve returns a Reservation that indicates how long the caller must wait.
func (g *GlobalRateLimiter) Reserve() *rate.Reservation {
	return g.limiter.Reserve()
}

// Tokens returns the number of tokens available now.
func (g *GlobalRateLimiter) Tokens() float64 {
	return g.limiter.Tokens()
}

// UpdateFromHeaders adjusts the rate limiter based on rate limit headers.
// If remaining is 0, it waits for the reset duration before allowing the next request.
func (g *GlobalRateLimiter) UpdateFromHeaders(info *RateLimitInfo) {
	if info == nil {
		return
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	// If no remaining requests, we need to wait for reset
	if info.Remaining == 0 && info.Reset > 0 {
		// Record when we can make the next request
		g.nextAllowed = time.Now().Add(info.Reset)
	}
}

// WaitAdaptive waits respecting both the token bucket and any adaptive delay from headers.
func (g *GlobalRateLimiter) WaitAdaptive(ctx context.Context) error {
	g.mu.Lock()
	waitUntil := g.nextAllowed
	g.mu.Unlock()

	// If we have a header-based wait time, respect it
	if !waitUntil.IsZero() && time.Now().Before(waitUntil) {
		waitDuration := time.Until(waitUntil)
		select {
		case <-time.After(waitDuration):
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	// Also respect the token bucket rate limiter
	return g.Wait(ctx)
}
