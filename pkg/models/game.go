package models

// Game represents a game returned by a gaming platform API.
type Game struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	PlaytimeMinutes int    `json:"playtimeMinutes,omitempty"`
}
