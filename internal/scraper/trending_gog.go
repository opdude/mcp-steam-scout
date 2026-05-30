package scraper

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/opdude/mcp-steam-scout/pkg/models"
)

const gogTrendingURL = "https://www.gog.com/games/ajax/filtered?mediaType=game&sort=popularity&limit=20"

type gogTrendingResponse struct {
	Products []struct {
		ID    int    `json:"id"`
		Title string `json:"title"`
	} `json:"products"`
}

func (s *TrendingScraper) fetchGOGTrending() ([]models.Game, error) {
	resp, err := s.Client.Get(gogTrendingURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch GOG trending: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GOG store returned %s", resp.Status)
	}

	var result gogTrendingResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode GOG response: %w", err)
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
