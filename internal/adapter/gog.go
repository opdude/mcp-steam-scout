package adapter

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/opdude/mcp-steam-scout/pkg/models"
)

const (
	gogClientID     = "46899977096215655"
	gogClientSecret = "9d85c43b1482497dbbce61f6e4aa173a433796eeae2ca8c5f6129f2dc4de46d9"
	gogTokenURL     = "https://auth.gog.com/token"
	gogEmbedAPI     = "https://embed.gog.com"
	gogWWWOrigin    = "https://www.gog.com"
)

type gogTokenResponse struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	UserID       string `json:"user_id"`
	SessionID    string `json:"session_id"`
}

type gogUserDataResponse struct {
	UserID   string `json:"userId"`
	Username string `json:"username"`
}

type gogLibraryResponse struct {
	Page            int `json:"page"`
	TotalPages      int `json:"totalPages"`
	TotalProducts   int `json:"totalProducts"`
	ProductsPerPage int `json:"productsPerPage"`
	Products        []struct {
		ID    int    `json:"id"`
		Title string `json:"title"`
	} `json:"products"`
}

type gogStatsResponse struct {
	Page  int `json:"page"`
	Limit int `json:"limit"`
	Pages int `json:"pages"`
	Total int `json:"total"`
	Links struct {
		Next *struct {
			Href string `json:"href"`
		} `json:"next"`
	} `json:"_links"`
	Embedded struct {
		Items []gogStatsItem `json:"items"`
	} `json:"_embedded"`
}

type gogStatsItem struct {
	Game struct {
		ID    string `json:"id"`
		Title string `json:"title"`
	} `json:"game"`
	Stats json.RawMessage `json:"stats"`
}

type gogStatValue struct {
	Playtime    int    `json:"playtime"`
	LastSession string `json:"lastSession"`
}

// GOGAdapter implements GOG library access via the website API.
type GOGAdapter struct {
	refreshToken string
	accessToken  string
	username     string
	gogAlCookie  string
	client       *http.Client

	mu          sync.Mutex
	cacheGames  []models.Game
	cacheExpiry time.Time
}

// NewGOGAdapter creates a GOGAdapter and exchanges the refresh token for an access token.
// gogAlCookie is optional — if set, it's used to fetch playtime from the GOG website API.
// Set it to the value of the gog-al cookie from your browser after logging into gog.com.
func NewGOGAdapter(refreshToken, gogAlCookie string) (*GOGAdapter, error) {
	a := &GOGAdapter{
		refreshToken: refreshToken,
		gogAlCookie:  gogAlCookie,
		client:       &http.Client{},
	}
	if err := a.refreshOAuthToken(); err != nil {
		return nil, fmt.Errorf("gog authentication failed: %w", err)
	}
	if err := a.fetchUserData(); err != nil {
		return nil, fmt.Errorf("gog user data fetch failed: %w", err)
	}
	return a, nil
}

