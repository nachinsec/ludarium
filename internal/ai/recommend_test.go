package ai

import "testing"

func TestParseRecommendations_FencedAndProse(t *testing.T) {
	raw := "Sure! Here you go:\n```json\n" +
		`[{"title":"Outer Wilds","reason":"Exploration."},{"title":"Hades","reason":"Roguelike."}]` +
		"\n```"
	recs, err := parseRecommendations(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(recs) != 2 || recs[0].Title != "Outer Wilds" {
		t.Fatalf("got %+v", recs)
	}
}

func TestParseRecommendations_Object(t *testing.T) {
	raw := `{"recommendations": [{"title":"Outer Wilds","reason":"Exploration."}]}`
	recs, err := parseRecommendations(raw)
	if err != nil || len(recs) != 1 || recs[0].Title != "Outer Wilds" {
		t.Fatalf("got %+v, err %v", recs, err)
	}
}

func TestParseRecommendations_NoArray(t *testing.T) {
	if _, err := parseRecommendations("I cannot help with that."); err == nil {
		t.Fatal("expected error for missing array")
	}
}

func TestExcludeOwned(t *testing.T) {
	owned := []string{"Hades", "Hollow Knight"}
	recs := []Recommendation{
		{Title: "hades", Reason: "already owned (case-insensitive)"},
		{Title: "Outer Wilds", Reason: "new pick"},
	}
	out := excludeOwned(recs, owned)
	if len(out) != 1 || out[0].Title != "Outer Wilds" {
		t.Fatalf("expected only the un-owned game, got %+v", out)
	}
}

func TestTaste(t *testing.T) {
	games := []GameInfo{
		{Title: "Played", Status: "backlog", Hours: 90},  // liked (hours)
		{Title: "Loved", Status: "completed", Rating: 5}, // liked (status)
		{Title: "Fresh", Status: "backlog", Hours: 0},    // not taste
	}
	liked := Taste(games)
	if len(liked) != 2 || liked[0].Title != "Played" {
		t.Fatalf("got %+v", liked)
	}
}
