package mcp

import (
	"regexp"
	"sort"
	"strings"

	"github.com/opdude/mcp-steam-scout/pkg/models"
)

var (
	// Patterns to strip from game titles for cross-platform matching.
	// Order matters: more specific patterns first.
	editionSuffixes = []string{
		" - definitive edition",
		" definitive edition",
		" - game of the year edition",
		" - game of the year",
		" game of the year edition",
		" game of the year",
		" - goty edition",
		" goty edition",
		" - goty",
		" goty",
		" - enhanced edition",
		" enhanced edition",
		" - complete edition",
		" complete edition",
		" - anniversary edition",
		" anniversary edition",
		" - emperor edition",
		" emperor edition",
		" - special edition",
		" special edition",
		" - deluxe edition",
		" deluxe edition",
		" - gold edition",
		" gold edition",
		" - ultimate edition",
		" ultimate edition",
		" - standard edition",
		" standard edition",
		" - legacy edition",
		" legacy edition",
		" - remake",
		" - remastered",
		" remastered",
		" redux",
	}

	reDemoBeta = regexp.MustCompile(`\s+(demo|beta)\s*$`)

	reTrailingPlatform = regexp.MustCompile(`\s*\[[^\]]*\]\s*$`)
	reYearTag          = regexp.MustCompile(`\s*\(\d{4}\)\s*$`)
	reSuffixes         = regexp.MustCompile(`\s*-\s*(multiplayer|single player|demo|beta)\s*$`)
	reTrademarks       = regexp.MustCompile(`[™®©]`)
	reParenSuffix      = regexp.MustCompile(`\s*\([^)]*\)\s*$`)
)

// NormalizeTitle reduces a game name to a canonical form for cross-platform
// matching. It strips edition suffixes, trademark symbols, platform tags,
// year tags, and trailing parentheses/brackets.
func NormalizeTitle(name string) string {
	s := name
	s = strings.ToLower(s)
	s = strings.TrimSpace(s)

	// Remove trademark symbols.
	s = reTrademarks.ReplaceAllString(s, "")

	// Strip known edition suffixes.
	for _, suffix := range editionSuffixes {
		s = strings.ReplaceAll(s, suffix, "")
	}

	// Strip trailing platform brackets like [PS4 & PS5].
	s = reTrailingPlatform.ReplaceAllString(s, "")

	// Strip year tags like (2013), (2009).
	s = reYearTag.ReplaceAllString(s, "")

	// Strip " - Multiplayer", " - Single Player", " - Demo", " - Beta".
	s = reSuffixes.ReplaceAllString(s, "")
	// Also strip trailing " demo", " beta" (without dash prefix).
	// Run in a loop to handle cascading like "Beta Demo".
	for {
		prev := s
		s = reDemoBeta.ReplaceAllString(s, "")
		if s == prev {
			break
		}
	}

	// Strip trailing parenthetical content if the normalized result
	// would otherwise already match — but only if it's clearly a
	// platform/re-release tag, not a game subtitle.
	// Examples: "(PlayStation®5)", "(Classic)", "(2003)"
	// We re-apply year removal + catch residual parenthetical tags.
	s = reParenSuffix.ReplaceAllString(s, "")

	s = strings.TrimSpace(s)
	return s
}

// displayTitle picks the best display name from a set of merged entries.
// Prefers the shortest name (less likely to have extra suffixes).
func displayTitle(entries []models.Game) string {
	if len(entries) == 0 {
		return ""
	}
	best := entries[0].Name
	for _, e := range entries[1:] {
		if len(e.Name) < len(best) {
			best = e.Name
		}
	}
	return best
}

// MergeLibraries takes games from multiple platform libraries, normalizes
// their titles, and merges matching entries. Returns games sorted by
// total playtime ascending (unplayed first).
func MergeLibraries(steamGames, psnGames, xboxGames, epicGames, gogGames []models.Game) []models.MergedGame {
	type entry struct {
		game models.Game
		key  string
	}

	var all []entry

	addEntries := func(games []models.Game) {
		for _, g := range games {
			all = append(all, entry{
				game: g,
				key:  NormalizeTitle(g.Name),
			})
		}
	}

	addEntries(steamGames)
	addEntries(psnGames)
	addEntries(xboxGames)
	addEntries(epicGames)
	addEntries(gogGames)

	// Group by normalized key.
	groups := make(map[string][]models.Game)
	seen := make(map[string]bool)
	for _, e := range all {
		groups[e.key] = append(groups[e.key], e.game)
		seen[e.key] = true
	}

	// Build unique keys list for sorted output.
	keys := make([]string, 0, len(seen))
	for k := range seen {
		keys = append(keys, k)
	}

	merged := make([]models.MergedGame, 0, len(keys))
	for _, key := range keys {
		entries := groups[key]
		total := 0
		platformSet := make(map[string]bool)
		for _, g := range entries {
			total += g.PlaytimeMinutes
			platformSet[g.Platform] = true
		}

		platforms := make([]string, 0, len(platformSet))
		for p := range platformSet {
			platforms = append(platforms, p)
		}
		sort.Strings(platforms)

		merged = append(merged, models.MergedGame{
			NormalizedName:       key,
			DisplayName:          displayTitle(entries),
			Entries:              entries,
			TotalPlaytimeMinutes: total,
			Platforms:            platforms,
		})
	}

	// Sort by total playtime ascending (unplayed first).
	sort.Slice(merged, func(i, j int) bool {
		if merged[i].TotalPlaytimeMinutes != merged[j].TotalPlaytimeMinutes {
			return merged[i].TotalPlaytimeMinutes < merged[j].TotalPlaytimeMinutes
		}
		return merged[i].NormalizedName < merged[j].NormalizedName
	})

	return merged
}
