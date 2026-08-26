// Package models holds the domain types used across the site.
//
// Content files in /content/<kind>/<slug>.md are loaded into these types
// at startup by internal/storage. Templates read directly from these.
package models

import "time"

// Service is a single offering shown on /services.
type Service struct {
	Title       string   `yaml:"title"        json:"title"`
	Slug        string   `yaml:"slug"         json:"slug"`
	Summary     string   `yaml:"summary"      json:"summary"`
	Icon        string   `yaml:"icon"         json:"icon"`
	Order       int      `yaml:"order"        json:"order"`
	Audience    []string `yaml:"audience"     json:"audience"`
	Tags        []string `yaml:"tags"         json:"tags"`
	Featured    bool     `yaml:"featured"     json:"featured"`
	CTALabel    string   `yaml:"cta_label"    json:"cta_label"`
	CTAHref     string   `yaml:"cta_href"     json:"cta_href"`
	PublishedAt time.Time `yaml:"published_at" json:"published_at"`
	UpdatedAt   time.Time `yaml:"updated_at"   json:"updated_at"`

	// Body is the rendered HTML produced by goldmark.
	BodyHTML string `yaml:"-" json:"body_html"`
	// BodyMD is the raw markdown source (for editing/preview).
	BodyMD string `yaml:"-" json:"-"`
}

// Portfolio is a single project on /portfolio.
type Portfolio struct {
	Title       string   `yaml:"title"        json:"title"`
	Slug        string   `yaml:"slug"         json:"slug"`
	Summary     string   `yaml:"summary"      json:"summary"`
	Client      string   `yaml:"client"       json:"client"`
	Industry    string   `yaml:"industry"     json:"industry"`
	Year        int      `yaml:"year"         json:"year"`
	Stack       []string `yaml:"stack"        json:"stack"`
	Tags        []string `yaml:"tags"         json:"tags"`
	Image       string   `yaml:"image"        json:"image"`
	URL         string   `yaml:"url"          json:"url"`
	Featured    bool     `yaml:"featured"     json:"featured"`
	PublishedAt time.Time `yaml:"published_at" json:"published_at"`

	BodyHTML string `yaml:"-" json:"body_html"`
	BodyMD   string `yaml:"-" json:"-"`
}

// Post is a single blog/article entry (optional).
type Post struct {
	Title       string    `yaml:"title"        json:"title"`
	Slug        string    `yaml:"slug"         json:"slug"`
	Summary     string    `yaml:"summary"      json:"summary"`
	Author      string    `yaml:"author"       json:"author"`
	Tags        []string  `yaml:"tags"         json:"tags"`
	PublishedAt time.Time `yaml:"published_at" json:"published_at"`
	UpdatedAt   time.Time `yaml:"updated_at"   json:"updated_at"`

	BodyHTML string `yaml:"-" json:"body_html"`
	BodyMD   string `yaml:"-" json:"-"`
}

// Inquiry is a contact form submission.
type Inquiry struct {
	ID        string    `json:"id"`
	ReceivedAt time.Time `json:"received_at"`
	Name      string    `json:"name"      validate:"required,min=2,max=100"`
	Email     string    `json:"email"     validate:"required,email"`
	Company   string    `json:"company"   validate:"max=200"`
	Phone     string    `json:"phone"     validate:"omitempty,max=50"`
	Topic     string    `json:"topic"     validate:"required,max=80"`
	Message   string    `json:"message"   validate:"required,min=10,max=5000"`
	// Honeypot — must be empty. Bots fill it.
	Website   string    `json:"website"    validate:"omitempty,len=0"`
	// Source metadata
	IP        string    `json:"ip"`
	UserAgent string    `json:"user_agent"`
	Referrer  string    `json:"referrer"`
	// Spam score (0.0–1.0). Above 0.8 → silently dropped.
	SpamScore float64   `json:"spam_score"`

	// Consent — set by handler from "consent" form field (HTML checkboxes
	// post "on" when checked). Validated explicitly after binding, not via tags,
	// so the validator doesn't run before we translate "on" → true.
	Consent bool `json:"consent"`
}

// HasContactInfo is a quick helper for templates.
func (i Inquiry) HasContactInfo() bool { return i.Email != "" || i.Phone != "" }
