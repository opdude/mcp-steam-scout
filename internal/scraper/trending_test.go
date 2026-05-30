package scraper

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestScraper(t *testing.T, handlers map[string]http.HandlerFunc) (*TrendingScraper, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h, ok := handlers[r.URL.Path]
		if ok {
			h(w, r)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	s := &TrendingScraper{
		Client: &http.Client{
			Transport: &rewriteTransport{base: srv.URL, inner: srv.Client().Transport},
		},
	}
	return s, srv
}

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
	s, srv := newTestScraper(t, map[string]http.HandlerFunc{})
	defer srv.Close()

	games, err := s.GetTrendingGames()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(games) != 0 {
		t.Fatalf("expected 0 games when all sources fail, got %d", len(games))
	}
}

func TestGetTrendingGames_ParsesItems(t *testing.T) {
	s, srv := newTestScraper(t, map[string]http.HandlerFunc{
		"/search/results/": func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"items": []map[string]any{
					{"name": "Game A", "logo": "https://cdn.example.com/store_item_assets/steam/apps/100/capsule.jpg"},
					{"name": "Game B", "logo": "https://cdn.example.com/store_item_assets/steam/apps/200/capsule.jpg"},
					{"name": "Game C", "logo": "https://cdn.example.com/store_item_assets/steam/apps/300/capsule.jpg"},
				},
			})
		},
	})
	defer srv.Close()

	games, err := s.GetTrendingGames()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(games) < 3 {
		t.Fatalf("expected at least 3 games, got %d", len(games))
	}
	found := 0
	for _, g := range games {
		switch g.ID {
		case "100", "200", "300":
			found++
		}
	}
	if found != 3 {
		t.Errorf("expected 3 steam games, found %d", found)
	}
}

func TestGetTrendingGames_Deduplicates(t *testing.T) {
	s, srv := newTestScraper(t, map[string]http.HandlerFunc{
		"/search/results/": func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"items": []map[string]any{
					{"name": "Game A", "logo": "https://cdn.example.com/store_item_assets/steam/apps/100/capsule.jpg"},
					{"name": "Game A", "logo": "https://cdn.example.com/store_item_assets/steam/apps/100/capsule.jpg"},
					{"name": "Game B", "logo": "https://cdn.example.com/store_item_assets/steam/apps/200/capsule.jpg"},
				},
			})
		},
	})
	defer srv.Close()

	games, err := s.GetTrendingGames()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	steamCount := 0
	for _, g := range games {
		if g.Platform == "steam" {
			steamCount++
		}
	}
	if steamCount != 2 {
		t.Fatalf("expected 2 steam games after dedup, got %d", steamCount)
	}
}

func TestGetTrendingGames_SkipsEmptyNamesAndMissingIDs(t *testing.T) {
	s, srv := newTestScraper(t, map[string]http.HandlerFunc{
		"/search/results/": func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"items": []map[string]any{
					{"name": "", "logo": "https://cdn.example.com/store_item_assets/steam/apps/100/capsule.jpg"},
					{"name": "No ID Game", "logo": "https://cdn.example.com/no-app-id-here.jpg"},
					{"name": "Valid Game", "logo": "https://cdn.example.com/store_item_assets/steam/apps/200/capsule.jpg"},
				},
			})
		},
	})
	defer srv.Close()

	games, err := s.GetTrendingGames()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	steamCount := 0
	for _, g := range games {
		if g.Platform == "steam" {
			steamCount++
		}
	}
	if steamCount != 1 {
		t.Fatalf("expected 1 steam game, got %d", steamCount)
	}
}

func TestGetTrending_GOG(t *testing.T) {
	s, srv := newTestScraper(t, map[string]http.HandlerFunc{
		"/search/results/": func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(map[string]any{"items": []map[string]any{}})
		},
		"/games/ajax/filtered": func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"products": []map[string]any{
					{"id": 1001, "title": "GOG Game A"},
					{"id": 1002, "title": "GOG Game B"},
				},
			})
		},
	})
	defer srv.Close()

	games, err := s.GetTrendingGames()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	gogCount := 0
	for _, g := range games {
		if g.Platform == "gog" {
			gogCount++
			if g.ID != "1001" && g.ID != "1002" {
				t.Errorf("unexpected gog game id: %s", g.ID)
			}
		}
	}
	if gogCount != 2 {
		t.Errorf("expected 2 gog games, got %d", gogCount)
	}
}
