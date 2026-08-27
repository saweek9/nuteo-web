// Package handlers — FAQ page.
//
// FAQ data lives in the i18n bundle (i18n/{lang}.yaml under
// the `faq.entries` key) so questions and answers can be translated
// without recompiling.
package handlers

import (
	"github.com/gin-gonic/gin"
)

// faqEntry is a single question/answer pair for the FAQ page.
type faqEntry struct {
	Question string
	Answer   string // HTML allowed (uses safeHTML in template)
}

// FAQ renders the FAQ page. Questions are pulled from the i18n bundle
// so each locale can ship its own set.
func (d *Deps) FAQ(c *gin.Context) {
	lang := i18nLang(c, d)

	entries := loadFAQ(d, lang)

	data := baseData(c, d, "FAQ")
	data["PageDescription"] = "Common questions about working with nuteo solution."
	data["FAQEntries"]      = entries
	data["ContactLabel"]    = d.I18n.T(lang, "faq.contact_label")
	data["ContactLink"]     = d.I18n.T(lang, "faq.contact_link")
	renderPage(c, data)
}

// loadFAQ reads the FAQ list from the i18n bundle.
//
// YAML schema:
//   faq:
//     entries:
//       - q: "Question 1?"
//         a: "Answer 1."
//       - q: "Question 2?"
//         a: "Answer 2."
//
// Falls back to a single English placeholder if the bundle is missing
// or the key path is wrong (test environment).
func loadFAQ(d *Deps, lang string) []faqEntry {
	if d.I18n == nil {
		return []faqEntry{{
			Question: "FAQ",
			Answer:   "The FAQ requires the i18n bundle to be wired. See internal/i18n.",
		}}
	}

	raw := d.I18n.RawMap(lang)
	root, _ := raw["faq"].(map[string]any)
	if root == nil {
		return []faqEntry{{
			Question: d.I18n.T(lang, "faq.fallback_q"),
			Answer:   d.I18n.T(lang, "faq.fallback_a"),
		}}
	}
	arr, _ := root["entries"].([]any)
	if len(arr) == 0 {
		return []faqEntry{{
			Question: d.I18n.T(lang, "faq.fallback_q"),
			Answer:   d.I18n.T(lang, "faq.fallback_a"),
		}}
	}

	out := make([]faqEntry, 0, len(arr))
	for _, item := range arr {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		q, _ := m["q"].(string)
		a, _ := m["a"].(string)
		if q == "" || a == "" {
			continue
		}
		out = append(out, faqEntry{Question: q, Answer: a})
	}
	return out
}
