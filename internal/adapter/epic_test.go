package adapter

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newTestEpicAdapter creates an EpicAdapter with a pre-set access token,
// bypassing the OAuth refresh so tests can focus on library fetching.
func newTestEpicAdapter(t *testing.T, handler http.HandlerFunc) (*EpicAdapter, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	a := &EpicAdapter{
		accessToken: "test-access-token",
		client: &http.Client{
			Transport: &rewriteTransport{base: srv.URL, inner: srv.Client().Transport},
		},
	}
	return a, srv
}

// newTestEpicAdapterForAuth creates an EpicAdapter with a refresh token but
// no access token, so NewEpicAdapter-style auth can be tested.
func newTestEpicAdapterForAuth(t *testing.T, handler http.HandlerFunc) (*EpicAdapter, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	a := &EpicAdapter{
		refreshToken: "test-refresh-token",
		client: &http.Client{
			Transport: &rewriteTransport{base: srv.URL, inner: srv.Client().Transport},
		},
	}
	return a, srv
}

func TestEpicNewAdapter_AuthOK(t *testing.T) {
	// We can test refreshAccessToken directly
	a, srv := newTestEpicAdapterForAuth(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/account/api/oauth/token" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_ = json.NewEncoder(w).Encode(epicTokenResponse{
			AccessToken:  "new-access-token",
			RefreshToken: "new-refresh-token",
			ExpiresIn:    7200,
			AccountID:    "acct123",
			DisplayName:  "TestUser",
		})
	})
	defer srv.Close()

	if err := a.refreshAccessToken(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.accessToken != "new-access-token" {
		t.Errorf("expected access token new-access-token, got %s", a.accessToken)
	}
	if a.accountID != "acct123" {
		t.Errorf("expected accountID acct123, got %s", a.accountID)
	}
	if a.displayName != "TestUser" {
		t.Errorf("expected displayName TestUser, got %s", a.displayName)
	}
}

func TestEpicNewAdapter_AuthHTTPError(t *testing.T) {
	a, srv := newTestEpicAdapterForAuth(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})
	defer srv.Close()

	if err := a.refreshAccessToken(); err == nil {
		t.Fatal("expected error on non-200 token response")
	}
}

func TestEpicNewAdapter_AuthErrorInResponse(t *testing.T) {
	a, srv := newTestEpicAdapterForAuth(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(epicTokenResponse{
			Error:        "invalid_grant",
			ErrorMessage: "Invalid refresh token",
		})
	})
	defer srv.Close()

	if err := a.refreshAccessToken(); err == nil {
		t.Fatal("expected error when response contains error field")
	}
}

func TestEpicNewAdapter_AuthMissingAccessToken(t *testing.T) {
	a, srv := newTestEpicAdapterForAuth(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(epicTokenResponse{
			RefreshToken: "some-refresh-token",
			ExpiresIn:    7200,
		})
	})
	defer srv.Close()

	if err := a.refreshAccessToken(); err == nil {
		t.Fatal("expected error when access_token is empty")
	}
}

func TestEpicGetLibrary_NoAccessToken(t *testing.T) {
	a := &EpicAdapter{}
	_, err := a.GetLibrary()
	if err == nil {
		t.Fatal("expected error when access token is empty")
	}
}

func TestEpicGetLibrary_HTTPError(t *testing.T) {
	a, srv := newTestEpicAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})
	defer srv.Close()

	_, err := a.GetLibrary()
	if err == nil {
		t.Fatal("expected error on non-200 response")
	}
}

