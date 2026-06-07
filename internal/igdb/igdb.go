// Package igdb queries the IGDB game database (auth via Twitch app tokens).
package igdb

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	tokenURL = "https://id.twitch.tv/oauth2/token"
	gamesURL  = "https://api.igdb.com/v4/games"
	imageFmt  = "https://images.igdb.com/igdb/image/upload/t_cover_big/%s.jpg"
	shotFmt   = "https://images.igdb.com/igdb/image/upload/t_screenshot_big/%s.jpg"
)

type Client struct {
	clientID string
	secret   string
	http     *http.Client

	mu      sync.Mutex
	token   string
	expires time.Time
}

func NewClient(clientID, secret string) *Client {
	return &Client{clientID: clientID, secret: secret, http: &http.Client{Timeout: 15 * time.Second}}
}

func (c *Client) Configured() bool { return c.clientID != "" && c.secret != "" }

// Game is a normalized IGDB result.
type Game struct {
	IGDBID      int      `json:"igdbId"`
	Name        string   `json:"name"`
	CoverURL    string   `json:"coverUrl"`
	ReleaseYear *int     `json:"releaseYear"`
	Genres      []string `json:"genres"`
	Developer   string   `json:"developer"`
	Summary     string   `json:"summary"`
	Screenshots []string `json:"screenshots"`
	Score       int      `json:"score"` // 0..100 community rating, 0 = none
}

// Search returns games matching a free-text query.
func (c *Client) Search(ctx context.Context, query string) ([]Game, error) {
	q := strings.ReplaceAll(query, `"`, "")
	body := fmt.Sprintf(`search "%s"; %s limit 12;`, q, gameFields)
	return c.queryGames(ctx, body)
}

// GameBySteamAppID resolves a Steam app id to its IGDB game, if any.
// Steam is external_game_source = 1 in IGDB.
func (c *Client) GameBySteamAppID(ctx context.Context, appID int) (*Game, error) {
	body := fmt.Sprintf(
		`%s where external_games.external_game_source = 1 & external_games.uid = "%d"; limit 1;`,
		gameFields, appID)
	games, err := c.queryGames(ctx, body)
	if err != nil || len(games) == 0 {
		return nil, err
	}
	return &games[0], nil
}

// GameByID fetches a single game by its IGDB id.
func (c *Client) GameByID(ctx context.Context, id int) (*Game, error) {
	games, err := c.queryGames(ctx, fmt.Sprintf("%s where id = %d; limit 1;", gameFields, id))
	if err != nil || len(games) == 0 {
		return nil, err
	}
	return &games[0], nil
}

// Popular returns highly-rated main games. (IGDB renamed category → game_type.)
func (c *Client) Popular(ctx context.Context) ([]Game, error) {
	return c.queryGames(ctx, gameFields+
		` where game_type = 0 & total_rating_count > 200 & cover != null;`+
		` sort total_rating_count desc; limit 18;`)
}

// Upcoming returns hyped main games releasing in the future.
func (c *Client) Upcoming(ctx context.Context) ([]Game, error) {
	body := fmt.Sprintf(gameFields+
		` where first_release_date > %d & game_type = 0 & cover != null & hypes > 2;`+
		` sort first_release_date asc; limit 18;`, time.Now().Unix())
	return c.queryGames(ctx, body)
}

// Candidates returns a varied set of well-known games for the taste quiz. A
// random window into the rating-sorted list keeps each run different.
func (c *Client) Candidates(ctx context.Context) ([]Game, error) {
	body := fmt.Sprintf(gameFields+
		` where game_type = 0 & total_rating_count > 30 & cover != null;`+
		` sort total_rating_count desc; limit 24; offset %d;`, rand.Intn(160))
	return c.queryGames(ctx, body)
}

const gameFields = `fields name,summary,total_rating,cover.image_id,screenshots.image_id,` +
	`first_release_date,genres.name,involved_companies.developer,involved_companies.company.name;`

func (c *Client) queryGames(ctx context.Context, body string) ([]Game, error) {
	token, err := c.appToken(ctx)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, gamesURL, strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Client-ID", c.clientID)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "text/plain")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("IGDB returned %s", resp.Status)
	}

	var raw []rawGame
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}
	out := make([]Game, 0, len(raw))
	for _, r := range raw {
		out = append(out, r.normalize())
	}
	return out, nil
}

type rawGame struct {
	ID               int     `json:"id"`
	Name             string  `json:"name"`
	Summary          string  `json:"summary"`
	TotalRating      float64 `json:"total_rating"`
	FirstReleaseDate int64   `json:"first_release_date"`
	Cover            struct {
		ImageID string `json:"image_id"`
	} `json:"cover"`
	Screenshots []struct {
		ImageID string `json:"image_id"`
	} `json:"screenshots"`
	Genres []struct {
		Name string `json:"name"`
	} `json:"genres"`
	InvolvedCompanies []struct {
		Developer bool `json:"developer"`
		Company   struct {
			Name string `json:"name"`
		} `json:"company"`
	} `json:"involved_companies"`
}

func (r rawGame) normalize() Game {
	g := Game{IGDBID: r.ID, Name: r.Name, Summary: r.Summary, Score: int(r.TotalRating + 0.5)}
	if r.Cover.ImageID != "" {
		g.CoverURL = fmt.Sprintf(imageFmt, r.Cover.ImageID)
	}
	if r.FirstReleaseDate > 0 {
		y := time.Unix(r.FirstReleaseDate, 0).UTC().Year()
		g.ReleaseYear = &y
	}
	for i, sh := range r.Screenshots {
		if i >= 6 {
			break
		}
		g.Screenshots = append(g.Screenshots, fmt.Sprintf(shotFmt, sh.ImageID))
	}
	for _, ge := range r.Genres {
		g.Genres = append(g.Genres, ge.Name)
	}
	for _, ic := range r.InvolvedCompanies {
		if ic.Developer {
			g.Developer = ic.Company.Name
			break
		}
	}
	return g
}
