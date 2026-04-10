package models

// Game represents a game returned by the Steam API.
type Game struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	PlaytimeMinutes int    `json:"playtimeMinutes,omitempty"`
}
