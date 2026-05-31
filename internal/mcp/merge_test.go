package mcp

import (
	"testing"

	"github.com/opdude/mcp-steam-scout/pkg/models"
)

func TestNormalizeTitle(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Edition suffixes
		{"The Elder Scrolls V: Skyrim Special Edition", "the elder scrolls v: skyrim"},
		{"The Elder Scrolls V: Skyrim - Special Edition", "the elder scrolls v: skyrim"},
		{"Metro 2033 Redux", "metro 2033"},
		{"Metro: Last Light Complete Edition", "metro: last light"},
		{"Total War: MEDIEVAL II - Definitive Edition", "total war: medieval ii"},
		{"Borderlands GOTY", "borderlands"},
		{"Borderlands Game of the Year", "borderlands"},
		{"BioShock Infinite", "bioshock infinite"},

		// Trademark symbols
		{"STAR WARS™ Battlefront™ II", "star wars battlefront ii"},
		{"DEATH STRANDING™", "death stranding"},
		{"Assassin's Creed® IV Black Flag", "assassin's creed iv black flag"},
		{"Tom Clancy’s Rainbow Six® Extraction", "tom clancy’s rainbow six extraction"},

		// Platform / year tags
		{"FINAL FANTASY VII (2013)", "final fantasy vii"},
		{"Company of Heroes - Legacy Edition", "company of heroes"},
		{"Grand Theft Auto V (PlayStation®5)", "grand theft auto v"},
		{"Insurgency: Sandstorm [PS4 & PS5]", "insurgency: sandstorm"},
		{"Warhammer 40,000: Dawn of War - Anniversary Edition", "warhammer 40,000: dawn of war"},
		{"Total War: ROME II - Emperor Edition", "total war: rome ii"},

		// Multiplayer / Demo / Beta suffixes
		{"Call of Duty: Modern Warfare 2 (2009) - Multiplayer", "call of duty: modern warfare 2"},
		{"Left 4 Dead 2 Demo", "left 4 dead 2"},
		{"DEFCON Beta Demo", "defcon"},

		// No change needed
		{"Portal 2", "portal 2"},
		{"Half-Life 2", "half-life 2"},
		{"Sid Meier's Civilization V", "sid meier's civilization v"},
		{"Dishonored", "dishonored"},

		// Edge: trailing whitespace
		{"  DOOM  ", "doom"},

		// Different edition for same base game
		{"Borderlands GOTY", "borderlands"},
		{"Borderlands 2", "borderlands 2"},
		{"Borderlands 3", "borderlands 3"},

		// Remastered variants
		{"Metro 2033 Redux", "metro 2033"},
		{"Metro 2033", "metro 2033"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := NormalizeTitle(tt.input)
			if got != tt.expected {
				t.Errorf("NormalizeTitle(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestMergeLibraries(t *testing.T) {
	steam := []models.Game{
		{ID: "220", Name: "Half-Life 2", Platform: "steam", PlaytimeMinutes: 212},
		{ID: "72850", Name: "The Elder Scrolls V: Skyrim", Platform: "steam", PlaytimeMinutes: 538},
		{ID: "49520", Name: "Borderlands 2", Platform: "steam", PlaytimeMinutes: 3153},
	}

	psn := []models.Game{
		{ID: "PPSA03747_00", Name: "The Elder Scrolls V: Skyrim Special Edition", Platform: "psn", PlaytimeMinutes: 2},
		{ID: "PPSA04609_00", Name: "ELDEN RING", Platform: "psn", PlaytimeMinutes: 9316},
	}

	xbox := []models.Game{
		{ID: "1770979388", Name: "Valheim", Platform: "xbox"},
		{ID: "1977289559", Name: "Baldur's Gate 3", Platform: "xbox"},
	}

	epic := []models.Game{
		{ID: "cd14dcaa4f3443f19f7169a980559c62:42ac1ee840304cb1807172a9b47dc8e3", Name: "Sid Meier's Civilization VI", Platform: "epic"},
	}

	gog := []models.Game{
		{ID: "1207664643", Name: "The Witcher 3: Wild Hunt", Platform: "gog", PlaytimeMinutes: 4509},
		{ID: "1207659102", Name: "FTL: Advanced Edition", Platform: "gog"},
	}

	merged := MergeLibraries(steam, psn, xbox, epic, gog)

	tests := []struct {
		name      string
		wantTotal int
		wantPlats int
	}{
		{
			name:      "the elder scrolls v: skyrim",
			wantTotal: 540, // 538 + 2
			wantPlats: 2,   // steam + psn
		},
		{
			name:      "half-life 2",
			wantTotal: 212,
			wantPlats: 1,
		},
		{
			name:      "borderlands 2",
			wantTotal: 3153,
			wantPlats: 1,
		},
		{
			name:      "elden ring",
			wantTotal: 9316,
			wantPlats: 1,
		},
		{
			name:      "valheim",
			wantTotal: 0,
			wantPlats: 1,
		},
		{
			name:      "baldur's gate 3",
			wantTotal: 0,
			wantPlats: 1,
		},
		{
			name:      "sid meier's civilization vi",
			wantTotal: 0,
			wantPlats: 1,
		},
		{
			name:      "the witcher 3: wild hunt",
			wantTotal: 4509,
			wantPlats: 1,
		},
		{
			name:      "ftl: advanced edition",
			wantTotal: 0,
			wantPlats: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, m := range merged {
				if m.NormalizedName == tt.name {
					if m.TotalPlaytimeMinutes != tt.wantTotal {
						t.Errorf("%s: TotalPlaytimeMinutes = %d, want %d", tt.name, m.TotalPlaytimeMinutes, tt.wantTotal)
					}
					if len(m.Platforms) != tt.wantPlats {
						t.Errorf("%s: platforms = %v, want %d", tt.name, m.Platforms, tt.wantPlats)
					}
					return
				}
			}
			t.Errorf("%s: not found in merged results", tt.name)
		})
	}

	// Verify unplayed games appear first (sorted ascending).
	if len(merged) > 1 {
		first := merged[0]
		second := merged[1]
		if first.TotalPlaytimeMinutes > second.TotalPlaytimeMinutes {
			t.Errorf("expected sort by ascending playtime, first=%d second=%d", first.TotalPlaytimeMinutes, second.TotalPlaytimeMinutes)
		}
	}
}
