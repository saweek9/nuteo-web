package main

import (
	"fmt"
	"html"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/nuteo/nuteo-web/internal/config"
	"github.com/nuteo/nuteo-web/internal/storage"
)

// rssHandler generates an Atom 1.0 feed for the blog.
//
// One <entry> per post, newest-first.
func rssHandler(cfg *config.Config, store *storage.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Content-Type", "application/atom+xml; charset=utf-8")

		now := time.Now().UTC()
		latest := now
		for _, p := range store.Posts {
			if p.UpdatedAt.After(latest) {
				latest = p.UpdatedAt
			}
		}

		fmt.Fprintf(c.Writer, `<?xml version="1.0" encoding="utf-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <title>%s</title>
  <subtitle>Engineering articles, post-mortems, and notes.</subtitle>
  <link href="%s" rel="alternate"/>
  <link href="%s/rss.xml" rel="self"/>
  <id>%s/</id>
  <updated>%s</updated>
`, html.EscapeString(cfg.SiteName),
			html.EscapeString(cfg.SiteURL),
			html.EscapeString(cfg.SiteURL),
			html.EscapeString(cfg.SiteURL),
			latest.UTC().Format(time.RFC3339))

		// Entries (newest first)
		for i := 0; i < len(store.Posts); i++ {
			p := store.Posts[len(store.Posts)-1-i]
			fmt.Fprintf(c.Writer, `  <entry>
    <title>%s</title>
    <link href="%s/blog/%s"/>
    <id>%s/blog/%s</id>
    <updated>%s</updated>
    <published>%s</published>
    <author><name>%s</name></author>
    <summary>%s</summary>
  </entry>
`,
				html.EscapeString(p.Title),
				html.EscapeString(cfg.SiteURL), html.EscapeString(p.Slug),
				html.EscapeString(cfg.SiteURL), html.EscapeString(p.Slug),
				p.UpdatedAt.UTC().Format(time.RFC3339),
				p.PublishedAt.UTC().Format(time.RFC3339),
				html.EscapeString(p.Author),
				html.EscapeString(p.Summary),
			)
		}

		fmt.Fprintf(c.Writer, "</feed>\n")
	}
}