func (g *GOGAdapter) refreshOAuthToken() error {
	form := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {g.refreshToken},
		"client_id":     {gogClientID},
		"client_secret": {gogClientSecret},
	}

	req, err := http.NewRequest("GET", gogTokenURL+"?"+form.Encode(), nil)
	if err != nil {
		return fmt.Errorf("failed to create token request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := g.client.Do(req)
	if err != nil {
		return fmt.Errorf("token request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("gog token API returned %s: %s", resp.Status, string(body))
	}

	var tokenResp gogTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return fmt.Errorf("failed to decode token response: %w", err)
	}

	if tokenResp.AccessToken == "" {
		return fmt.Errorf("gog token response missing access_token")
	}

	g.accessToken = tokenResp.AccessToken
	if tokenResp.RefreshToken != "" {
		g.refreshToken = tokenResp.RefreshToken
	}
	return nil
}

func (g *GOGAdapter) fetchUserData() error {
	req, err := http.NewRequest("GET", gogEmbedAPI+"/userData.json", nil)
	if err != nil {
		return fmt.Errorf("failed to create user data request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+g.accessToken)

	resp, err := g.client.Do(req)
	if err != nil {
		return fmt.Errorf("user data request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("gog user data returned %s", resp.Status)
	}

	var userResp gogUserDataResponse
	if err := json.NewDecoder(resp.Body).Decode(&userResp); err != nil {
		return fmt.Errorf("failed to decode user data: %w", err)
	}

	g.username = userResp.Username
	return nil
}

// GetLibrary fetches the user's GOG library with playtime data.
// Results are cached for 5 minutes.
func (g *GOGAdapter) GetLibrary() ([]models.Game, error) {
	if g.accessToken == "" {
		return nil, fmt.Errorf("access token not set; use NewGOGAdapter")
	}
	if g.client == nil {
		g.client = &http.Client{}
	}

	g.mu.Lock()
	if g.cacheGames != nil && time.Now().Before(g.cacheExpiry) {
		games := g.cacheGames
		g.mu.Unlock()
		return games, nil
	}
	g.mu.Unlock()

	games, err := g.fetchLibrary()
	if err != nil {
		return nil, err
	}

	stats, err := g.fetchStats()
	if err != nil {
		log.Printf("gog stats fetch failed (playtime will be unavailable): %v", err)
	}
	if stats != nil {
		for i := range games {
			if pt, ok := stats[games[i].ID]; ok {
				games[i].PlaytimeMinutes = pt
			}
		}
	}

	g.mu.Lock()
	g.cacheGames = games
	g.cacheExpiry = time.Now().Add(5 * time.Minute)
	g.mu.Unlock()

	return games, nil
}

func (g *GOGAdapter) fetchLibrary() ([]models.Game, error) {
	var allGames []models.Game
	page := 1

	for {
		u := gogEmbedAPI + "/account/getFilteredProducts?mediaType=1&page=" + strconv.Itoa(page)
		req, err := http.NewRequest("GET", u, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create library request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+g.accessToken)

		resp, err := g.client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("library request failed: %w", err)
		}

		body, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to read library response: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			if resp.StatusCode == http.StatusUnauthorized {
				if err := g.refreshOAuthToken(); err != nil {
					return nil, fmt.Errorf("gog re-auth failed: %w", err)
				}
				continue
			}
			return nil, fmt.Errorf("gog library API returned %s: %s", resp.Status, string(body))
		}

		var libResp gogLibraryResponse
		if err := json.Unmarshal(body, &libResp); err != nil {
			return nil, fmt.Errorf("failed to decode library response: %w", err)
		}

		for _, p := range libResp.Products {
			allGames = append(allGames, models.Game{
				ID:       strconv.Itoa(p.ID),
				Name:     p.Title,
				Platform: "gog",
			})
		}

		if page >= libResp.TotalPages {
			break
		}
		page++
	}

	return allGames, nil
}

func (g *GOGAdapter) fetchStats() (map[string]int, error) {
	if g.gogAlCookie == "" {
		return nil, nil
	}

	stats := make(map[string]int)
	page := 1

	for {
		u := fmt.Sprintf("%s/u/%s/games/stats?sort=recent_playtime&order=desc&page=%d", gogWWWOrigin, g.username, page)
		req, err := http.NewRequest("GET", u, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create stats request: %w", err)
		}
		req.Header.Set("Cookie", "gog-al="+g.gogAlCookie)
		req.Header.Set("Accept", "application/json")

		resp, err := g.client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("stats request failed: %w", err)
		}

		body, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to read stats response: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			if resp.StatusCode == http.StatusUnauthorized {
				if err := g.refreshOAuthToken(); err != nil {
					return nil, fmt.Errorf("gog re-auth failed: %w", err)
				}
				continue
			}
			return nil, fmt.Errorf("gog stats API returned %s: %s", resp.Status, string(body))
		}

		var statsResp gogStatsResponse
		if err := json.Unmarshal(body, &statsResp); err != nil {
			return nil, fmt.Errorf("failed to decode stats response: %w", err)
		}

		for _, item := range statsResp.Embedded.Items {
			if len(item.Stats) == 0 || string(item.Stats) == "[]" {
				continue
			}
			var statMap map[string]gogStatValue
			if err := json.Unmarshal(item.Stats, &statMap); err != nil {
				continue
			}
			for _, sv := range statMap {
				stats[item.Game.ID] = sv.Playtime
				break
			}
		}

		if statsResp.Links.Next == nil {
			break
		}
		page++
	}

	return stats, nil
}
