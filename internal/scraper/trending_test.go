package scraper

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestScraper(t *testing.T, handler http.HandlerFunc) (*TrendingScraper, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	s := &TrendingScraper{
		Client: &http.Client{
			Transport: &rewriteTransport{base: srv.URL, inner: srv.Client().Transport},
		},
	}
	return s, srv
}

// rewriteTransport rewrites the scheme+host of every request to point at the test server.
type rewriteTransport struct {
	base  string
	inner http.RoundTripper
}

func (r *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req2 := req.Clone(req.Context())
	req2.URL.Scheme = "http"
	req2.URL.Host = r.base[len("http://"):]
	return r.inner.RoundTrip(req2)
}

func TestGetTrendingGames_HTTPError(t *testing.T) {
	s, srv := newTestScraper(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	})
	defer srv.Close()

	_, err := s.GetTrendingGames()
	if err == nil {
		t.Fatal("expected error on non-200 response")
	}
}

func TestGetTrendingGames_ParsesTopSellersAndNewReleases(t *testing.T) {
	s, srv := newTestScraper(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"top_sellers": map[string]any{
				"items": []map[string]any{
					{"id": 1, "name": "Game A"},
					{"id": 2, "name": "Game B"},
				},
			},
			"new_releases": map[string]any{
				"items": []map[string]any{
					{"id": 3, "name": "Game C"},
				},
			},
		})
	})
	defer srv.Close()

	games, err := s.GetTrendingGames()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(games) != 3 {
		t.Fatalf("expected 3 games, got %d", len(games))
	}
}

func TestGetTrendingGames_DeduplicatesAcrossSections(t *testing.T) {
	s, srv := newTestScraper(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"top_sellers": map[string]any{
				"items": []map[string]any{
					{"id": 1, "name": "Game A"},
				},
			},
			"new_releases": map[string]any{
				"items": []map[string]any{
					{"id": 1, "name": "Game A"}, // duplicate
					{"id": 2, "name": "Game B"},
				},
			},
		})
	})
	defer srv.Close()

	games, err := s.GetTrendingGames()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(games) != 2 {
		t.Fatalf("expected 2 games after dedup, got %d", len(games))
	}
}

func TestGetTrendingGames_SkipsEmptyNames(t *testing.T) {
	s, srv := newTestScraper(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"top_sellers": map[string]any{
				"items": []map[string]any{
					{"id": 1, "name": ""},
					{"id": 2, "name": "Valid Game"},
				},
			},
			"new_releases": map[string]any{
				"items": []map[string]any{},
			},
		})
	})
	defer srv.Close()

	games, err := s.GetTrendingGames()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(games) != 1 {
		t.Fatalf("expected 1 game, got %d", len(games))
	}
	if games[0].Name != "Valid Game" {
		t.Errorf("unexpected game name: %s", games[0].Name)
	}
}
