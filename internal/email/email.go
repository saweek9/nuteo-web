// Package email sends transactional email from the contact form.
//
// Sender is an interface so we can swap implementations (SMTP today,
// Resend/SendGrid later) without touching handlers.
package email

import (
	"context"
	"fmt"
	"net/smtp"
	"strings"

	"github.com/nuteo/nuteo-web/internal/config"
	"github.com/nuteo/nuteo-web/internal/models"
)

// Sender is the interface every backend implements.
type Sender interface {
	SendInquiry(ctx context.Context, inq models.Inquiry) error
}

// --- SMTP implementation ---

// SMTPSender sends email via a standard SMTP server (Gmail, Mailgun, SES, etc.).
type SMTPSender struct {
	cfg *config.Config
	auth smtp.Auth
}

// NewSMTPSender constructs an SMTPSender from the config.
// If SMTP_USER / SMTP_PASS are empty, no auth is used (rare).
func NewSMTPSender(cfg *config.Config) *SMTPSender {
	var auth smtp.Auth
	if cfg.SMTPUser != "" {
		auth = smtp.PlainAuth("", cfg.SMTPUser, cfg.SMTPPass, cfg.SMTPHost)
	}
	return &SMTPSender{cfg: cfg, auth: auth}
}

// SendInquiry sends a notification email to CONTACT_EMAIL_TO with the inquiry.
func (s *SMTPSender) SendInquiry(_ context.Context, inq models.Inquiry) error {
	subject := fmt.Sprintf("[nuteo inquiry] %s — %s", inq.Topic, inq.Name)
	body := buildPlainText(inq)

	msg := []byte(strings.Join([]string{
		fmt.Sprintf("From: %s", s.cfg.ContactEmailFrom),
		fmt.Sprintf("To: %s", s.cfg.ContactEmailTo),
		fmt.Sprintf("Subject: %s", subject),
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"",
		body,
	}, "\r\n"))

	addr := fmt.Sprintf("%s:%d", s.cfg.SMTPHost, s.cfg.SMTPPort)
	return smtp.SendMail(addr, s.auth, s.cfg.ContactEmailFrom,
		[]string{s.cfg.ContactEmailTo}, msg)
}

func buildPlainText(inq models.Inquiry) string {
	var b strings.Builder
	fmt.Fprintf(&b, "New inquiry received\n\n")
	fmt.Fprintf(&b, "From:    %s <%s>\n", inq.Name, inq.Email)
	if inq.Company != "" { fmt.Fprintf(&b, "Company: %s\n", inq.Company) }
	if inq.Phone   != "" { fmt.Fprintf(&b, "Phone:   %s\n", inq.Phone) }
	fmt.Fprintf(&b, "Topic:   %s\n", inq.Topic)
	fmt.Fprintf(&b, "When:    %s\n", inq.ReceivedAt.Format("2006-01-02 15:04:05 UTC"))
	fmt.Fprintf(&b, "IP:      %s\n", inq.IP)
	fmt.Fprintf(&b, "UA:      %s\n", inq.UserAgent)
	fmt.Fprintln(&b, "")
	fmt.Fprintln(&b, "Message:")
	fmt.Fprintln(&b, strings.Repeat("-", 60))
	fmt.Fprintln(&b, inq.Message)
	return b.String()
}
