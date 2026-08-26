package main

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/nuteo/nuteo-web/internal/config"
	"github.com/nuteo/nuteo-web/internal/storage"
)

// sitemapHandler generates a sitemap.xml from in-memory content.
// One <url> per: home, services index, each service, portfolio index, each project, about, contact.
func sitemapHandler(cfg *config.Config, store *storage.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Content-Type", "application/xml")
		now := time.Now().UTC().Format("2006-01-02")

		fmt.Fprintf(c.Writer, `<?xml version="1.0" encoding="UTF-8"?>`+"\n")
		fmt.Fprintf(c.Writer, `<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`+"\n")

		emit := func(loc string, mod string) {
			fmt.Fprintf(c.Writer, "  <url><loc>%s%s</loc><lastmod>%s</lastmod></url>\n",
				cfg.SiteURL, loc, mod)
		}

		emit("/", now)
		emit("/services", now)
		for _, s := range store.Services {
			emit("/services/"+s.Slug, s.UpdatedAt.Format("2006-01-02"))
		}
		emit("/portfolio", now)
		for _, p := range store.Portfolio {
			emit("/portfolio/"+p.Slug, p.PublishedAt.Format("2006-01-02"))
		}
		emit("/about", now)
		emit("/contact", now)
		emit("/faq", now)
		emit("/blog", now)
		for _, p := range store.Posts {
			emit("/blog/"+p.Slug, p.UpdatedAt.Format("2006-01-02"))
		}

		fmt.Fprintf(c.Writer, `</urlset>`)
	}
}
