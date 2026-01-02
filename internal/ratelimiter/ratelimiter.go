// Package ratelimiter provides rate limiting functionality for the validator-dashboard.
package ratelimiter

import (
	"context"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// GlobalRateLimiter provides a shared rate limiter for outbound API calls.
// This ensures we respect the Beaconcha API rate limit of 1 request per second.
type GlobalRateLimiter struct {
	limiter  *rate.Limiter
	mu       sync.Mutex
	lastCall time.Time
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

// IPRateLimiter provides per-IP rate limiting for incoming requests.
// This prevents abuse from any single client.
type IPRateLimiter struct {
	limiters map[string]*ipLimiterEntry
	mu       sync.RWMutex
	rate     rate.Limit
	burst    int
	cleanup  time.Duration
}

type ipLimiterEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// NewIPRateLimiter creates a new per-IP rate limiter.
// requests: number of requests allowed
// window: time window for the requests
func NewIPRateLimiter(requests int, window time.Duration) *IPRateLimiter {
	ipl := &IPRateLimiter{
		limiters: make(map[string]*ipLimiterEntry),
		rate:     rate.Limit(float64(requests) / window.Seconds()),
		burst:    requests, // Allow burst up to the full limit
		cleanup:  window * 2,
	}

	// Start cleanup goroutine to remove stale entries
	go ipl.cleanupLoop()

	return ipl
}

// GetLimiter returns the rate limiter for a given IP address.
func (ipl *IPRateLimiter) GetLimiter(ip string) *rate.Limiter {
	ipl.mu.Lock()
	defer ipl.mu.Unlock()

	entry, exists := ipl.limiters[ip]
	if !exists {
		limiter := rate.NewLimiter(ipl.rate, ipl.burst)
		ipl.limiters[ip] = &ipLimiterEntry{
			limiter:  limiter,
			lastSeen: time.Now(),
		}
		return limiter
	}

	entry.lastSeen = time.Now()
	return entry.limiter
}

// Allow checks if a request from the given IP should be allowed.
func (ipl *IPRateLimiter) Allow(ip string) bool {
	return ipl.GetLimiter(ip).Allow()
}

// cleanupLoop periodically removes stale IP entries to prevent memory leaks.
func (ipl *IPRateLimiter) cleanupLoop() {
	ticker := time.NewTicker(ipl.cleanup)
	defer ticker.Stop()

	for range ticker.C {
		ipl.mu.Lock()
		for ip, entry := range ipl.limiters {
			if time.Since(entry.lastSeen) > ipl.cleanup {
				delete(ipl.limiters, ip)
			}
		}
		ipl.mu.Unlock()
	}
}
