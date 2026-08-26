// Package storage loads content from /content/<kind>/*.md files into models.
//
// Files use YAML frontmatter (delimited by `---`) followed by a Markdown body.
// Frontmatter is unmarshaled into the model struct; the body is rendered to
// HTML via goldmark.
//
// LoadAll() reads everything once at startup into in-memory slices. The site
// is small enough (tens of files) that this is the right tradeoff vs a DB.
// For dynamic updates, swap the implementation behind the same interface.
package storage

import (
	"fmt"
	"path/filepath"
	"sort"

	"github.com/nuteo/nuteo-web/internal/models"
)

// Store holds loaded content. Safe for concurrent reads after LoadAll.
type Store struct {
	Services   []models.Service
	Portfolio  []models.Portfolio
	Posts      []models.Post

	servicesBySlug  map[string]int
	portfolioBySlug map[string]int
	postsBySlug     map[string]int
}

// New creates a Store ready for LoadAll.
func New() *Store {
	return &Store{
		servicesBySlug:  map[string]int{},
		portfolioBySlug: map[string]int{},
		postsBySlug:     map[string]int{},
	}
}

// LoadAll reads every .md file under root and populates the store.
func (s *Store) LoadAll(root string) error {
	svcs, err := loadMarkdownDir(filepath.Join(root, "services"), func() *serviceModel { return &serviceModel{} })
	if err != nil {
		return fmt.Errorf("services: %w", err)
	}
	ports, err := loadMarkdownDir(filepath.Join(root, "portfolio"), func() *portfolioModel { return &portfolioModel{} })
	if err != nil {
		return fmt.Errorf("portfolio: %w", err)
	}
	posts, err := loadMarkdownDir(filepath.Join(root, "posts"), func() *postModel { return &postModel{} })
	if err != nil {
		return fmt.Errorf("posts: %w", err)
	}

	s.Services = make([]models.Service, len(svcs))
	for i, v := range svcs { s.Services[i] = v.Service }
	s.Portfolio = make([]models.Portfolio, len(ports))
	for i, v := range ports { s.Portfolio[i] = v.Portfolio }
	s.Posts = make([]models.Post, len(posts))
	for i, v := range posts { s.Posts[i] = v.Post }

	sort.SliceStable(s.Services, func(i, j int) bool {
		return s.Services[i].Order < s.Services[j].Order
	})

	// Build slug indexes AFTER sort, so index → post-sort position.
	for i, v := range s.Services       { s.servicesBySlug[v.Slug] = i }
	for i, v := range s.Portfolio      { s.portfolioBySlug[v.Slug] = i }
	for i, v := range s.Posts          { s.postsBySlug[v.Slug] = i }

	return nil
}

// ServiceBySlug returns the service with the given slug, or nil.
func (s *Store) ServiceBySlug(slug string) *models.Service {
	if i, ok := s.servicesBySlug[slug]; ok {
		return &s.Services[i]
	}
	return nil
}

// PortfolioBySlug returns the project with the given slug, or nil.
func (s *Store) PortfolioBySlug(slug string) *models.Portfolio {
	if i, ok := s.portfolioBySlug[slug]; ok {
		return &s.Portfolio[i]
	}
	return nil
}

// PostBySlug returns the post with the given slug, or nil.
func (s *Store) PostBySlug(slug string) *models.Post {
	if i, ok := s.postsBySlug[slug]; ok {
		return &s.Posts[i]
	}
	return nil
}

// FeaturedServices returns services marked featured=true, sorted by Order.
func (s *Store) FeaturedServices() []models.Service {
	out := []models.Service{}
	for _, v := range s.Services {
		if v.Featured { out = append(out, v) }
	}
	return out
}

// FeaturedPortfolio returns portfolio items marked featured=true.
func (s *Store) FeaturedPortfolio() []models.Portfolio {
	out := []models.Portfolio{}
	for _, v := range s.Portfolio {
		if v.Featured { out = append(out, v) }
	}
	return out
}
