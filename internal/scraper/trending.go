package scraper

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/opdude/mcp-steam-scout/pkg/models"
)

const featuredURL = "https://store.steampowered.com/api/featuredcategories"

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

// GetTrendingGames fetches trending games from Steam and GOG.
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

	allGames := make([]models.Game, 0, len(steamGames)+len(gogGames))
	allGames = append(allGames, steamGames...)
	allGames = append(allGames, gogGames...)

	s.mu.Lock()
	s.cacheGames = allGames
	s.cacheExpiry = time.Now().Add(30 * time.Minute)
	s.mu.Unlock()

	return allGames, nil
}

func (s *TrendingScraper) fetchSteamTrending() ([]models.Game, error) {
	resp, err := s.Client.Get(featuredURL)
	if err != nil {
		return nil, fmt.Errorf("steam request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("steam store returned %s", resp.Status)
	}

	var result struct {
		TopSellers *struct {
			Items []struct {
				ID              int    `json:"id"`
				Name            string `json:"name"`
				Discounted      bool   `json:"discounted"`
				DiscountPercent int    `json:"discount_percent"`
				OriginalPrice   int    `json:"original_price"`
				FinalPrice      int    `json:"final_price"`
				Currency        string `json:"currency"`
			} `json:"items"`
		} `json:"top_sellers"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode steam response: %w", err)
	}

	if result.TopSellers == nil {
		return nil, nil
	}

	games := make([]models.Game, 0, len(result.TopSellers.Items))
	for i, item := range result.TopSellers.Items {
		if item.Name == "" {
			continue
		}
		priceAmount := fmt.Sprintf("%d.%02d", item.FinalPrice/100, item.FinalPrice%100)
		priceBaseAmount := fmt.Sprintf("%d.%02d", item.OriginalPrice/100, item.OriginalPrice%100)
		games = append(games, models.Game{
			ID:                strconv.Itoa(item.ID),
			Name:              item.Name,
			Platform:          "steam",
			PriceAmount:       priceAmount,
			PriceBaseAmount:   priceBaseAmount,
			PriceIsDiscounted: item.Discounted,
			PriceCurrency:     item.Currency,
			Rank:              i + 1,
		})
	}

	return games, nil
}
