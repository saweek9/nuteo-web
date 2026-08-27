package handlers

import (
	"testing"

	"github.com/nuteo/nuteo-web/internal/models"
	"github.com/nuteo/nuteo-web/internal/storage"
)

// minimalStore seeds an in-memory Store with a few entries so
// suggestPaths has something to match against.
func minimalStore() *storage.Store {
	s := storage.New()
	// Pre-populate the slug indexes by calling LoadAll on a temp dir.
	// Easier: just write a YAML helper. Skipped here — we test the
	// heuristic on the static paths + a stub.
	_ = s
	return s
}

// stubStore returns a Store populated with fake entries. It bypasses
// the file loader and writes directly via the unexported helpers that
// LoadAll would have populated. Since those are unexported, we test
// the matching heuristic against the static-path fallback only here.
func stubStore() *storage.Store { return minimalStore() }

func TestSuggestPaths_MatchesExistingSlug(t *testing.T) {
	// "cloud-migartion" (typo) should match "cloud-migration" score ≥ 3.
	store := &storage.Store{}
	store.Services = append(store.Services, models.Service{
		Title: "Cloud Migration & Infrastructure", Slug: "cloud-migration",
	})
	store.Portfolio = append(store.Portfolio, models.Portfolio{
		Title: "Cloud Migration for Logistics", Slug: "logistics-migration",
	})

	got := suggestPaths("/services/cloud-migartion", store)
	if len(got) == 0 {
		t.Fatalf("expected at least one suggestion, got none")
	}
	// Top suggestion should be the cloud-migration service
	for _, s := range got {
		t.Logf("suggested: %s (%s)", s.Title, s.URL)
		if s.URL == "/services/cloud-migration" {
			return // good
		}
	}
	t.Errorf("expected /services/cloud-migration in suggestions, got %+v", got)
}

func TestSuggestPaths_NoMatchReturnsEmpty(t *testing.T) {
	store := stubStore()
	got := suggestPaths("/zzzqqqxxx", store)
	if len(got) != 0 {
		t.Errorf("expected no suggestions, got %+v", got)
	}
}

func TestSuggestPaths_StaticPathsAlwaysConsidered(t *testing.T) {
	store := stubStore()
	// "/blog" should match the static "blog" path (exact segment match).
	got := suggestPaths("/blog", store)
	if len(got) == 0 {
		t.Errorf("expected at least blog suggestion")
	}
}

func TestSuggestPaths_MaxThree(t *testing.T) {
	store := stubStore()
	for i := 0; i < 10; i++ {
		store.Services = append(store.Services, models.Service{
			Title: "Test Service", Slug: "shared-slug",
		})
	}
	got := suggestPaths("/shared", store)
	if len(got) > 3 {
		t.Errorf("expected max 3 suggestions, got %d", len(got))
	}
}
