package scraper

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/opdude/mcp-steam-scout/pkg/models"
)

const epicTrendingQuery = `{"query":"{Catalog{searchStore(count:20 sortBy:POPULARITY){elements{title id productSlug offerType}}}}"}`

type epicTrendingResponse struct {
	Data struct {
		Catalog struct {
			SearchStore struct {
				Elements []struct {
					Title       string `json:"title"`
					ID          string `json:"id"`
					ProductSlug string `json:"productSlug"`
					OfferType   string `json:"offerType"`
				} `json:"elements"`
			} `json:"searchStore"`
		} `json:"Catalog"`
	} `json:"data"`
}

func (s *TrendingScraper) fetchEpicTrending() ([]models.Game, error) {
	req, err := s.Client.Post("https://store.epicgames.com/graphql", "application/json", strings.NewReader(epicTrendingQuery))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Epic trending: %w", err)
	}
	defer func() { _ = req.Body.Close() }()

	if req.StatusCode != 200 {
		return nil, fmt.Errorf("epic store returned %s", req.Status)
	}

	var result epicTrendingResponse
	if err := json.NewDecoder(req.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode Epic response: %w", err)
	}

	games := make([]models.Game, 0, len(result.Data.Catalog.SearchStore.Elements))
	seen := make(map[string]bool)
	for _, e := range result.Data.Catalog.SearchStore.Elements {
		if e.Title == "" || seen[e.ID] {
			continue
		}
		if e.OfferType != "BASE_GAME" && e.OfferType != "BUNDLE" {
			continue
		}
		seen[e.ID] = true
		games = append(games, models.Game{
			ID:       e.ID,
			Name:     e.Title,
			Platform: "epic",
		})
	}

	return games, nil
}
