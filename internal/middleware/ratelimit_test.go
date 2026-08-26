package middleware

import (
	"testing"
	"time"
)

func TestIPRateLimiterAllow(t *testing.T) {
	rl := NewIPRateLimiter(3, 100*time.Millisecond)

	// First 3 requests from same IP should pass
	for i := 1; i <= 3; i++ {
		ok, retry := rl.Allow("1.2.3.4")
		if !ok {
			t.Errorf("request %d: got !ok, want ok", i)
		}
		if retry != 0 {
			t.Errorf("request %d: retry=%v, want 0", i, retry)
		}
	}

	// 4th request from same IP should fail
	ok, retry := rl.Allow("1.2.3.4")
	if ok {
		t.Error("4th request: got ok, want !ok")
	}
	if retry <= 0 {
		t.Errorf("4th request: retry=%v, want > 0", retry)
	}
	if retry > 100*time.Millisecond {
		t.Errorf("4th request: retry=%v, want <= 100ms", retry)
	}

	// Different IP should be unaffected
	ok, _ = rl.Allow("5.6.7.8")
	if !ok {
		t.Error("different IP: got !ok, want ok")
	}
}

func TestIPRateLimiterWindowReset(t *testing.T) {
	rl := NewIPRateLimiter(2, 50*time.Millisecond)

	// Burn 2 requests
	rl.Allow("1.1.1.1")
	rl.Allow("1.1.1.1")
	if ok, _ := rl.Allow("1.1.1.1"); ok {
		t.Error("3rd request in window: got ok, want !ok")
	}

	// Wait for window to expire
	time.Sleep(60 * time.Millisecond)

	// Should now allow again
	ok, _ := rl.Allow("1.1.1.1")
	if !ok {
		t.Error("after window: got !ok, want ok")
	}
}

func TestIPRateLimiterGC(t *testing.T) {
	rl := NewIPRateLimiter(2, 30*time.Millisecond)

	rl.Allow("1.1.1.1")
	rl.Allow("2.2.2.2")

	if got := len(rl.clients); got != 2 {
		t.Errorf("clients: got %d want 2", got)
	}

	// Wait for window + GC tick (window is 30ms; GC runs every 30ms; wait 100ms)
	time.Sleep(100 * time.Millisecond)

	rl.mu.Lock()
	n := len(rl.clients)
	rl.mu.Unlock()
	if n != 0 {
		t.Errorf("clients after GC: got %d want 0", n)
	}
}

func TestIPRateLimiterIndependent(t *testing.T) {
	rl := NewIPRateLimiter(1, 1*time.Second)

	// Each IP gets its own quota
	for _, ip := range []string{"a", "b", "c"} {
		ok, _ := rl.Allow(ip)
		if !ok {
			t.Errorf("IP %s: first request should pass", ip)
		}
		// Second request from same IP fails
		ok, _ = rl.Allow(ip)
		if ok {
			t.Errorf("IP %s: second request should fail", ip)
		}
	}
}
