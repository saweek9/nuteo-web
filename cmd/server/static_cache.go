package main

import (
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

// staticCache serves files under /static/* with a long max-age and
// immutable cache hint, so browsers cache aggressively across visits.
//
// We can't hash filenames without a build pipeline, so we rely on:
//   - 1-year max-age (content doesn't change without a deploy)
//   - Cache-Control: immutable (HTTP cache directive)
//   - The service worker invalidating on demand when updates ship.
//
// Files we don't want cached (sitemap, RSS, sw.js) are routed
// separately with their own Cache-Control.
func staticCache(staticDir string) gin.HandlerFunc {
	// Compressible extensions get gzip pre-compressed variants if present.
	return func(c *gin.Context) {
		// Apply cache headers.
		c.Header("Cache-Control", "public, max-age=31536000, immutable")

		// Resolve + serve the file via http.ServeFile.
		p := c.Param("filepath")
		full := filepath.Join(staticDir, filepath.Clean("/"+p))
		// Path-traversal guard.
		if !strings.HasPrefix(full, staticDir) {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
		http.ServeFile(c.Writer, c.Request, full)
	}
}
