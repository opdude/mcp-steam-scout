package recommender

import (
	"testing"

	"github.com/opdude/mcp-steam-scout/pkg/models"
)

func TestRecommend_AllPlatforms(t *testing.T) {
	r := New()
	rec := r.Recommend(
		[]models.Game{
			{ID: "1", Name: "Elden Ring", Platform: "steam", PlaytimeMinutes: 20400},
			{ID: "2", Name: "Skyrim", Platform: "steam", PlaytimeMinutes: 5400},
			{ID: "3", Name: "Disco Elysium", Platform: "steam", PlaytimeMinutes: 12},
			{ID: "4", Name: "Baldur's Gate 3", Platform: "steam"},
		},
		[]models.Game{
			{ID: "p1", Name: "God of War", Platform: "psn", PlaytimeMinutes: 2700},
			{ID: "p2", Name: "Horizon", Platform: "psn", PlaytimeMinutes: 1800},
		},
		[]models.Game{
			{ID: "x1", Name: "Halo Infinite", Platform: "xbox"},
			{ID: "x2", Name: "Skyrim Special Edition", Platform: "xbox", PlaytimeMinutes: 120},
		},
		[]models.Game{
			{ID: "e1", Name: "Sid Meier's Civilization VI", Platform: "epic"},
		},
		[]models.Game{
			{ID: "g1", Name: "The Witcher 3", Platform: "gog", PlaytimeMinutes: 4509},
		},
		[]models.Game{
			{ID: "t1", Name: "Elden Ring", Platform: "steam"},
			{ID: "t2", Name: "New Trending Game", Platform: "steam"},
			{ID: "t3", Name: "Hades", Platform: "steam"},
		},
	)

	if rec.TotalGames != 9 {
		t.Errorf("TotalGames = %d, want 8", rec.TotalGames)
	}

	if len(rec.Platforms) != 5 {
		t.Errorf("Platforms = %v, want 5", rec.Platforms)
	}

	found := false
	for _, uv := range rec.Unplayed {
		if uv.Name == "Baldur's Gate 3" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Baldur's Gate 3 should be in unplayed (0 playtime)")
	}

	found = false
	for _, db := range rec.Dabbled {
		if db.Name == "Disco Elysium" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Disco Elysium should be in dabbled (12 min)")
	}

	if len(rec.TopPlayed) < 3 {
		t.Errorf("TopPlayed = %d, want at least 3", len(rec.TopPlayed))
	}
	if rec.TopPlayed[0].Name != "Elden Ring" {
		t.Errorf("TopPlayed[0] = %s, want Elden Ring", rec.TopPlayed[0].Name)
	}

	if len(rec.TrendingOverlap) != 1 || rec.TrendingOverlap[0].Name != "Elden Ring" {
		t.Errorf("TrendingOverlap = %v, want [Elden Ring]", rec.TrendingOverlap)
	}

	if len(rec.TrendingCandidates) != 2 {
		t.Errorf("TrendingCandidates = %d, want 2", len(rec.TrendingCandidates))
	}
}

func TestRecommend_SkyrimMerge(t *testing.T) {
	r := New()
	rec := r.Recommend(
		[]models.Game{
			{ID: "1", Name: "Skyrim", Platform: "steam", PlaytimeMinutes: 5400},
			{ID: "2", Name: "Skyrim Special Edition", Platform: "steam", PlaytimeMinutes: 30},
		},
		nil, nil, nil, nil, nil,
	)

	for _, ng := range rec.TopPlayed {
		if ng.Name == "Skyrim" || ng.Name == "Skyrim Special Edition" {
			if ng.TotalPlaytimeMinutes != 5430 {
				t.Errorf("Skyrim merged playtime = %d, want 5430", ng.TotalPlaytimeMinutes)
			}
			if len(ng.Entries) != 2 {
				t.Errorf("Skyrim merged entries = %d, want 2", len(ng.Entries))
			}
		}
	}

	if len(rec.Unplayed)+len(rec.Dabbled)+1 != len(rec.TopPlayed) {
		t.Errorf("expected 1 normalized game total (%d unplayed + %d dabbled + 1 top played = %d top played)",
			len(rec.Unplayed), len(rec.Dabbled), len(rec.TopPlayed))
	}
}

func TestRecommend_OnlySteam(t *testing.T) {
	r := New()
	rec := r.Recommend(
		[]models.Game{
			{ID: "1", Name: "Game A", Platform: "steam", PlaytimeMinutes: 100},
			{ID: "2", Name: "Game B", Platform: "steam"},
		},
		nil, nil, nil, nil, nil,
	)

	if rec.TotalGames != 2 {
		t.Errorf("TotalGames = %d, want 2", rec.TotalGames)
	}
	if len(rec.Platforms) != 1 || rec.Platforms[0] != "steam" {
		t.Errorf("Platforms = %v, want [steam]", rec.Platforms)
	}
}

func TestRecommend_NoGames(t *testing.T) {
	r := New()
	rec := r.Recommend(nil, nil, nil, nil, nil, nil)
	if rec.TotalGames != 0 {
		t.Errorf("TotalGames = %d, want 0", rec.TotalGames)
	}
}

func TestRecommend_NoTrendingOverlap(t *testing.T) {
	r := New()
	rec := r.Recommend(
		[]models.Game{
			{ID: "1", Name: "Portal", Platform: "steam", PlaytimeMinutes: 600},
		},
		nil, nil, nil, nil,
		[]models.Game{
			{ID: "t1", Name: "Half-Life 3", Platform: "steam"},
		},
	)

	if len(rec.TrendingOverlap) != 0 {
		t.Errorf("TrendingOverlap = %v, want empty", rec.TrendingOverlap)
	}
	if len(rec.TrendingCandidates) != 1 {
		t.Errorf("TrendingCandidates = %d, want 1", len(rec.TrendingCandidates))
	}
}
