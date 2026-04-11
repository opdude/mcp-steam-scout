package scraper

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"

	"github.com/opdude/mcp-steam-scout/pkg/models"
)

const trendingURL = "https://store.steampowered.com/search/results/?filter=topsellers&json=1&count=100"

// appIDFromLogo extracts the Steam app ID from a store capsule image URL.
// Example: https://...steam/apps/730/capsule_sm_120.jpg → "730"
var appIDFromLogo = regexp.MustCompile(`/apps/(\d+)/`)

// TrendingScraper fetches trending games from external sources.
type TrendingScraper struct {
	Client *http.Client
}

// NewTrendingScraper creates a new TrendingScraper.
func NewTrendingScraper() *TrendingScraper {
	return &TrendingScraper{
		Client: &http.Client{},
	}
}

// GetTrendingGames fetches the top 100 trending games from the Steam store top sellers list.
func (s *TrendingScraper) GetTrendingGames() ([]models.Game, error) {
	resp, err := s.Client.Get(trendingURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch trending games: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("steam store returned error: %s", resp.Status)
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
			ID:   id,
			Name: item.Name,
		})
	}

	return games, nil
}
