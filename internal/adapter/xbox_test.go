package adapter

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// newTestXboxAdapter creates an XboxAdapter with pre-set tokens and a mock HTTP transport.
func newTestXboxAdapter(t *testing.T, handler http.HandlerFunc) (*XboxAdapter, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	deviceKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	a := &XboxAdapter{
		refreshToken: "test-refresh-token",
		signerKey:    key,
		deviceToken: &xblDeviceToken{
			Token:    "test-device-token",
			NotAfter: time.Now().Add(1 * time.Hour),
			proofKey: deviceKey,
		},
		oauthToken: &oauthTokenHolder{
			accessToken:  "test-access-token",
			refreshToken: "test-refresh-token",
			expiry:       time.Now().Add(1 * time.Hour),
		},
	}

	orig := http.DefaultTransport
	http.DefaultTransport = &rewriteTransport{base: srv.URL, inner: srv.Client().Transport}
	t.Cleanup(func() { http.DefaultTransport = orig })

	return a, srv
}

func TestXboxGetLibrary_NoRefreshToken(t *testing.T) {
	a := &XboxAdapter{}
	_, err := a.GetLibrary(context.Background())
	if err == nil {
		t.Fatal("expected error when refresh token is empty")
	}
}

func TestXboxGetLibrary_HTTPError(t *testing.T) {
	a, srv := newTestXboxAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		// Ensure SISU returns non-200 (the first auth call made)
		switch {
		case strings.HasSuffix(r.URL.Path, "/authorize"):
			w.WriteHeader(http.StatusUnauthorized)
			w.Header().Set("x-err", "auth_failed")
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
	defer srv.Close()

	_, err := a.GetLibrary(context.Background())
	if err == nil {
		t.Fatal("expected error on SISU auth failure")
	}
}

func TestXboxGetLibrary_SISUAuthError(t *testing.T) {
	a, srv := newTestXboxAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/authorize") {
			// SISU returns 200 but with no UserToken (age gate scenario)
			w.Header().Set("x-err", "age_gate_required")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"UserToken":null}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer srv.Close()

	_, err := a.GetLibrary(context.Background())
	if err == nil {
		t.Fatal("expected error when SISU returns no UserToken")
	}
}

func TestXboxGetLibrary_XSTSError(t *testing.T) {
	a, srv := newTestXboxAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/authorize"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"UserToken": map[string]any{
					"Token": "sisu-token",
					"DisplayClaims": map[string]any{
						"xui": []map[string]any{
							{"uhs": "test-uhs"},
						},
					},
				},
			})
		case strings.HasSuffix(r.URL.Path, "/xsts/authorize"):
			w.WriteHeader(http.StatusUnauthorized)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
	defer srv.Close()

	_, err := a.GetLibrary(context.Background())
	if err == nil {
		t.Fatal("expected error on XSTS auth failure")
	}
}

func TestXboxGetLibrary_ParsesGamesAndPlaytime(t *testing.T) {
	a, srv := newTestXboxAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/xsts/authorize"):
			w.Header().Set("Date", time.Now().Format(time.RFC1123))
			_ = json.NewEncoder(w).Encode(map[string]any{
				"Token": "xsts-token",
				"DisplayClaims": map[string]any{
					"xui": []map[string]any{
						{
							"uhs": "test-uhs",
							"xid": "test-xuid",
							"gtg": "TestGamer",
						},
					},
				},
			})

		case strings.HasSuffix(r.URL.Path, "/authorize"):
			w.Header().Set("Date", time.Now().Format(time.RFC1123))
			_ = json.NewEncoder(w).Encode(map[string]any{
				"UserToken": map[string]any{
					"Token": "sisu-token",
					"DisplayClaims": map[string]any{
						"xui": []map[string]any{
							{"uhs": "test-uhs"},
						},
					},
				},
			})

		case strings.Contains(r.URL.Path, "/users/xuid("):
			// titlehub response
			_ = json.NewEncoder(w).Encode(xboxTitleResponse{
				Titles: []struct {
					TitleID      string   `json:"titleId"`
					Name         string   `json:"name"`
					Type         string   `json:"type"`
					Devices      []string `json:"devices"`
					TitleHistory struct {
						LastTimePlayed string `json:"lastTimePlayed"`
					} `json:"titleHistory"`
				}{
					{
						TitleID: "123456789",
						Name:    "Test Game One",
						Type:    "Game",
						Devices: []string{"Win32"},
						TitleHistory: struct {
							LastTimePlayed string `json:"lastTimePlayed"`
						}{LastTimePlayed: "2024-01-01T00:00:00Z"},
					},
					{
						TitleID: "987654321",
						Name:    "Test Game Two",
						Type:    "Game",
						Devices: []string{"XboxConsole"},
						TitleHistory: struct {
							LastTimePlayed string `json:"lastTimePlayed"`
						}{LastTimePlayed: "2024-06-15T12:30:00Z"},
					},
					{
						TitleID: "555555555",
						Name:    "An App",
						Type:    "App",
						Devices: []string{"Win32"},
					},
				},
			})

		case strings.HasSuffix(r.URL.Path, "/batch"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"statlistscollection": []map[string]any{
					{
						"stats": []map[string]any{
							{"titleid": "123456789", "name": "MinutesPlayed", "value": "450"},
							{"titleid": "987654321", "name": "MinutesPlayed", "value": "1200"},
						},
					},
				},
			})

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
	defer srv.Close()

	games, err := a.GetLibrary(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(games) != 2 {
		t.Fatalf("expected 2 games (filtering out non-Game types), got %d", len(games))
	}
	if games[0].ID != "123456789" || games[0].Name != "Test Game One" {
		t.Errorf("unexpected first game: %+v", games[0])
	}
	if games[0].PlaytimeMinutes != 450 {
		t.Errorf("expected playtime 450 for game one, got %d", games[0].PlaytimeMinutes)
	}
	if games[1].ID != "987654321" || games[1].Name != "Test Game Two" {
		t.Errorf("unexpected second game: %+v", games[1])
	}
	if games[1].PlaytimeMinutes != 1200 {
		t.Errorf("expected playtime 1200 for game two, got %d", games[1].PlaytimeMinutes)
	}
}

