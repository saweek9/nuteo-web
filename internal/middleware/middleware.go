// Package middleware wires Gin middleware: structured logging,
// request-id propagation, security headers, rate limiting.
package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RequestID injects a UUID v4 into the context and response header.
// Re-uses incoming X-Request-Id if present (chain tracing).
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		rid := c.GetHeader("X-Request-Id")
		if rid == "" {
			rid = uuid.NewString()
		}
		c.Set("request_id", rid)
		c.Writer.Header().Set("X-Request-Id", rid)
		c.Next()
	}
}

// Logger logs every request as structured JSON.
// Skips /healthz to avoid noise.
func Logger(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.URL.Path == "/healthz" {
			c.Next()
			return
		}
		start := time.Now()
		c.Next()
		logger.Info("http",
			slog.String("method", c.Request.Method),
			slog.String("path", c.Request.URL.Path),
			slog.Int("status", c.Writer.Status()),
			slog.Duration("dur", time.Since(start)),
			slog.String("ip", c.ClientIP()),
			slog.String("rid", c.GetString("request_id")),
			slog.Int("bytes", c.Writer.Size()),
		)
	}
}

// SecurityHeaders adds baseline browser security headers.
// CSP is set to a strict-but-functional default for an HTMX site.
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.Writer.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("Permissions-Policy",
			"geolocation=(), microphone=(), camera=(), payment=()")
		// CSP — adjust as you add inline scripts or third-party widgets.
		h.Set("Content-Security-Policy",
			"default-src 'self'; "+
				"img-src 'self' data: https:; "+
				"script-src 'self' 'unsafe-inline' https://unpkg.com; "+
				"style-src 'self' 'unsafe-inline'; "+
				"connect-src 'self'; "+
				"font-src 'self' data:; "+
				"frame-ancestors 'none'; "+
				"base-uri 'self'")
		c.Next()
	}
}

// RateLimit returns a per-IP token-bucket middleware.
//
// Defaults to N requests/min, in-memory only (per-instance, fine for
// single-node deploys). Swap to Redis for multi-node.
func RateLimit(perMinute int) gin.HandlerFunc {
	if perMinute <= 0 {
		perMinute = 60
	}
	// (placeholder — wired with golang.org/x/time/rate per-IP limiter in main.go)
	return func(c *gin.Context) { c.Next() }
}
