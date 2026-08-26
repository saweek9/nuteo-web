package config

import (
	"os"
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	// Clear env vars that might interfere with defaults test
	for _, k := range []string{"ENV", "ADDR", "READ_TIMEOUT", "WRITE_TIMEOUT",
		"CONTENT_DIR", "STATIC_DIR", "CONTACT_EMAIL_TO", "CONTACT_EMAIL_FROM",
		"CSRF_SECRET", "SITE_NAME", "SITE_URL"} {
		_ = os.Unsetenv(k)
	}

	cases := []struct {
		name    string
		env     map[string]string
		want    *Config
		wantErr bool
	}{
		{
			name: "all defaults",
			env: map[string]string{
				"CONTACT_EMAIL_TO":   "hi@x.com",
				"CONTACT_EMAIL_FROM": "noreply@x.com",
				"CSRF_SECRET":         "x",
			},
			want: &Config{
				Env:               "development",
				Addr:              ":8080",
				ReadTimeout:       10 * time.Second,
				WriteTimeout:      10 * time.Second,
				ContentDir:        "./content",
				StaticDir:         "./web/static",
				ContactEmailTo:    "hi@x.com",
				ContactEmailFrom:  "noreply@x.com",
				CSRFSecret:        "x",
				SMTPPort:          587,
				SiteName:          "nuteo solution",
				SiteURL:           "https://nuteo.example.com",
				LogoPath:          "/static/images/logo.svg",
			},
		},
		{
			name: "production mode",
			env: map[string]string{
				"ENV":                 "production",
				"ADDR":                ":9000",
				"CONTACT_EMAIL_TO":    "ops@example.com",
				"CONTACT_EMAIL_FROM":  "noreply@example.com",
				"CSRF_SECRET":         "long-random-secret",
				"SMTP_HOST":           "smtp.example.com",
				"SMTP_PORT":           "2525",
				"SMTP_USER":           "user",
				"SMTP_PASS":           "pass",
				"SITE_NAME":           "Custom Co",
				"SITE_URL":            "https://custom.example.com",
				"ADMIN_USER":          "admin",
				"ADMIN_PASSWORD":      "secret",
				"RATE_LIMIT_RPM":      "10",
			},
			want: &Config{
				Env:               "production",
				Addr:              ":9000",
				ReadTimeout:       10 * time.Second,
				WriteTimeout:      10 * time.Second,
				ContentDir:        "./content",
				StaticDir:         "./web/static",
				ContactEmailTo:    "ops@example.com",
				ContactEmailFrom:  "noreply@example.com",
				CSRFSecret:        "long-random-secret",
				SMTPHost:          "smtp.example.com",
				SMTPPort:          2525,
				SMTPUser:          "user",
				SMTPPass:          "pass",
				SiteName:          "Custom Co",
				SiteURL:           "https://custom.example.com",
				LogoPath:          "/static/images/logo.svg",
				AdminUser:         "admin",
				AdminPassword:     "secret",
				RateLimitRPM:      10,
			},
		},
		{
			name: "missing required — CONTACT_EMAIL_TO",
			env: map[string]string{
				"CONTACT_EMAIL_FROM": "noreply@x.com",
				"CSRF_SECRET":        "x",
			},
			wantErr: true,
		},
		{
			name: "missing required — CONTACT_EMAIL_FROM",
			env: map[string]string{
				"CONTACT_EMAIL_TO": "hi@x.com",
				"CSRF_SECRET":      "x",
			},
			wantErr: true,
		},
		{
			name: "missing required — CSRF_SECRET",
			env: map[string]string{
				"CONTACT_EMAIL_TO":   "hi@x.com",
				"CONTACT_EMAIL_FROM": "noreply@x.com",
			},
			wantErr: true,
		},
		{
			name: "production without SMTP fails",
			env: map[string]string{
				"ENV":                "production",
				"CONTACT_EMAIL_TO":   "hi@x.com",
				"CONTACT_EMAIL_FROM": "noreply@x.com",
				"CSRF_SECRET":        "x",
			},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Reset
			for _, k := range []string{"ENV", "ADDR", "READ_TIMEOUT", "WRITE_TIMEOUT",
				"CONTENT_DIR", "STATIC_DIR", "CONTACT_EMAIL_TO", "CONTACT_EMAIL_FROM",
				"CSRF_SECRET", "SITE_NAME", "SITE_URL", "SMTP_HOST", "SMTP_PORT",
				"SMTP_USER", "SMTP_PASS", "ADMIN_USER", "ADMIN_PASSWORD", "RATE_LIMIT_RPM"} {
				_ = os.Unsetenv(k)
			}
			// Set
			for k, v := range tc.env {
				_ = os.Setenv(k, v)
			}

			got, err := Load()
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil (config=%+v)", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Env != tc.want.Env {
				t.Errorf("Env: got %q want %q", got.Env, tc.want.Env)
			}
			if got.Addr != tc.want.Addr {
				t.Errorf("Addr: got %q want %q", got.Addr, tc.want.Addr)
			}
			if got.SMTPHost != tc.want.SMTPHost {
				t.Errorf("SMTPHost: got %q want %q", got.SMTPHost, tc.want.SMTPHost)
			}
			if got.SMTPPort != tc.want.SMTPPort {
				t.Errorf("SMTPPort: got %d want %d", got.SMTPPort, tc.want.SMTPPort)
			}
			if got.CSRFSecret != tc.want.CSRFSecret {
				t.Errorf("CSRFSecret: got %q want %q", got.CSRFSecret, tc.want.CSRFSecret)
			}
			if got.SiteURL != tc.want.SiteURL {
				t.Errorf("SiteURL: got %q want %q", got.SiteURL, tc.want.SiteURL)
			}
			if got.AdminUser != tc.want.AdminUser {
				t.Errorf("AdminUser: got %q want %q", got.AdminUser, tc.want.AdminUser)
			}
		})
	}
}

func TestIsProd(t *testing.T) {
	c := &Config{Env: "production"}
	if !c.IsProd() {
		t.Error("IsProd() should be true for Env=production")
	}

	c = &Config{Env: "development"}
	if c.IsProd() {
		t.Error("IsProd() should be false for Env=development")
	}

	c = &Config{Env: ""}
	if c.IsProd() {
		t.Error("IsProd() should be false for Env=''")
	}
}
