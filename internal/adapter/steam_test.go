package adapter

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestAdapter(t *testing.T, handler http.HandlerFunc) (*SteamAdapter, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	a := &SteamAdapter{
		APIKey: "test-key",
		Client: srv.Client(),
	}
	// Point requests at the test server by overriding via a transport that rewrites the host.
	// Simpler: we'll use a custom RoundTripper that redirects all requests to the test server.
	a.Client = &http.Client{
		Transport: &rewriteTransport{base: srv.URL, inner: srv.Client().Transport},
	}
	return a, srv
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

func TestGetLibrary_NoAPIKey(t *testing.T) {
	a := NewSteamAdapter("", "")
	_, err := a.GetLibrary()
	if err == nil {
		t.Fatal("expected error when API key is empty")
	}
}

func TestGetLibrary_HTTPError(t *testing.T) {
	a, srv := newTestAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})
	defer srv.Close()

	_, err := a.GetLibrary()
	if err == nil {
		t.Fatal("expected error on non-200 response")
	}
}

func TestGetLibrary_ParsesGamesAndPlaytime(t *testing.T) {
	a, srv := newTestAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"response": map[string]any{
				"games": []map[string]any{
					{"appid": 220, "name": "Half-Life 2", "playtime_forever": 342},
					{"appid": 440, "name": "Team Fortress 2", "playtime_forever": 0},
				},
			},
		})
	})
	defer srv.Close()
	a.DefaultSteamID = "12345"

	games, err := a.GetLibrary()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(games) != 2 {
		t.Fatalf("expected 2 games, got %d", len(games))
	}
	if games[0].ID != "220" || games[0].Name != "Half-Life 2" {
		t.Errorf("unexpected first game: %+v", games[0])
	}
	if games[0].PlaytimeMinutes != 342 {
		t.Errorf("expected playtime 342, got %d", games[0].PlaytimeMinutes)
	}
	if games[1].PlaytimeMinutes != 0 {
		t.Errorf("expected playtime 0 for TF2, got %d", games[1].PlaytimeMinutes)
	}
}

func TestGetLibrary_EmptyLibrary(t *testing.T) {
	a, srv := newTestAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"response": map[string]any{},
		})
	})
	defer srv.Close()
	a.DefaultSteamID = "12345"

	games, err := a.GetLibrary()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(games) != 0 {
		t.Errorf("expected empty library, got %d games", len(games))
	}
}

func TestResolveVanityURL_NoAPIKey(t *testing.T) {
	a := NewSteamAdapter("", "")
	_, err := a.ResolveVanityURL("someuser")
	if err == nil {
		t.Fatal("expected error when API key is empty")
	}
}

func TestResolveVanityURL_HTTPError(t *testing.T) {
	a, srv := newTestAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	defer srv.Close()

	_, err := a.ResolveVanityURL("someuser")
	if err == nil {
		t.Fatal("expected error on non-200 response")
	}
}

func TestResolveVanityURL_NotFound(t *testing.T) {
	a, srv := newTestAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"response": map[string]any{
				"success": 42,
				"message": "No match",
			},
		})
	})
	defer srv.Close()

	_, err := a.ResolveVanityURL("unknownuser")
	if err == nil {
		t.Fatal("expected error when success != 1")
	}
}

func TestResolveVanityURL_Success(t *testing.T) {
	a, srv := newTestAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"response": map[string]any{
				"steamid": "76561197962821445",
				"success": 1,
			},
		})
	})
	defer srv.Close()

	id, err := a.ResolveVanityURL("opdude")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "76561197962821445" {
		t.Errorf("expected steam ID 76561197962821445, got %s", id)
	}
}
