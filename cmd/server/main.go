// nuteo-web — main entrypoint.
//
// Wires config → content store → email sender → Gin handlers → HTTP server.
// All dependencies are constructed once at startup; content is loaded once
// into memory; handlers read from the in-memory store per request.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/nuteo/nuteo-web/internal/config"
	"github.com/nuteo/nuteo-web/internal/email"
	"github.com/nuteo/nuteo-web/internal/handlers"
	"github.com/nuteo/nuteo-web/internal/i18n"
	mw "github.com/nuteo/nuteo-web/internal/middleware"
	"github.com/nuteo/nuteo-web/internal/storage"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		logger.Error("config load", "err", err)
		os.Exit(1)
	}
	if !cfg.IsProd() {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// Load content
	store := storage.New()
	if err := store.LoadAll(cfg.ContentDir); err != nil {
		logger.Error("content load", "err", err, "dir", cfg.ContentDir)
		os.Exit(1)
	}
	logger.Info("content loaded",
		"services", len(store.Services),
		"portfolio", len(store.Portfolio),
		"posts", len(store.Posts),
	)

	// Load i18n bundle from ./i18n
	bundle, err := i18n.NewBundle("./i18n")
	if err != nil {
		logger.Error("i18n load", "err", err)
		os.Exit(1)
	}

	// Wire dependencies
	mail := email.NewSMTPSender(cfg)
	deps := &handlers.Deps{Cfg: cfg, Store: store, Mail: mail, I18n: bundle}

	// Build router
	r := gin.New()
	r.MaxMultipartMemory = 1 << 20 // 1 MB — contact form is tiny

	// Compile templates once — one *template.Template per page (each already
	// has header/footer partials parsed in). Stash the map in context.
	tmpls := deps.CompileTemplates()
	r.Use(func(c *gin.Context) {
		c.Set("tmpls", tmpls)
		// Lookup by route pattern registered for this handler is awkward in
		// a global middleware, so each handler sets its own page via a
		// helper. We still expose the map so the helper can grab the right one.
		c.Next()
	})
	r.Use(mw.RequestID())
	r.Use(mw.Logger(logger))
	r.Use(mw.SecurityHeaders())
	// Rate limit (60 req / minute per IP — generous for marketing sites)
	rateLimiter := mw.NewIPRateLimiter(60, time.Minute)
	r.Use(rateLimiter.RateLimit())

	// Static
	staticDir, _ := filepath.Abs(cfg.StaticDir)
	r.Static("/static", staticDir)
	r.StaticFile("/favicon.ico", filepath.Join(staticDir, "favicon.ico"))

	// SEO
	r.GET("/healthz", func(c *gin.Context) { c.String(http.StatusOK, "ok") })
	r.GET("/robots.txt", func(c *gin.Context) {
		c.Header("Content-Type", "text/plain")
		c.String(http.StatusOK, "User-agent: *\nAllow: /\nSitemap: %s/sitemap.xml\n", cfg.SiteURL)
	})
	r.GET("/sitemap.xml", sitemapHandler(cfg, store))
	r.GET("/rss.xml",     rssHandler(cfg, store))

	// Pages — setPage middleware tags the template per route. The POST handler
	// overrides the page tag at runtime when rendering the thanks page.
	r.GET("/",                  handlers.SetPage("home.html"),         deps.Home)
	r.GET("/services",          handlers.SetPage("services.html"),     deps.Services)
	r.GET("/services/:slug",    handlers.SetPage("service.html"),     deps.ServiceDetail)
	r.GET("/portfolio",         handlers.SetPage("portfolio.html"),    deps.Portfolio)
	r.GET("/portfolio/:slug",   handlers.SetPage("project.html"),     deps.ProjectDetail)
	r.GET("/blog",              handlers.SetPage("posts.html"),        deps.Posts)
	r.GET("/blog/tag/:tag",     handlers.SetPage("posts.html"),        deps.PostsByTag)
	r.GET("/blog/:slug",        handlers.SetPage("post.html"),         deps.PostDetail)
	r.GET("/about",             handlers.SetPage("about.html"),       deps.About)
	r.GET("/faq",               handlers.SetPage("faq.html"),         deps.FAQ)
	r.GET("/search",            handlers.SetPage("search.html"),       deps.Search)
	r.GET("/contact",           handlers.SetPage("contact.html"),     deps.Contact)
	r.POST("/contact",          handlers.SetPage("contact.html"),     deps.ContactSubmit)
	r.POST("/newsletter",       deps.NewsletterSubscribe)

	// Admin panel — protected by RequireAdmin middleware
	adminSecret := []byte(cfg.CSRFSecret)
	r.GET("/admin/login",       handlers.SetPage("admin/login.html"), deps.AdminLogin)
	r.POST("/admin/login",      handlers.SetPage("admin/login.html"), deps.AdminLogin)
	r.POST("/admin/logout",     deps.AdminLogout)
	adminGroup := r.Group("/admin")
	adminGroup.Use(mw.RequireAdmin(adminSecret))
	adminGroup.GET("",                   handlers.SetPage("admin/dashboard.html"), deps.AdminDashboard)
	adminGroup.GET("/dashboard",         handlers.SetPage("admin/dashboard.html"), deps.AdminDashboard)

	// HTTP server with graceful shutdown
	srv := &http.Server{
		Addr:         cfg.Addr,
		Handler:      r,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	}

	go func() {
		logger.Info("server listening", "addr", cfg.Addr, "env", cfg.Env)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("listen", "err", err)
			os.Exit(1)
		}
	}()

	// Wait for SIGINT / SIGTERM
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	logger.Info("shutdown initiated")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("shutdown", "err", err)
	}
	logger.Info("bye")
}