func TestEpicGetLibrary_ReauthOn401(t *testing.T) {
	callCount := 0
	a, srv := newTestEpicAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		callCount++
		switch {
		case strings.Contains(r.URL.Path, "/account/api/oauth/token"):
			_ = json.NewEncoder(w).Encode(epicTokenResponse{
				AccessToken:  "reauth-token",
				RefreshToken: "new-refresh",
				ExpiresIn:    7200,
				AccountID:    "acct123",
				DisplayName:  "TestUser",
			})

		case strings.Contains(r.URL.Path, "/library/api/public/items"):
			if callCount == 1 {
				// First call — no access token in query (test adapter has accessToken
				// but actual request will have it in header)
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			// Second call after re-auth — succeed
			_ = json.NewEncoder(w).Encode(epicLibraryResponse{
				Records: []struct {
					AppName       string `json:"appName"`
					CatalogItemID string `json:"catalogItemId"`
					Namespace     string `json:"namespace"`
					ProductID     string `json:"productId"`
					SandboxName   string `json:"sandboxName"`
				}{
					{AppName: "Fortnite", CatalogItemID: "fn-id", Namespace: "fn-ns", ProductID: "fn-pid", SandboxName: "live"},
				},
				ResponseMetadata: nil,
			})

		case strings.Contains(r.URL.Path, "/catalog/api/shared/namespace/"):
			_ = json.NewEncoder(w).Encode(map[string]catalogItemInfo{
				"fn-id": {Title: "Fortnite"},
			})

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
	defer srv.Close()

	games, err := a.GetLibrary()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(games) != 1 {
		t.Fatalf("expected 1 game, got %d", len(games))
	}
	if a.accessToken != "reauth-token" {
		t.Errorf("expected access token to be updated to reauth-token, got %s", a.accessToken)
	}
}

func TestEpicGetLibrary_ParsesGamesAndNames(t *testing.T) {
	a, srv := newTestEpicAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/library/api/public/items"):
			_ = json.NewEncoder(w).Encode(epicLibraryResponse{
				Records: []struct {
					AppName       string `json:"appName"`
					CatalogItemID string `json:"catalogItemId"`
					Namespace     string `json:"namespace"`
					ProductID     string `json:"productId"`
					SandboxName   string `json:"sandboxName"`
				}{
					{AppName: "Kinglet", CatalogItemID: "civ6-id", Namespace: "civ6-ns", ProductID: "civ6-pid", SandboxName: "live"},
					{AppName: "Fortnite", CatalogItemID: "fn-id", Namespace: "fn-ns", ProductID: "fn-pid", SandboxName: "live"},
				},
				ResponseMetadata: nil,
			})

		case strings.Contains(r.URL.Path, "/catalog/api/shared/namespace/"):
			_ = json.NewEncoder(w).Encode(map[string]catalogItemInfo{
				"civ6-id": {Title: "Sid Meier's Civilization VI"},
				"fn-id":   {Title: "Fortnite"},
			})

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
	defer srv.Close()

	games, err := a.GetLibrary()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(games) != 2 {
		t.Fatalf("expected 2 games, got %d", len(games))
	}
	if games[0].ID != "civ6-ns:civ6-id" || games[0].Name != "Sid Meier's Civilization VI" {
		t.Errorf("unexpected first game: %+v", games[0])
	}
	if games[1].ID != "fn-ns:fn-id" || games[1].Name != "Fortnite" {
		t.Errorf("unexpected second game: %+v", games[1])
	}
}

func TestEpicGetLibrary_CatalogFallbackToAppName(t *testing.T) {
	a, srv := newTestEpicAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/library/api/public/items"):
			_ = json.NewEncoder(w).Encode(epicLibraryResponse{
				Records: []struct {
					AppName       string `json:"appName"`
					CatalogItemID string `json:"catalogItemId"`
					Namespace     string `json:"namespace"`
					ProductID     string `json:"productId"`
					SandboxName   string `json:"sandboxName"`
				}{
					{AppName: "SomeCodename", CatalogItemID: "item-1", Namespace: "ns1", ProductID: "p1", SandboxName: "live"},
				},
				ResponseMetadata: nil,
			})

		case strings.Contains(r.URL.Path, "/catalog/api/shared/namespace/"):
			// Return empty — no title found
			_ = json.NewEncoder(w).Encode(map[string]catalogItemInfo{})

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
	defer srv.Close()

	games, err := a.GetLibrary()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(games) != 1 {
		t.Fatalf("expected 1 game, got %d", len(games))
	}
	if games[0].Name != "SomeCodename" {
		t.Errorf("expected fallback to app name SomeCodename, got %s", games[0].Name)
	}
}

func TestEpicGetLibrary_DeduplicatesByNamespaceAndID(t *testing.T) {
	a, srv := newTestEpicAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/library/api/public/items"):
			_ = json.NewEncoder(w).Encode(epicLibraryResponse{
				Records: []struct {
					AppName       string `json:"appName"`
					CatalogItemID string `json:"catalogItemId"`
					Namespace     string `json:"namespace"`
					ProductID     string `json:"productId"`
					SandboxName   string `json:"sandboxName"`
				}{
					{AppName: "Fortnite", CatalogItemID: "fn-id", Namespace: "fn-ns", ProductID: "fn-pid", SandboxName: "live"},
					{AppName: "Fortnite", CatalogItemID: "fn-id", Namespace: "fn-ns", ProductID: "fn-pid", SandboxName: "live"},
					{AppName: "FN", CatalogItemID: "fn-id", Namespace: "fn-ns", ProductID: "fn-pid", SandboxName: "live"},
				},
				ResponseMetadata: nil,
			})

		case strings.Contains(r.URL.Path, "/catalog/api/shared/namespace/"):
			_ = json.NewEncoder(w).Encode(map[string]catalogItemInfo{
				"fn-id": {Title: "Fortnite"},
			})

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
	defer srv.Close()

	games, err := a.GetLibrary()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(games) != 1 {
		t.Fatalf("expected 1 game after dedup, got %d", len(games))
	}
	if games[0].Name != "Fortnite" {
		t.Errorf("expected Fortnite, got %s", games[0].Name)
	}
}

func TestEpicGetLibrary_EmptyLibrary(t *testing.T) {
	a, srv := newTestEpicAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(epicLibraryResponse{
			Records: []struct {
				AppName       string `json:"appName"`
				CatalogItemID string `json:"catalogItemId"`
				Namespace     string `json:"namespace"`
				ProductID     string `json:"productId"`
				SandboxName   string `json:"sandboxName"`
			}{},
			ResponseMetadata: nil,
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

func TestEpicGetLibrary_Pagination(t *testing.T) {
	callCount := 0
	a, srv := newTestEpicAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/library/api/public/items"):
			callCount++
			if callCount == 1 {
				_ = json.NewEncoder(w).Encode(epicLibraryResponse{
					Records: []struct {
						AppName       string `json:"appName"`
						CatalogItemID string `json:"catalogItemId"`
						Namespace     string `json:"namespace"`
						ProductID     string `json:"productId"`
						SandboxName   string `json:"sandboxName"`
					}{
						{AppName: "GameA", CatalogItemID: "a-id", Namespace: "a-ns", ProductID: "a-pid", SandboxName: "live"},
					},
					ResponseMetadata: &struct {
						NextCursor string `json:"nextCursor"`
					}{NextCursor: "cursor-2"},
				})
			} else {
				_ = json.NewEncoder(w).Encode(epicLibraryResponse{
					Records: []struct {
						AppName       string `json:"appName"`
						CatalogItemID string `json:"catalogItemId"`
						Namespace     string `json:"namespace"`
						ProductID     string `json:"productId"`
						SandboxName   string `json:"sandboxName"`
					}{
						{AppName: "GameB", CatalogItemID: "b-id", Namespace: "b-ns", ProductID: "b-pid", SandboxName: "live"},
					},
					ResponseMetadata: nil,
				})
			}

		case strings.Contains(r.URL.Path, "/catalog/api/shared/namespace/"):
			_ = json.NewEncoder(w).Encode(map[string]catalogItemInfo{
				"a-id": {Title: "Game A"},
				"b-id": {Title: "Game B"},
			})

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
	defer srv.Close()

	games, err := a.GetLibrary()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(games) != 2 {
		t.Fatalf("expected 2 games across 2 pages, got %d", len(games))
	}
}
