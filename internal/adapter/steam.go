package adapter

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/opdude/mcp-steam-scout/pkg/models"
)

// SteamAdapter implements the interface for Steam.
type SteamAdapter struct {
	APIKey         string
	DefaultSteamID string
	Client         *http.Client
}

// NewSteamAdapter creates a new SteamAdapter.
func NewSteamAdapter(apiKey, defaultSteamID string) *SteamAdapter {
	return &SteamAdapter{
		APIKey:         apiKey,
		DefaultSteamID: defaultSteamID,
		Client:         &http.Client{},
	}
}

// GetLibrary fetches the user's owned games from the Steam Web API using the DefaultSteamID.
func (s *SteamAdapter) GetLibrary() ([]models.Game, error) {
	if s.APIKey == "" {
		return nil, fmt.Errorf("steam API key is not configured")
	}

	if s.DefaultSteamID == "" {
		return nil, fmt.Errorf("no default user ID configured")
	}

	url := fmt.Sprintf("http://api.steampowered.com/IPlayerService/GetOwnedGames/v0001/?steamid=%s&key=%s&format=json&include_appinfo=true", s.DefaultSteamID, s.APIKey)

	resp, err := s.Client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch steam library: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("steam api returned error: %s", resp.Status)
	}

	var result struct {
		Response struct {
			Games []struct {
				AppID           int    `json:"appid"`
				Name            string `json:"name"`
				PlaytimeForever int    `json:"playtime_forever"`
			} `json:"games"`
		} `json:"response"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode steam response: %w", err)
	}

	games := make([]models.Game, 0, len(result.Response.Games))
	for _, g := range result.Response.Games {
		games = append(games, models.Game{
			ID:              fmt.Sprintf("%d", g.AppID),
			Name:            g.Name,
			PlaytimeMinutes: g.PlaytimeForever,
		})
	}

	return games, nil
}

// ResolveVanityURL resolves a Steam vanity URL (username) to a numeric Steam ID.
func (s *SteamAdapter) ResolveVanityURL(vanityURL string) (string, error) {
	if s.APIKey == "" {
		return "", fmt.Errorf("steam API key is not configured")
	}

	url := fmt.Sprintf("http://api.steampowered.com/ISteamUser/ResolveVanityURL/v0001/?key=%s&vanityurl=%s", s.APIKey, vanityURL)

	resp, err := s.Client.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to resolve vanity URL: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("steam api returned error: %s", resp.Status)
	}

	var result struct {
		Response struct {
			SteamID string `json:"steamid"`
			Success int    `json:"success"`
			Message string `json:"message"`
		} `json:"response"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode steam response: %w", err)
	}

	if result.Response.Success != 1 {
		return "", fmt.Errorf("could not resolve vanity URL %q: %s", vanityURL, result.Response.Message)
	}

	return result.Response.SteamID, nil
}
