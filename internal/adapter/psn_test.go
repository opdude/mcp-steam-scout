package adapter

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// newTestPSNAdapter creates a PSNAdapter with a pre-configured access token, bypassing the
// OAuth exchange so tests can focus on individual methods.
func newTestPSNAdapter(t *testing.T, handler http.HandlerFunc) (*PSNAdapter, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	a := &PSNAdapter{
		AccessToken: "test-access-token",
		Client: &http.Client{
			Transport: &rewriteTransport{base: srv.URL, inner: srv.Client().Transport},
		},
	}
	return a, srv
}

// newTestPSNAdapterForAuth creates a PSNAdapter without an access token, so authenticate()
// can be tested end-to-end against a mock server.
func newTestPSNAdapterForAuth(t *testing.T, handler http.HandlerFunc) (*PSNAdapter, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	a := &PSNAdapter{
		NPSSO: "test-npsso",
		Client: &http.Client{
			Transport: &rewriteTransport{base: srv.URL, inner: srv.Client().Transport},
		},
	}
	return a, srv
}

func TestPSNAuthenticate_Success(t *testing.T) {
	a, srv := newTestPSNAdapterForAuth(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/authz/v3/oauth/authorize":
			w.Header().Set("Location", "com.scee.psxandroid.scecompcall://redirect?code=test-code-123")
			w.WriteHeader(http.StatusFound)
		case "/api/authz/v3/oauth/token":
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "test-token-xyz"})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	if err := a.authenticate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.AccessToken != "test-token-xyz" {
		t.Errorf("expected access token test-token-xyz, got %s", a.AccessToken)
	}
}

func TestPSNAuthenticate_AuthHTTPError(t *testing.T) {
	a, srv := newTestPSNAdapterForAuth(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	if err := a.authenticate(); err == nil {
		t.Fatal("expected error when auth endpoint returns non-redirect")
	}
}

func TestPSNAuthenticate_MissingCode(t *testing.T) {
	a, srv := newTestPSNAdapterForAuth(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Redirect without a code param
		w.Header().Set("Location", "com.scee.psxandroid.scecompcall://redirect")
		w.WriteHeader(http.StatusFound)
	}))
	defer srv.Close()

	if err := a.authenticate(); err == nil {
		t.Fatal("expected error when no code in redirect location")
	}
}

func TestPSNAuthenticate_TokenHTTPError(t *testing.T) {
	a, srv := newTestPSNAdapterForAuth(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/authz/v3/oauth/authorize":
			w.Header().Set("Location", "com.scee.psxandroid.scecompcall://redirect?code=test-code")
			w.WriteHeader(http.StatusFound)
		case "/api/authz/v3/oauth/token":
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
	defer srv.Close()

	if err := a.authenticate(); err == nil {
		t.Fatal("expected error when token endpoint returns non-200")
	}
}

func TestPSNAuthenticate_MissingAccessToken(t *testing.T) {
	a, srv := newTestPSNAdapterForAuth(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/authz/v3/oauth/authorize":
			w.Header().Set("Location", "com.scee.psxandroid.scecompcall://redirect?code=test-code")
			w.WriteHeader(http.StatusFound)
		case "/api/authz/v3/oauth/token":
			// Returns JSON but with no access_token field
			_ = json.NewEncoder(w).Encode(map[string]any{"token_type": "bearer"})
		}
	}))
	defer srv.Close()

	if err := a.authenticate(); err == nil {
		t.Fatal("expected error when access_token is absent from token response")
	}
}

func TestPSNGetLibrary_NoAccessToken(t *testing.T) {
	a := &PSNAdapter{}
	_, err := a.GetLibrary()
	if err == nil {
		t.Fatal("expected error when access token is empty")
	}
}

func TestPSNGetLibrary_HTTPError(t *testing.T) {
	a, srv := newTestPSNAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})
	defer srv.Close()

	_, err := a.GetLibrary()
	if err == nil {
		t.Fatal("expected error on non-200 response")
	}
}

func TestPSNGetLibrary_ParsesGamesAndPlaytime(t *testing.T) {
	a, srv := newTestPSNAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"titles": []map[string]any{
				{"titleId": "PPSA01234_00", "name": "Astro's Playroom", "playDuration": "PT10H30M"},
				{"titleId": "PPSA05678_00", "name": "Demon's Souls", "playDuration": "PT25H"},
				{"titleId": "PPSA09012_00", "name": "New Game", "playDuration": ""},
			},
		})
	})
	defer srv.Close()

	games, err := a.GetLibrary()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(games) != 3 {
		t.Fatalf("expected 3 games, got %d", len(games))
	}

	if games[0].ID != "PPSA01234_00" || games[0].Name != "Astro's Playroom" {
		t.Errorf("unexpected first game: %+v", games[0])
	}
	if games[0].PlaytimeMinutes != 630 { // 10h30m = 630 min
		t.Errorf("expected playtime 630, got %d", games[0].PlaytimeMinutes)
	}
	if games[1].PlaytimeMinutes != 1500 { // 25h = 1500 min
		t.Errorf("expected playtime 1500, got %d", games[1].PlaytimeMinutes)
	}
	if games[2].PlaytimeMinutes != 0 {
		t.Errorf("expected playtime 0 for unplayed game, got %d", games[2].PlaytimeMinutes)
	}
}

func TestPSNGetLibrary_EmptyLibrary(t *testing.T) {
	a, srv := newTestPSNAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"titles": []any{},
		})
	})
	defer srv.Close()

	games, err := a.GetLibrary()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(games) != 0 {
		t.Errorf("expected empty library, got %d games", len(games))
	}
}

func TestParseISO8601Duration(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"PT10H30M", 630},
		{"PT25H", 1500},
		{"PT45M", 45},
		{"PT1H", 60},
		{"PT0S", 0},
		{"", 0},
		{"PT2H15M30S", 135}, // seconds ignored
	}

	for _, tt := range tests {
		got := parseISO8601Duration(tt.input)
		if got != tt.expected {
			t.Errorf("parseISO8601Duration(%q) = %d, want %d", tt.input, got, tt.expected)
		}
	}
}
