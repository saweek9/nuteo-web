package main

import (
	"compress/gzip"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

// gzipStatic serves a `.gz` sibling of a static asset when the client
// accepts gzip encoding and the precompressed file exists.
//
// Falls back to the on-disk file when no .gz is present, so this is
// safe to enable unconditionally. Only acts on the /static prefix and
// on extensions that compress well (css, js, svg, html).
func gzipStatic(staticDir string) gin.HandlerFunc {
	compressible := map[string]bool{
		".css":  true,
		".js":   true,
		".svg":  true,
		".html": true,
		".json": true,
		".txt":  true,
		".xml":  true,
	}

	return func(c *gin.Context) {
		// Only intercept /static/ requests, otherwise pass through.
		if !strings.HasPrefix(c.Request.URL.Path, "/static/") {
			c.Next()
			return
		}
		ext := strings.ToLower(filepath.Ext(c.Request.URL.Path))
		if !compressible[ext] {
			c.Next()
			return
		}

		// Accept-Encoding must contain gzip.
		acceptEnc := c.GetHeader("Accept-Encoding")
		if !strings.Contains(strings.ToLower(acceptEnc), "gzip") {
			c.Next()
			return
		}

		// Try `<path>.gz` first.
		gzPath := filepath.Join(staticDir, strings.TrimPrefix(c.Request.URL.Path, "/static/")+".gz")
		f, err := os.Open(gzPath)
		if err != nil {
			c.Next() // fall back to the usual static handler
			return
		}
		defer f.Close()

		// Sanity-check: the file must not be world-writable.
		st, _ := f.Stat()
		if st == nil || st.Mode()&0o022 != 0 {
			c.Next()
			return
		}

		// Set Vary so caches don't serve gzip to clients that don't accept it.
		c.Header("Vary", "Accept-Encoding")
		c.Header("Content-Encoding", "gzip")
		c.Header("Content-Length", "")
		c.Status(http.StatusOK)
		c.Writer.Header().Set("Content-Type", mimeByExt(ext))
		gz, err := gzip.NewReader(f)
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}
		defer gz.Close()
		_, _ = io.Copy(c.Writer, gz)
		c.Abort() // short-circuit Gin's static handler
	}
}

func mimeByExt(ext string) string {
	switch ext {
	case ".css":
		return "text/css; charset=utf-8"
	case ".js":
		return "application/javascript; charset=utf-8"
	case ".svg":
		return "image/svg+xml"
	case ".html":
		return "text/html; charset=utf-8"
	case ".json":
		return "application/json; charset=utf-8"
	case ".xml":
		return "application/xml; charset=utf-8"
	case ".txt":
		return "text/plain; charset=utf-8"
	}
	return "application/octet-stream"
}