func TestXboxGetLibrary_EmptyLibrary(t *testing.T) {
	a, srv := newTestXboxAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/xsts/authorize"):
			w.Header().Set("Date", time.Now().Format(time.RFC1123))
			_ = json.NewEncoder(w).Encode(map[string]any{
				"Token": "xsts-token",
				"DisplayClaims": map[string]any{
					"xui": []map[string]any{
						{
							"uhs": "test-uhs",
							"xid": "test-xuid",
							"gtg": "TestGamer",
						},
					},
				},
			})

		case strings.HasSuffix(r.URL.Path, "/authorize"):
			w.Header().Set("Date", time.Now().Format(time.RFC1123))
			_ = json.NewEncoder(w).Encode(map[string]any{
				"UserToken": map[string]any{
					"Token": "sisu-token",
					"DisplayClaims": map[string]any{
						"xui": []map[string]any{
							{"uhs": "test-uhs"},
						},
					},
				},
			})

		case strings.Contains(r.URL.Path, "/users/xuid("):
			_ = json.NewEncoder(w).Encode(xboxTitleResponse{
				Titles: []struct {
					TitleID      string   `json:"titleId"`
					Name         string   `json:"name"`
					Type         string   `json:"type"`
					Devices      []string `json:"devices"`
					TitleHistory struct {
						LastTimePlayed string `json:"lastTimePlayed"`
					} `json:"titleHistory"`
				}{},
			})

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
	defer srv.Close()

	games, err := a.GetLibrary(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(games) != 0 {
		t.Errorf("expected empty library, got %d games", len(games))
	}
}

func TestXboxGetLibrary_BatchErrorIsNonFatal(t *testing.T) {
	a, srv := newTestXboxAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/xsts/authorize"):
			w.Header().Set("Date", time.Now().Format(time.RFC1123))
			_ = json.NewEncoder(w).Encode(map[string]any{
				"Token": "xsts-token",
				"DisplayClaims": map[string]any{
					"xui": []map[string]any{
						{
							"uhs": "test-uhs",
							"xid": "test-xuid",
							"gtg": "TestGamer",
						},
					},
				},
			})

		case strings.HasSuffix(r.URL.Path, "/authorize"):
			w.Header().Set("Date", time.Now().Format(time.RFC1123))
			_ = json.NewEncoder(w).Encode(map[string]any{
				"UserToken": map[string]any{
					"Token": "sisu-token",
					"DisplayClaims": map[string]any{
						"xui": []map[string]any{
							{"uhs": "test-uhs"},
						},
					},
				},
			})

		case strings.Contains(r.URL.Path, "/users/xuid("):
			_ = json.NewEncoder(w).Encode(xboxTitleResponse{
				Titles: []struct {
					TitleID      string   `json:"titleId"`
					Name         string   `json:"name"`
					Type         string   `json:"type"`
					Devices      []string `json:"devices"`
					TitleHistory struct {
						LastTimePlayed string `json:"lastTimePlayed"`
					} `json:"titleHistory"`
				}{
					{
						TitleID: "123456789",
						Name:    "Test Game One",
						Type:    "Game",
						Devices: []string{"Win32"},
						TitleHistory: struct {
							LastTimePlayed string `json:"lastTimePlayed"`
						}{LastTimePlayed: "2024-01-01T00:00:00Z"},
					},
				},
			})

		case strings.HasSuffix(r.URL.Path, "/batch"):
			w.WriteHeader(http.StatusInternalServerError)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
	defer srv.Close()

	games, err := a.GetLibrary(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(games) != 1 {
		t.Fatalf("expected 1 game, got %d", len(games))
	}
	// Playtime should be 0 since batch call failed
	if games[0].PlaytimeMinutes != 0 {
		t.Errorf("expected playtime 0 after batch failure, got %d", games[0].PlaytimeMinutes)
	}
}

func TestXboxGetLibrary_TitlehubHTTPError(t *testing.T) {
	a, srv := newTestXboxAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/xsts/authorize"):
			w.Header().Set("Date", time.Now().Format(time.RFC1123))
			_ = json.NewEncoder(w).Encode(map[string]any{
				"Token": "xsts-token",
				"DisplayClaims": map[string]any{
					"xui": []map[string]any{
						{
							"uhs": "test-uhs",
							"xid": "test-xuid",
							"gtg": "TestGamer",
						},
					},
				},
			})

		case strings.HasSuffix(r.URL.Path, "/authorize"):
			w.Header().Set("Date", time.Now().Format(time.RFC1123))
			_ = json.NewEncoder(w).Encode(map[string]any{
				"UserToken": map[string]any{
					"Token": "sisu-token",
					"DisplayClaims": map[string]any{
						"xui": []map[string]any{
							{"uhs": "test-uhs"},
						},
					},
				},
			})

		case strings.Contains(r.URL.Path, "/users/xuid("):
			w.WriteHeader(http.StatusBadRequest)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
	defer srv.Close()

	_, err := a.GetLibrary(context.Background())
	if err == nil {
		t.Fatal("expected error on titlehub HTTP error")
	}
}
