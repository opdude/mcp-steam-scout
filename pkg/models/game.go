package models

// Game represents a game returned by a gaming platform API.
type Game struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	Platform          string `json:"platform"`
	PlaytimeMinutes   int    `json:"playtimeMinutes,omitempty"`
	PriceAmount       string `json:"priceAmount,omitempty"`
	PriceBaseAmount   string `json:"priceBaseAmount,omitempty"`
	PriceIsDiscounted bool   `json:"priceIsDiscounted,omitempty"`
	PriceCurrency     string `json:"priceCurrency,omitempty"`
}
