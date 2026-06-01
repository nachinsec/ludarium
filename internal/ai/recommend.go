package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// GameInfo is the minimal per-game signal the recommender needs.
type GameInfo struct {
	Title  string
	Status string
	Rating int // 0 = unrated
	Hours  float64
}

type Recommendation struct {
	Title  string `json:"title"`
	Reason string `json:"reason"`
}

const systemPrompt = `You are a video game recommendation assistant.
Given a user's taste (games they played a lot or rated highly) and the games they ALREADY OWN, recommend NEW games they do not own yet.
Rules:
- NEVER recommend a game that appears in the OWNED list.
- Recommend real, existing games that match their taste.
- Pick 5, best first. Each reason must be one short sentence.
Reply with ONLY a JSON object of this exact shape:
{"recommendations": [{"title": "...", "reason": "..."}]}`

// RecommendNew asks the model for games the user does NOT own, based on taste.
// Grounding against a real game DB happens upstream (the caller resolves titles
// via IGDB); here we only drop anything that's clearly already owned.
func RecommendNew(ctx context.Context, c *Client, liked []GameInfo, owned []string) ([]Recommendation, error) {
	var b strings.Builder
	b.WriteString("TASTE (games that reflect the user's preferences):\n")
	for _, g := range liked {
		fmt.Fprintf(&b, "- %s", g.Title)
		switch {
		case g.Hours > 0 && g.Rating > 0:
			fmt.Fprintf(&b, " (%.0fh, rated %d/5)", g.Hours, g.Rating)
		case g.Hours > 0:
			fmt.Fprintf(&b, " (%.0fh)", g.Hours)
		case g.Rating > 0:
			fmt.Fprintf(&b, " (rated %d/5)", g.Rating)
		}
		b.WriteString("\n")
	}
	b.WriteString("\nOWNED (never recommend any of these):\n")
	for _, t := range owned {
		fmt.Fprintf(&b, "- %s\n", t)
	}

	raw, err := c.chat(ctx, systemPrompt, b.String())
	if err != nil {
		return nil, err
	}
	recs, err := parseRecommendations(raw)
	if err != nil {
		return nil, err
	}
	return excludeOwned(recs, owned), nil
}

// Taste returns the games that signal the user's preferences, most-played first.
func Taste(games []GameInfo) []GameInfo {
	var liked []GameInfo
	for _, g := range games {
		if g.Status == "completed" || g.Rating >= 4 || g.Hours >= 5 {
			liked = append(liked, g)
		}
	}
	sort.Slice(liked, func(i, j int) bool { return liked[i].Hours > liked[j].Hours })
	if len(liked) > 10 {
		liked = liked[:10]
	}
	return liked
}

// excludeOwned drops recommendations the user already owns (case-insensitive).
func excludeOwned(recs []Recommendation, owned []string) []Recommendation {
	set := make(map[string]bool, len(owned))
	for _, t := range owned {
		set[strings.ToLower(strings.TrimSpace(t))] = true
	}
	out := make([]Recommendation, 0, len(recs))
	for _, r := range recs {
		if !set[strings.ToLower(strings.TrimSpace(r.Title))] {
			out = append(out, r)
		}
	}
	return out
}

// parseRecommendations rescues the recommendations from the model's reply,
// accepting either {"recommendations": [...]} or a bare [...] array, possibly
// wrapped in prose or code fences.
func parseRecommendations(raw string) ([]Recommendation, error) {
	if i, j := strings.Index(raw, "{"), strings.LastIndex(raw, "}"); i >= 0 && j > i {
		var obj struct {
			Recommendations []Recommendation `json:"recommendations"`
		}
		if err := json.Unmarshal([]byte(raw[i:j+1]), &obj); err == nil && len(obj.Recommendations) > 0 {
			return obj.Recommendations, nil
		}
	}
	if i, j := strings.Index(raw, "["), strings.LastIndex(raw, "]"); i >= 0 && j > i {
		var recs []Recommendation
		if err := json.Unmarshal([]byte(raw[i:j+1]), &recs); err == nil {
			return recs, nil
		}
	}
	return nil, fmt.Errorf("AI did not return a recommendation list")
}
