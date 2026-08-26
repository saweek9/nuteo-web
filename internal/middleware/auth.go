// Package middleware — admin auth.
//
// Auth model: HTTP-only signed cookie carrying admin user ID +
// expiry. Login sets cookie; logout deletes. Handlers check via
// RequireAdmin() middleware.
//
// For production you'd add:
//   - bcrypt password hash
//   - constant-time token comparison
//   - rate limiting on /admin/login
//   - session table in DB
package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// AdminSession is the signed cookie payload.
type AdminSession struct {
	User      string
	ExpiresAt time.Time
}

const (
	adminCookieName = "admin_session"
	adminRole       = "admin"
	sessionDuration = 8 * time.Hour
)

// SignSession produces a hex-encoded HMAC-SHA256 signature.
//
// Cookie value format: "<user_id>:<unix_expiry>:<hex_signature>"
// where signature is HMAC(secret, user_id + ":" + unix_expiry).
func SignSession(secret []byte, user string, expires time.Time) string {
	exp := strconv.FormatInt(expires.Unix(), 10)
	payload := user + ":" + exp
	sig := hmacSign(secret, payload)
	return payload + ":" + sig
}

// VerifySession parses and validates the cookie value.
// Returns the session if valid; nil if tampered, malformed, or expired.
func VerifySession(secret []byte, cookie string) *AdminSession {
	// user:exp:sig
	parts := splitN(cookie, ':', 3)
	if len(parts) != 3 {
		return nil
	}
	payload := parts[0] + ":" + parts[1]
	expected := hmacSign(secret, payload)
	if !hmac.Equal([]byte(expected), []byte(parts[2])) {
		return nil
	}
	exp, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return nil
	}
	if time.Now().Unix() > exp {
		return nil
	}
	return &AdminSession{User: parts[0], ExpiresAt: time.Unix(exp, 0)}
}

// hmacSign returns the hex-encoded HMAC-SHA256 of payload.
func hmacSign(secret []byte, payload string) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}

func splitN(s string, sep byte, n int) []string {
	out := []string{}
	start := 0
	count := 0
	for i := 0; i < len(s); i++ {
		if s[i] == sep {
			out = append(out, s[start:i])
			start = i + 1
			count++
			if count == n-1 {
				break
			}
		}
	}
	out = append(out, s[start:])
	return out
}

// RequireAdmin blocks requests without a valid admin session.
// On failure: redirect to /admin/login (HTML requests) or 401 (API).
func RequireAdmin(secret []byte) gin.HandlerFunc {
	return func(c *gin.Context) {
		cookie, err := c.Cookie(adminCookieName)
		if err != nil {
			redirectOr401(c, "/admin/login")
			return
		}
		session := VerifySession(secret, cookie)
		if session == nil {
			redirectOr401(c, "/admin/login")
			return
		}
		c.Set("admin_user", session.User)
		c.Next()
	}
}

func redirectOr401(c *gin.Context, to string) {
	if wantsJSON(c) {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	c.Redirect(http.StatusFound, to)
	c.Abort()
}

func wantsJSON(c *gin.Context) bool {
	if c.GetHeader("Accept") == "application/json" {
		return true
	}
	if c.GetHeader("X-Requested-With") == "XMLHttpRequest" {
		return true
	}
	return false
}

// AdminUser returns the current admin username, or empty string.
func AdminUser(c *gin.Context) string {
	if v, ok := c.Get("admin_user"); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
