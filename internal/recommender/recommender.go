package recommender

import (
	"sort"

	"github.com/opdude/mcp-steam-scout/pkg/models"
)

type NormalizedGame struct {
	Name                 string        `json:"name"`
	Entries              []models.Game `json:"entries"`
	TotalPlaytimeMinutes int           `json:"totalPlaytimeMinutes"`
}

type TrendingCandidate struct {
	Game        models.Game `json:"game"`
	MatchReason string      `json:"matchReason"`
}

type Recommendation struct {
	Unplayed           []NormalizedGame    `json:"unplayed"`
	Dabbled            []NormalizedGame    `json:"dabbled"`
	TopPlayed          []NormalizedGame    `json:"topPlayed"`
	TrendingOverlap    []NormalizedGame    `json:"trendingOverlap"`
	TrendingCandidates []TrendingCandidate `json:"trendingCandidates"`
	TotalGames         int                 `json:"totalGames"`
	Platforms          []string            `json:"platforms"`
}

type Recommender struct{}

func New() *Recommender {
	return &Recommender{}
}

func (r *Recommender) Recommend(
	steamGames []models.Game,
	psnGames []models.Game,
	xboxGames []models.Game,
	epicGames []models.Game,
	gogGames []models.Game,
	trending []models.Game,
) Recommendation {
	totalCap := len(steamGames) + len(psnGames) + len(xboxGames) + len(epicGames) + len(gogGames)
	allGames := make([]models.Game, 0, totalCap)
	allGames = append(allGames, steamGames...)
	allGames = append(allGames, psnGames...)
	allGames = append(allGames, xboxGames...)
	allGames = append(allGames, epicGames...)
	allGames = append(allGames, gogGames...)

	platforms := collectPlatforms(allGames)
	merged := mergeGames(allGames)

	normalized := make([]NormalizedGame, 0, len(merged))
	for _, g := range merged {
		normalized = append(normalized, g)
	}

	sort.Slice(normalized, func(i, j int) bool {
		return normalized[i].TotalPlaytimeMinutes > normalized[j].TotalPlaytimeMinutes
	})

	var unplayed, dabbled, topPlayed []NormalizedGame
	for _, ng := range normalized {
		switch {
		case ng.TotalPlaytimeMinutes == 0:
			unplayed = append(unplayed, ng)
		case ng.TotalPlaytimeMinutes <= 300:
			dabbled = append(dabbled, ng)
		}
	}
	topPlayed = normalized
	if len(topPlayed) > 20 {
		topPlayed = topPlayed[:20]
	}

	trendingOverlap := findTrendingOverlap(merged, trending)
	trendingCandidates := findTrendingCandidates(normalized, trending)

	return Recommendation{
		Unplayed:           unplayed,
		Dabbled:            dabbled,
		TopPlayed:          topPlayed,
		TrendingOverlap:    trendingOverlap,
		TrendingCandidates: trendingCandidates,
		TotalGames:         len(normalized),
		Platforms:          platforms,
	}
}

func collectPlatforms(games []models.Game) []string {
	seen := make(map[string]bool)
	var platforms []string
	for _, g := range games {
		if !seen[g.Platform] {
			seen[g.Platform] = true
			platforms = append(platforms, g.Platform)
		}
	}
	sort.Strings(platforms)
	return platforms
}

func mergeGames(games []models.Game) map[string]NormalizedGame {
	normalized := make(map[string]NormalizedGame)
	for _, g := range games {
		key := normalizeName(g.Name)
		existing, ok := normalized[key]
		if !ok {
			existing = NormalizedGame{Name: g.Name}
		}
		if g.Name != existing.Name && len(g.Name) > len(existing.Name) {
			existing.Name = g.Name
		}
		existing.Entries = append(existing.Entries, g)
		existing.TotalPlaytimeMinutes += g.PlaytimeMinutes
		normalized[key] = existing
	}
	return normalized
}

func findTrendingOverlap(
	merged map[string]NormalizedGame,
	trending []models.Game,
) []NormalizedGame {
	var overlap []NormalizedGame
	for _, t := range trending {
		key := normalizeName(t.Name)
		if ng, ok := merged[key]; ok {
			overlap = append(overlap, ng)
		}
	}
	return overlap
}

func findTrendingCandidates(
	topPlayed []NormalizedGame,
	trending []models.Game,
) []TrendingCandidate {
	if len(trending) == 0 {
		return nil
	}

	owned := make(map[string]bool)
	for _, ng := range topPlayed {
		owned[normalizeName(ng.Name)] = true
	}

	var candidates []TrendingCandidate
	for _, t := range trending {
		key := normalizeName(t.Name)
		if owned[key] {
			continue
		}
		candidates = append(candidates, TrendingCandidate{
			Game:        t,
			MatchReason: "trending game you don't own yet",
		})
	}

	if len(candidates) > 10 {
		candidates = candidates[:10]
	}
	return candidates
}
