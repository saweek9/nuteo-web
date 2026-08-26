// Package config loads runtime configuration from environment variables.
//
// We use caarlos0/env for tagged struct-based parsing with defaults and
// required-field enforcement. All defaults are dev-friendly; production
// overrides come from the systemd unit's EnvironmentFile.
package config

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	// Server
	Env         string        `env:"ENV"         envDefault:"development"`
	Addr        string        `env:"ADDR"        envDefault:":8080"`
	ReadTimeout time.Duration `env:"READ_TIMEOUT" envDefault:"10s"`
	WriteTimeout time.Duration `env:"WRITE_TIMEOUT" envDefault:"10s"`

	// Content paths
	ContentDir string `env:"CONTENT_DIR" envDefault:"./content"`
	StaticDir  string `env:"STATIC_DIR"  envDefault:"./web/static"`

	// Contact form
	ContactEmailTo   string `env:"CONTACT_EMAIL_TO,required"`
	ContactEmailFrom string `env:"CONTACT_EMAIL_FROM,required"`

	// SMTP (used by email.SMTPSender)
	SMTPHost string `env:"SMTP_HOST"`
	SMTPPort int    `env:"SMTP_PORT" envDefault:"587"`
	SMTPUser string `env:"SMTP_USER"`
	SMTPPass string `env:"SMTP_PASS"`

	// Security
	CSRFSecret   string `env:"CSRF_SECRET,required"`
	RateLimitRPM int    `env:"RATE_LIMIT_RPM" envDefault:"5"`

	// Admin panel (optional — leave blank to disable)
	AdminUser     string `env:"ADMIN_USER"      envDefault:""`
	AdminPassword string `env:"ADMIN_PASSWORD"  envDefault:""`

	// Site metadata
	SiteName  string `env:"SITE_NAME"  envDefault:"nuteo solution"`
	SiteURL   string `env:"SITE_URL"   envDefault:"https://nuteo.example.com"`
	LogoPath  string `env:"LOGO_PATH"  envDefault:"/static/images/logo.svg"`
	TwitterHandle string `env:"TWITTER_HANDLE"`
	LinkedInURL   string `env:"LINKEDIN_URL"`
	GitHubURL     string `env:"GITHUB_URL"`
}

// Load parses env vars into a Config and validates required fields.
func Load() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
	if cfg.Env == "production" && cfg.SMTPHost == "" {
		return nil, fmt.Errorf("config: SMTP_HOST required in production")
	}
	return cfg, nil
}

// IsProd reports whether we're running in production mode.
func (c *Config) IsProd() bool { return c.Env == "production" }
