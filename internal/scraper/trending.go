package scraper

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/opdude/mcp-steam-scout/pkg/models"
)

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

// GetTrendingGames fetches currently trending games from Steam's top sellers.
func (s *TrendingScraper) GetTrendingGames() ([]models.Game, error) {
	resp, err := s.Client.Get("https://store.steampowered.com/api/featuredcategories/")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch trending games: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("steam store returned error: %s", resp.Status)
	}

	var result struct {
		TopSellers struct {
			Items []struct {
				ID   int    `json:"id"`
				Name string `json:"name"`
			} `json:"items"`
		} `json:"top_sellers"`
		NewReleases struct {
			Items []struct {
				ID   int    `json:"id"`
				Name string `json:"name"`
			} `json:"items"`
		} `json:"new_releases"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode steam response: %w", err)
	}

	seen := make(map[int]bool)
	var games []models.Game

	for _, item := range result.TopSellers.Items {
		if item.Name == "" || seen[item.ID] {
			continue
		}
		seen[item.ID] = true
		games = append(games, models.Game{
			ID:   fmt.Sprintf("%d", item.ID),
			Name: item.Name,
		})
	}

	for _, item := range result.NewReleases.Items {
		if item.Name == "" || seen[item.ID] {
			continue
		}
		seen[item.ID] = true
		games = append(games, models.Game{
			ID:   fmt.Sprintf("%d", item.ID),
			Name: item.Name,
		})
	}

	return games, nil
}
