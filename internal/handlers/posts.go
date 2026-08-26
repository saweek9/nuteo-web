package handlers

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/nuteo/nuteo-web/internal/models"
)

// PostBySlug is a thin wrapper around the store lookup for templates
// that need it (vs. handlers which use Store directly).
func (d *Deps) PostBySlug(slug string) *models.Post {
	return d.Store.PostBySlug(slug)
}

// Posts renders the blog index page — lists all posts newest-first.
func (d *Deps) Posts(c *gin.Context) {
	posts := d.Store.Posts
	// Newest first
	if len(posts) > 1 {
		// Reverse in place (lightweight; we have few posts)
		for i, j := 0, len(posts)-1; i < j; i, j = i+1, j-1 {
			posts[i], posts[j] = posts[j], posts[i]
		}
	}
	data := baseData(c, d, "Blog")
	data["PageDescription"] = "Engineering articles, post-mortems, and notes from the team."
	data["Posts"] = posts
	renderPage(c, data)
}

// PostDetail renders a single blog post.
//
// Renders 404 if the post doesn't exist.
func (d *Deps) PostDetail(c *gin.Context) {
	slug := c.Param("slug")
	p := d.Store.PostBySlug(slug)
	if p == nil {
		d.NotFound(c)
		return
	}
	data := baseData(c, d, p.Title)
	data["PageDescription"] = p.Summary
	data["Post"] = p
	data["Year"] = p.PublishedAt.Year()
	renderPage(c, data)
}

// PostsByTag renders posts filtered by tag (for /blog/tag/<tag>).
// Returns 404 if no posts match.
func (d *Deps) PostsByTag(c *gin.Context) {
	tag := c.Param("tag")
	var matched []models.Post
	for _, p := range d.Store.Posts {
		for _, t := range p.Tags {
			if t == tag {
				matched = append(matched, p)
				break
			}
		}
	}
	if len(matched) == 0 {
		d.NotFound(c)
		return
	}
	data := baseData(c, d, fmt.Sprintf("Tag: %s", tag))
	data["PageDescription"] = fmt.Sprintf("Articles tagged %s.", tag)
	data["Posts"] = matched
	data["Tag"] = tag
	renderPage(c, data)
}
