package middleware

import (
	"net"
	"net/http"
	"sync"
	"time"
)

type clientInfo struct {
	lastSeen time.Time
	tokens   int
}

// IPRateLimit applies a simple token-bucket per-IP to limit abuse
func IPRateLimit(next http.Handler) http.Handler {
	var mu sync.Mutex
	clients := make(map[string]*clientInfo)

	// refill: 30 req/min => 0.5 req/sec, but allow burst of 10
	maxTokens := 30
	refillInterval := time.Minute

	// background janitor
	go func() {
		ticker := time.NewTicker(refillInterval)
		for range ticker.C {
			mu.Lock()
			for k, v := range clients {
				v.tokens = maxTokens
				if time.Since(v.lastSeen) > 10*time.Minute {
					delete(clients, k)
				}
			}
			mu.Unlock()
		}
	}()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			ip = r.RemoteAddr
		}

		mu.Lock()
		ci, ok := clients[ip]
		if !ok {
			ci = &clientInfo{lastSeen: time.Now(), tokens: maxTokens}
			clients[ip] = ci
		}
		if ci.tokens <= 0 {
			mu.Unlock()
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		ci.tokens--
		ci.lastSeen = time.Now()
		mu.Unlock()

		next.ServeHTTP(w, r)
	})
}
