package scraper

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/opdude/mcp-steam-scout/pkg/models"
)

const (
	gogPopularURL = "https://www.gog.com/games/ajax/filtered?mediaType=game&sort=popularity&limit=10"
	gogNewURL     = "https://www.gog.com/games/ajax/filtered?mediaType=game&sort=date&limit=24"
)

type gogTrendingResponse struct {
	Products []struct {
		ID    int    `json:"id"`
		Title string `json:"title"`
	} `json:"products"`
}

func (s *TrendingScraper) fetchGOGTrending() ([]models.Game, error) {
	popular, _ := s.fetchGOGList(gogPopularURL)
	newReleases, _ := s.fetchGOGList(gogNewURL)

	seen := make(map[string]bool)
	var games []models.Game

	for _, list := range [][]models.Game{popular, newReleases} {
		for _, g := range list {
			if seen[g.ID] {
				continue
			}
			seen[g.ID] = true
			games = append(games, g)
		}
	}

	return games, nil
}

func (s *TrendingScraper) fetchGOGList(url string) ([]models.Game, error) {
	resp, err := s.Client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("gog request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gog store returned %s", resp.Status)
	}

	var result gogTrendingResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode gog response: %w", err)
	}

	games := make([]models.Game, 0, len(result.Products))
	seen := make(map[int]bool)
	for _, p := range result.Products {
		if p.Title == "" || seen[p.ID] {
			continue
		}
		seen[p.ID] = true
		games = append(games, models.Game{
			ID:       strconv.Itoa(p.ID),
			Name:     p.Title,
			Platform: "gog",
		})
	}

	return games, nil
}
