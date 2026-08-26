package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// FAQEntry is a single question/answer pair for the FAQ page.
type FAQEntry struct {
	Question string
	Answer   string // plain text; rendered through safeHTML for inline links
}

// FAQ renders the FAQ page with native <details>/<summary> accordions.
func (d *Deps) FAQ(c *gin.Context) {
	data := baseData(c, d, "FAQ")
	data["PageDescription"] = "Common questions about working with nuteo solution."
	data["FAQs"] = []FAQEntry{
		{
			Question: "What's your typical engagement size?",
			Answer:   "Most projects run 6–12 weeks with 1–2 senior engineers embedded in your team. We scope to a concrete outcome — a migrated platform, a working CI/CD pipeline, an incident review — and we hand off everything we build. Longer retainers (1–2 days/week) are common for ongoing platform work.",
		},
		{
			Question: "Do you work remotely?",
			Answer:   "Yes. Most of our work is async-first with overlapping working hours. We have a few hours of guaranteed overlap with APAC and EU teams. For Bangkok-based clients, we're happy to be on-site 1–2 days per week.",
		},
		{
			Question: "What tech stacks do you specialize in?",
			Answer:   "Backend: <strong>Go, Python, TypeScript/Node</strong>. Infrastructure: <strong>Kubernetes, Terraform, AWS/GCP</strong>. Observability: <strong>OpenTelemetry, Prometheus, Grafana</strong>. CI/CD: <strong>GitHub Actions, ArgoCD, Buildkite</strong>. We're happy to learn your stack — but most of our leverage is in those areas.",
		},
		{
			Question: "Can you help with an in-flight incident?",
			Answer:   "Yes, we offer incident response retainers. We'll embed with your on-call team within 2 hours of a P0/P1 and work the issue alongside you, then write the post-mortem. <a href=\"/services/devops\">See our SRE offering →</a>",
		},
		{
			Question: "Do you build new products from scratch?",
			Answer:   "Occasionally, but we're not a product studio. Most of our work is platform/infrastructure — the systems that run things, not the things themselves. If you need a new app built, we can refer you.",
		},
		{
			Question: "How do you price engagements?",
			Answer:   "Two ways: <strong>fixed-price</strong> for well-defined scopes (e.g. \"migrate this Kubernetes cluster to EKS in 8 weeks\") or <strong>time & materials</strong> for exploratory work. We don't do value-based pricing — it doesn't fit how engineering work actually goes. Most engagements land in the $15k–$80k range.",
		},
		{
			Question: "Where do you host things?",
			Answer:   "For client work, we deploy to wherever you already are. For our own infrastructure (this website, internal tools), we use a mix of <a href=\"https://fly.io\" rel=\"noopener\">Fly.io</a>, AWS, and a few self-hosted VPSes.",
		},
		{
			Question: "Do you sign NDAs?",
			Answer:   "Yes, before any detailed conversation about your systems. We're happy to use your standard NDA or ours.",
		},
		{
			Question: "What size company do you work with?",
			Answer:   "Anywhere from Series A startups to mid-market enterprise. We've worked with 5-person teams and 500-engineer orgs. The sweet spot is \"we know what we want but need help getting there\" — usually 20–200 engineers.",
		},
	}
	renderPage(c, data)
}

// Compile-time check we don't break the http import when trimmed.
var _ = http.StatusOK
