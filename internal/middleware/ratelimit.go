// Package middleware — fixed-window rate limiter using a token bucket.
//
// Default: 60 requests / 60s per IP. Configurable via rate-limit headers.
//
// Trade-offs vs. Redis-based limiters:
//   + Single binary, no extra dep
//   + Sub-microsecond cost per request (map lookup + counter bump)
//   - Per-instance state — multiple gateway replicas each count separately
//   - Memory grows with unique IPs per window — GC every window
//
// For a single-binary deploy (us) this is the right choice. For
// multi-replica, swap in Redis or use ulule/limiter's Redis store.
package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// IPRateLimiter is a thread-safe token bucket per client IP.
type IPRateLimiter struct {
	mu       sync.Mutex
	limit    int           // max requests per window
	window   time.Duration // window length
	clients  map[string]*clientBucket
}

type clientBucket struct {
	count   int
	resetAt time.Time
}

// NewIPRateLimiter creates a rate limiter allowing `limit` requests
// per `window` per IP. A reasonable default is 60 / minute.
func NewIPRateLimiter(limit int, window time.Duration) *IPRateLimiter {
	rl := &IPRateLimiter{
		limit:   limit,
		window:  window,
		clients: map[string]*clientBucket{},
	}
	go rl.gcLoop()
	return rl
}

func (rl *IPRateLimiter) gcLoop() {
	ticker := time.NewTicker(rl.window)
	defer ticker.Stop()
	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for ip, b := range rl.clients {
			if now.After(b.resetAt) {
				delete(rl.clients, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// Allow returns true if the request should proceed, false if it
// should be rejected with 429.
func (rl *IPRateLimiter) Allow(ip string) (allowed bool, retryAfter time.Duration) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	b, ok := rl.clients[ip]
	if !ok || now.After(b.resetAt) {
		rl.clients[ip] = &clientBucket{count: 1, resetAt: now.Add(rl.window)}
		return true, 0
	}

	if b.count >= rl.limit {
		return false, b.resetAt.Sub(now)
	}

	b.count++
	return true, 0
}

// RateLimit is the gin middleware that enforces per-IP rate limits.
// Rejected requests get 429 + Retry-After + JSON error (or HTML).
func (rl *IPRateLimiter) RateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip rate limit for health check, static files
		path := c.Request.URL.Path
		if path == "/healthz" || path == "/robots.txt" || path == "/sitemap.xml" || path == "/rss.xml" {
			c.Next()
			return
		}
		// Skip static assets
		if len(path) >= 8 && path[:8] == "/static/" {
			c.Next()
			return
		}

		ip := c.ClientIP()
		ok, retry := rl.Allow(ip)
		if !ok {
			secs := int(retry.Seconds())
			if secs < 1 {
				secs = 1
			}
			c.Header("Retry-After", strconvI(secs))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":       "rate limit exceeded",
				"retry_after": secs,
			})
			return
		}
		c.Next()
	}
}

// strconvI is a tiny int-to-string to avoid importing strconv here.
func strconvI(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
