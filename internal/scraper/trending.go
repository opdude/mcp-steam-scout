package scraper

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"sync"
	"time"

	"github.com/opdude/mcp-steam-scout/pkg/models"
)

const trendingURL = "https://store.steampowered.com/search/results/?filter=topsellers&json=1&count=100"

// appIDFromLogo extracts the Steam app ID from a store capsule image URL.
// Example: https://...steam/apps/730/capsule_sm_120.jpg → "730"
var appIDFromLogo = regexp.MustCompile(`/apps/(\d+)/`)

// TrendingScraper fetches trending games from external sources.
type TrendingScraper struct {
	Client *http.Client

	mu          sync.Mutex
	cacheGames  []models.Game
	cacheExpiry time.Time
}

// NewTrendingScraper creates a new TrendingScraper.
func NewTrendingScraper() *TrendingScraper {
	return &TrendingScraper{
		Client: &http.Client{},
	}
}

// GetTrendingGames fetches trending games from Steam, GOG, and Epic Games Store.
// Results are cached for 30 minutes.
func (s *TrendingScraper) GetTrendingGames() ([]models.Game, error) {
	s.mu.Lock()
	if s.cacheGames != nil && time.Now().Before(s.cacheExpiry) {
		games := s.cacheGames
		s.mu.Unlock()
		return games, nil
	}
	s.mu.Unlock()

	steamGames, _ := s.fetchSteamTrending()
	gogGames, _ := s.fetchGOGTrending()
	epicGames, _ := s.fetchEpicTrending()

	total := len(steamGames) + len(gogGames) + len(epicGames)
	allGames := make([]models.Game, 0, total)
	allGames = append(allGames, steamGames...)
	allGames = append(allGames, gogGames...)
	allGames = append(allGames, epicGames...)

	s.mu.Lock()
	s.cacheGames = allGames
	s.cacheExpiry = time.Now().Add(30 * time.Minute)
	s.mu.Unlock()

	return allGames, nil
}

func (s *TrendingScraper) fetchSteamTrending() ([]models.Game, error) {
	resp, err := s.Client.Get(trendingURL)
	if err != nil {
		return nil, fmt.Errorf("steam request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("steam store returned %s", resp.Status)
	}

	var result struct {
		Items []struct {
			Name string `json:"name"`
			Logo string `json:"logo"`
		} `json:"items"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode steam response: %w", err)
	}

	seen := make(map[string]bool)
	var games []models.Game

	for _, item := range result.Items {
		if item.Name == "" {
			continue
		}
		m := appIDFromLogo.FindStringSubmatch(item.Logo)
		if m == nil {
			continue
		}
		id := m[1]
		if seen[id] {
			continue
		}
		seen[id] = true
		games = append(games, models.Game{
			ID:       id,
			Name:     item.Name,
			Platform: "steam",
		})
	}

	return games, nil
}
