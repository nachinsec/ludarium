package psn

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type Game struct {
	Name            string
	ImageURL        string
	Platform        string
	PlaytimeMinutes int
	LastPlayed      string
}

// Library returns the user's PlayStation play history (with playtime), paginated.
func (c *Client) Library(ctx context.Context, accessToken string) ([]Game, error) {
	var games []Game
	for offset := 0; ; {
		url := fmt.Sprintf("%s?limit=200&offset=%d&categories=ps4_game,ps5_native_game", gamelistURL, offset)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+accessToken)

		resp, err := c.http.Do(req)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode >= 300 {
			resp.Body.Close()
			return nil, fmt.Errorf("PSN gamelist returned %s", resp.Status)
		}

		var page struct {
			Titles []struct {
				Name               string `json:"name"`
				ImageURL           string `json:"imageUrl"`
				Category           string `json:"category"`
				PlayDuration       string `json:"playDuration"`
				LastPlayedDateTime string `json:"lastPlayedDateTime"`
			} `json:"titles"`
			TotalItemCount int `json:"totalItemCount"`
		}
		err = json.NewDecoder(resp.Body).Decode(&page)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}

		for _, t := range page.Titles {
			games = append(games, Game{
				Name:            t.Name,
				ImageURL:        t.ImageURL,
				Platform:        platform(t.Category),
				PlaytimeMinutes: parseDuration(t.PlayDuration),
				LastPlayed:      t.LastPlayedDateTime,
			})
		}

		offset += len(page.Titles)
		if len(page.Titles) == 0 || offset >= page.TotalItemCount {
			break
		}
	}
	return games, nil
}

func platform(category string) string {
	switch category {
	case "ps5_native_game":
		return "PS5"
	case "ps4_game":
		return "PS4"
	case "ps3_game":
		return "PS3"
	default:
		return "PlayStation"
	}
}

// parseDuration converts an ISO-8601 duration like "PT127H51M2S" to minutes.
func parseDuration(d string) int {
	d = strings.TrimPrefix(d, "PT")
	var minutes, num int
	for _, r := range d {
		switch {
		case r >= '0' && r <= '9':
			num = num*10 + int(r-'0')
		case r == 'H':
			minutes += num * 60
			num = 0
		case r == 'M':
			minutes += num
			num = 0
		case r == 'S':
			num = 0 // ignore seconds
		default:
			num = 0
		}
	}
	return minutes
}
