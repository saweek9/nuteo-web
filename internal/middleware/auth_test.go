package middleware

import (
	"testing"
	"time"
)

func TestSignAndVerifySession(t *testing.T) {
	secret := []byte("super-secret-key-for-tests")
	user := "admin@example.com"
	expiry := time.Now().Add(1 * time.Hour)

	signed := SignSession(secret, user, expiry)

	// Valid signature
	session := VerifySession(secret, signed)
	if session == nil {
		t.Fatal("VerifySession: got nil for valid signed session")
	}
	if session.User != user {
		t.Errorf("User: got %q want %q", session.User, user)
	}
	if !session.ExpiresAt.Equal(expiry.Truncate(time.Second)) {
		t.Errorf("ExpiresAt: got %v want %v", session.ExpiresAt, expiry)
	}
}

func TestVerifySessionTampered(t *testing.T) {
	secret := []byte("secret")
	wrong := []byte("wrong-secret")
	expiry := time.Now().Add(1 * time.Hour)

	signed := SignSession(secret, "admin", expiry)

	cases := []struct {
		name string
		cookie string
		secret []byte
	}{
		{"wrong secret", signed, wrong},
		{"modified user", "evil" + signed[5:], secret},  // 5 chars = "admin"
		{"modified signature", signed[:len(signed)-4] + "0000", secret},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if VerifySession(tc.secret, tc.cookie) != nil {
				t.Errorf("VerifySession(%q, %q): expected nil", tc.name, tc.cookie)
			}
		})
	}
}

func TestVerifySessionExpired(t *testing.T) {
	secret := []byte("secret")
	expiry := time.Now().Add(-1 * time.Hour) // already expired

	signed := SignSession(secret, "admin", expiry)
	if VerifySession(secret, signed) != nil {
		t.Error("VerifySession: expected nil for expired session")
	}
}

func TestVerifySessionMalformed(t *testing.T) {
	secret := []byte("secret")

	cases := []string{
		"",                       // empty
		"just-one-part",          // no colons
		"a:b",                    // no signature
		"a:b:NOTHEX",             // signature not hex
		"a:notanumber:sig",       // expiry not int
	}

	for _, c := range cases {
		t.Run(c, func(t *testing.T) {
			if VerifySession(secret, c) != nil {
				t.Errorf("VerifySession(%q): expected nil", c)
			}
		})
	}
}

func TestSessionSignatureIsDeterministic(t *testing.T) {
	// Same inputs → same signature (HMAC is deterministic)
	secret := []byte("k")
	a := SignSession(secret, "u", time.Unix(1234567890, 0))
	b := SignSession(secret, "u", time.Unix(1234567890, 0))
	if a != b {
		t.Errorf("SignSession not deterministic: %q != %q", a, b)
	}

	// Different user → different sig
	c := SignSession(secret, "u2", time.Unix(1234567890, 0))
	if a == c {
		t.Error("SignSession: same sig for different users")
	}
}

func TestSplitN(t *testing.T) {
	// splitN returns up to n parts; if n exceeds the number of separators,
	// the remainder is kept as one part in the final element.
	cases := []struct {
		in   string
		n    int
		want []string
	}{
		{"a:b:c", 3, []string{"a", "b", "c"}},
		{"a:b:c", 2, []string{"a", "b:c"}},
		{"a:b:c", 4, []string{"a", "b", "c"}}, // n=4 > parts, returns 3
		{"", 3, []string{""}},
		{":", 2, []string{"", ""}},
		{"a", 1, []string{"a"}},
	}
	for _, tc := range cases {
		got := splitN(tc.in, ':', tc.n)
		if !equalStrings(got, tc.want) {
			t.Errorf("splitN(%q, %d): got %v want %v", tc.in, tc.n, got, tc.want)
		}
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
