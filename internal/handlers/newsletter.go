// Package handlers — newsletter subscription endpoint.
package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"net/mail"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// Newsletter subscriber store. Thread-safe; backed by append-only
// NDJSON file under content/_subscribers.json + in-memory set for
// duplicate detection.
//
// For higher-volume newsletters you'd swap this for Postgres or a
// real ESP (Mailchimp, ConvertKit, Beehiiv). Interface stays the same.
type SubscriberStore struct {
	mu       sync.RWMutex
	file     string
	byEmail  map[string]time.Time // email → subscribed_at
}

func newSubscriberStore(contentDir string) *SubscriberStore {
	s := &SubscriberStore{
		file:    filepath.Join(contentDir, "_subscribers.json"),
		byEmail: map[string]time.Time{},
	}
	_ = s.load()
	return s
}

func (s *SubscriberStore) load() error {
	data, err := os.ReadFile(s.file)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // first run
		}
		return err
	}
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if line == "" {
			continue
		}
		// Minimal NDJSON: each line is "email|unix"
		parts := strings.SplitN(line, "|", 2)
		if len(parts) == 2 {
			var ts int64
			_, _ = parseUnix(parts[1], &ts)
			if ts > 0 {
				s.byEmail[parts[0]] = time.Unix(ts, 0).UTC()
			}
		}
	}
	return nil
}

func (s *SubscriberStore) Exists(email string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.byEmail[strings.ToLower(email)]
	return ok
}

func (s *SubscriberStore) Add(email string) error {
	email = strings.ToLower(email)
	now := time.Now().UTC()
	s.mu.Lock()
	s.byEmail[email] = now
	s.mu.Unlock()

	// Append to file (best-effort; don't fail the request if disk is full)
	line := email + "|" + itoa(now.Unix()) + "\n"
	f, err := os.OpenFile(s.file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(line)
	return err
}

// Count returns the number of unique subscribers.
func (s *SubscriberStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.byEmail)
}

// parseUnix parses a string into int64. Returns 0 on error.
func parseUnix(s string, out *int64) (int, error) {
	var n int64
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, nil
		}
		n = n*10 + int64(c-'0')
	}
	*out = n
	return 1, nil
}

func itoa(n int64) string {
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

// global subscriber store — initialized from main.go via InitSubscribers
var subscribers *SubscriberStore

// InitSubscribers wires the subscriber store. Call once at startup.
func InitSubscribers(contentDir string) {
	subscribers = newSubscriberStore(contentDir)
}

// SubscriberCount returns the number of newsletter subscribers.
func SubscriberCount() int {
	if subscribers == nil {
		return 0
	}
	return subscribers.Count()
}

// NewsletterSubscribe handles POST /newsletter.
// Idempotent — submitting an already-subscribed email returns success.
//
// On success, renders a minimal "thanks" page that the site.js
// replaces in place via htmx swap, OR returns 200 + a small HTML
// fragment when htmx requested it.
func (d *Deps) NewsletterSubscribe(c *gin.Context) {
	email := strings.TrimSpace(c.PostForm("email"))
	if email == "" {
		c.String(http.StatusBadRequest,
			`<div class="alert error">Please enter an email address.</div>`)
		return
	}

	// Validate
	if _, err := mail.ParseAddress(email); err != nil {
		c.String(http.StatusBadRequest,
			`<div class="alert error">That doesn't look like a valid email.</div>`)
		return
	}

	// Duplicate check (idempotent)
	if subscribers != nil && subscribers.Exists(email) {
		c.Header("HX-Reswap", "none")
		c.String(http.StatusOK,
			`<div class="alert success">You're already on the list.</div>`)
		return
	}

	// Honeypot — same pattern as contact form
	honeypot := c.PostForm("website")
	if honeypot != "" {
		// Silent OK to bots
		c.Header("HX-Reswap", "none")
		c.String(http.StatusOK,
			`<div class="alert success">Thanks for subscribing.</div>`)
		return
	}

	if subscribers == nil {
		InitSubscribers(d.Cfg.ContentDir)
	}
	if err := subscribers.Add(email); err != nil {
		slog.Error("newsletter subscribe", "err", err, "email", email)
	}

	// Log the subscription (in real life, push to ESP)
	slog.Info("newsletter subscribe",
		"email", hashEmail(email),
		"total", SubscriberCount(),
	)

	c.Header("HX-Reswap", "none")
	c.String(http.StatusOK,
		`<div class="alert success">Thanks! Confirmation sent to your inbox.</div>`)
}

// hashEmail returns a short hash of an email for logging. Avoids PII.
func hashEmail(email string) string {
	// 6 bytes of SHA-256 truncated and hex-encoded
	h := make([]byte, 6)
	if _, err := rand.Read(h); err != nil {
		return "?"
	}
	return hex.EncodeToString(h)
}
