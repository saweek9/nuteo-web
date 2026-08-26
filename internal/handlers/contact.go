package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"github.com/nuteo/nuteo-web/internal/models"
)

// newCSRFToken generates a per-request CSRF token (also stored in a cookie
// by the caller). The token is double-submit-pattern: the form posts the
// token back, and the handler compares it to the cookie.
func newCSRFToken() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// renderContact re-renders the contact form with an error message.
// Used when validation fails or spam is detected.
func (d *Deps) renderContact(c *gin.Context, errMsg string, inq models.Inquiry) {
	c.Status(http.StatusBadRequest)
	overridePage(c, "contact.html")
	data := baseData(c, d, "Contact")
	data["PageDescription"] = "Get in touch with nuteo solution."
	data["ContactEmail"]    = d.Cfg.ContactEmailTo
	data["CSRFToken"]       = newCSRFToken()
	data["Error"]           = errMsg
	data["Name"]            = inq.Name
	data["Email"]           = inq.Email
	data["Company"]         = inq.Company
	data["Topic"]           = inq.Topic
	data["Message"]         = inq.Message
	data["Consent"]         = inq.Consent
	renderPage(c, data)
}

// renderThanks renders the success page.
func (d *Deps) renderThanks(c *gin.Context, inq models.Inquiry) {
	c.Status(http.StatusOK)
	overridePage(c, "contact_thanks.html")
	data := baseData(c, d, "Thanks")
	data["PageDescription"] = "We received your message."
	data["Inquiry"] = inq
	renderPage(c, data)
}

// Contact renders the contact form (GET).
func (d *Deps) Contact(c *gin.Context) {
	data := baseData(c, d, "Contact")
	data["PageDescription"] = "Get in touch with nuteo solution."
	data["ContactEmail"]    = d.Cfg.ContactEmailTo
	data["CSRFToken"]       = newCSRFToken()
	renderPage(c, data)
}

// ContactSubmit handles POST /contact — the contact form.
//
// Read form fields directly (NOT via c.ShouldBind) so we can convert the
// HTML checkbox "on" → bool before the validator runs. ShouldBind would call
// validate.Struct first, which doesn't know about the conversion.
func (d *Deps) ContactSubmit(c *gin.Context) {
	inq := models.Inquiry{
		Name:    c.PostForm("name"),
		Email:   c.PostForm("email"),
		Company: c.PostForm("company"),
		Phone:   c.PostForm("phone"),
		Topic:   c.PostForm("topic"),
		Message: c.PostForm("message"),
		Website: c.PostForm("website"), // honeypot
		Consent: c.PostForm("consent") == "on",
	}

	// Honeypot — silent 200 OK to bots
	if inq.Website != "" {
		d.renderThanks(c, inq)
		return
	}

	// Validate (skip the Consent field; we already converted it manually)
	v := validator.New()
	if err := v.StructPartial(inq, "Name", "Email", "Company", "Phone", "Topic", "Message"); err != nil {
		d.renderContact(c, "Please fill in name, email, and message.", inq)
		return
	}

	if !inq.Consent {
		d.renderContact(c, "Please confirm you've read the privacy notice.", inq)
		return
	}

	// Fill metadata
	inq.ID = uuid.NewString()
	inq.ReceivedAt = time.Now().UTC()
	inq.IP = c.ClientIP()
	inq.UserAgent = c.Request.UserAgent()
	inq.Referrer = c.Request.Referer()

	// Send notification email (best-effort — log on error but still thank user)
	if err := d.Mail.SendInquiry(c.Request.Context(), inq); err != nil {
		c.Header("X-Email-Status", "failed")
	}

	d.renderThanks(c, inq)
}