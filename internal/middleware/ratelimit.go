package middleware

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// RateLimit returns a middleware that limits each IP to rps requests per second
// using a simple token bucket implemented with standard library only.
// maxBurst controls the initial burst capacity.
func RateLimit(rps int, maxBurst int) func(http.Handler) http.Handler {
	rl := &rateLimiter{
		rps:      rps,
		maxBurst: maxBurst,
		clients:  make(map[string]*bucket),
	}
	// Prune idle clients every minute.
	go rl.cleanup()
	return rl.middleware
}

type bucket struct {
	tokens   float64
	lastSeen time.Time
	mu       sync.Mutex
}

type rateLimiter struct {
	rps      int
	maxBurst int
	mu       sync.Mutex
	clients  map[string]*bucket
}

func (rl *rateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	b, ok := rl.clients[ip]
	if !ok {
		b = &bucket{tokens: float64(rl.maxBurst), lastSeen: time.Now()}
		rl.clients[ip] = b
	}
	rl.mu.Unlock()

	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(b.lastSeen).Seconds()
	b.lastSeen = now

	// Refill tokens.
	b.tokens += elapsed * float64(rl.rps)
	if b.tokens > float64(rl.maxBurst) {
		b.tokens = float64(rl.maxBurst)
	}

	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

func (rl *rateLimiter) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := realIP(r)
		if !rl.allow(ip) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(map[string]string{"error": "rate limit exceeded"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (rl *rateLimiter) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		rl.mu.Lock()
		for ip, b := range rl.clients {
			b.mu.Lock()
			idle := time.Since(b.lastSeen)
			b.mu.Unlock()
			if idle > 5*time.Minute {
				delete(rl.clients, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// realIP extracts the client IP, respecting common proxy headers.
func realIP(r *http.Request) string {
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		return ip
	}
	return r.RemoteAddr
}
