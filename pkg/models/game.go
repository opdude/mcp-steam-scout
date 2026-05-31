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
	Rank              int    `json:"rank,omitempty"`
}

// MergedGame represents a game that has been normalized and merged across
// multiple platform libraries. TotalPlaytimeMinutes is the sum of playtime
// across all entries.
type MergedGame struct {
	NormalizedName       string   `json:"normalizedName"`
	DisplayName          string   `json:"displayName"`
	Entries              []Game   `json:"entries"`
	TotalPlaytimeMinutes int      `json:"totalPlaytimeMinutes"`
	Platforms            []string `json:"platforms"`
}
