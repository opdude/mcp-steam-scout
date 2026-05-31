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
		"/api/featuredcategories": func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"top_sellers": map[string]any{
					"items": []map[string]any{
						{"id": 100, "name": "Game A", "discounted": false, "discount_percent": 0, "original_price": 5999, "final_price": 5999, "currency": "USD"},
						{"id": 200, "name": "Game B", "discounted": true, "discount_percent": 50, "original_price": 3999, "final_price": 1999, "currency": "EUR"},
						{"id": 300, "name": "Game C", "discounted": false, "discount_percent": 0, "original_price": 0, "final_price": 0, "currency": "USD"},
					},
				},
			})
		},
	})
	defer srv.Close()

	games, err := s.GetTrendingGames()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	found := 0
	for _, g := range games {
		switch g.ID {
		case "100":
			found++
			if g.Rank != 1 {
				t.Errorf("expected Game A rank 1, got %d", g.Rank)
			}
			if g.PriceAmount != "59.99" || g.PriceBaseAmount != "59.99" {
				t.Errorf("unexpected prices for Game A: %s / %s", g.PriceAmount, g.PriceBaseAmount)
			}
		case "200":
			found++
			if g.Rank != 2 {
				t.Errorf("expected Game B rank 2, got %d", g.Rank)
			}
			if !g.PriceIsDiscounted {
				t.Error("expected Game B to be discounted")
			}
			if g.PriceCurrency != "EUR" {
				t.Errorf("expected EUR currency, got %s", g.PriceCurrency)
			}
		case "300":
			found++
			if g.Rank != 3 {
				t.Errorf("expected Game C rank 3, got %d", g.Rank)
			}
		}
	}
	if found != 3 {
		t.Errorf("expected 3 steam games, found %d", found)
	}
}

func TestGetTrendingGames_SkipsEmptyNames(t *testing.T) {
	s, srv := newTestScraper(t, map[string]http.HandlerFunc{
		"/api/featuredcategories": func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"top_sellers": map[string]any{
					"items": []map[string]any{
						{"id": 100, "name": "", "discounted": false, "discount_percent": 0, "original_price": 0, "final_price": 0, "currency": "USD"},
						{"id": 200, "name": "Valid Game", "discounted": false, "discount_percent": 0, "original_price": 5999, "final_price": 5999, "currency": "USD"},
					},
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
		"/api/featuredcategories": func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(map[string]any{"top_sellers": map[string]any{"items": []map[string]any{}}})
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
			if g.ID == "1001" && g.Rank != 1 {
				t.Errorf("expected GOG Game A rank 1, got %d", g.Rank)
			}
			if g.ID == "1002" && g.Rank != 2 {
				t.Errorf("expected GOG Game B rank 2, got %d", g.Rank)
			}
			if g.ID != "1001" && g.ID != "1002" {
				t.Errorf("unexpected gog game id: %s", g.ID)
			}
		}
	}
	if gogCount != 2 {
		t.Errorf("expected 2 gog games, got %d", gogCount)
	}
}
