package recommender

import "testing"

func TestNormalizeName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Skyrim", "skyrim"},
		{"The Elder Scrolls V: Skyrim", "elder scrolls v skyrim"},
		{"Skyrim Special Edition", "skyrim"},
		{"Skyrim Special Edition (PC)", "skyrim"},
		{"Skyrim - Special Edition", "skyrim"},
		{"Elden Ring", "elden ring"},
		{"Elden Ring Deluxe Edition", "elden ring"},
		{"God of War Ragnarök", "god of war ragnarök"},
		{"Cyberpunk 2077", "cyberpunk 2077"},
		{"Baldur's Gate 3", "baldur's gate 3"},
		{"Baldur's Gate 3 Deluxe Edition", "baldur's gate 3"},
		{"Hades", "hades"},
		{"Hades [Game of the Year]", "hades"},
		{"Game of the Year Edition", ""},
		{"The Witcher 3: Wild Hunt - Game of the Year Edition", "witcher 3 wild hunt"},
		{"Disco Elysium - The Final Cut", "disco elysium the final cut"},
		{"Divinity: Original Sin 2 - Definitive Edition", "divinity original sin 2"},
		{"", ""},
	}
	for _, tt := range tests {
		got := normalizeName(tt.input)
		if got != tt.want {
			t.Errorf("normalizeName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
