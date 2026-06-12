// Package igdb queries the IGDB game database (auth via Twitch app tokens).
package igdb

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	tokenURL      = "https://id.twitch.tv/oauth2/token"
	gamesURL      = "https://api.igdb.com/v4/games"
	eventsURL     = "https://api.igdb.com/v4/events"
	imageFmt      = "https://images.igdb.com/igdb/image/upload/t_cover_big/%s.jpg"
	coverSmallFmt = "https://images.igdb.com/igdb/image/upload/t_cover_small/%s.jpg"
	shotFmt       = "https://images.igdb.com/igdb/image/upload/t_screenshot_big/%s.jpg"
	logoFmt       = "https://images.igdb.com/igdb/image/upload/t_logo_med/%s.png"
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
	Score       int      `json:"score"`       // 0..100 community rating, 0 = none
	Platforms   []string `json:"platforms"`   // short names, e.g. PS5, Switch 2, PC
	Hype        int      `json:"hype"`        // anticipation count (pre-release)
	ReleaseDate int64    `json:"releaseDate"` // unix seconds of first release, 0 = unknown

	// Detail-only (populated by GameByID).
	Videos  []GameVideo   `json:"videos"`
	Links   []GameLink    `json:"links"`
	Similar []SimilarGame `json:"similar"`
}

type GameVideo struct {
	Name      string `json:"name"`
	YoutubeID string `json:"youtubeId"`
}

type GameLink struct {
	Store string `json:"store"`
	URL   string `json:"url"`
}

type SimilarGame struct {
	IGDBID   int    `json:"igdbId"`
	Name     string `json:"name"`
	CoverURL string `json:"coverUrl"`
}

// Event is a gaming showcase/conference with its announced lineup.
type Event struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	StartTime     int64  `json:"startTime"` // unix seconds
	LogoURL       string `json:"logoUrl"`
	LiveStreamURL string `json:"liveStreamUrl"`
	GameCount     int    `json:"gameCount"`

	gameIDs []int // internal: ids to fetch the lineup
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

// GameByID fetches a single game by its IGDB id, with the richer detail fields
// (trailers, similar games, store links) used by the detail view.
func (c *Client) GameByID(ctx context.Context, id int) (*Game, error) {
	games, err := c.queryGames(ctx, fmt.Sprintf("%s where id = %d; limit 1;", gameDetailFields, id))
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

// Calendar returns anticipated games with a future release date, soonest first,
// for the release calendar.
func (c *Client) Calendar(ctx context.Context) ([]Game, error) {
	body := fmt.Sprintf(gameFields+
		` where first_release_date > %d & game_type = 0 & cover != null & hypes > 10;`+
		` sort first_release_date asc; limit 80;`, time.Now().Unix())
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

const baseGameFields = `name,summary,total_rating,cover.image_id,screenshots.image_id,` +
	`first_release_date,genres.name,platforms.name,hypes,involved_companies.developer,involved_companies.company.name`

// gameFields is the lean set for lists; gameDetailFields adds trailers, store
// links and similar games — only worth fetching for the single-game detail view.
const gameFields = "fields " + baseGameFields + ";"
const gameDetailFields = "fields " + baseGameFields +
	",videos.video_id,videos.name,websites.url,similar_games.id,similar_games.name,similar_games.cover.image_id;"

// Events returns recent gaming showcases (Summer Game Fest, State of Play,
// Nintendo Direct, …) that IGDB tracks, newest first. Dynamic: IGDB adds new
// conferences as they happen.
func (c *Client) Events(ctx context.Context) ([]Event, error) {
	body := `fields name,start_time,event_logo.image_id,live_stream_url,games;` +
		` where start_time != null & games != null; sort start_time desc; limit 40;`
	return c.queryEvents(ctx, body)
}

// EventGames returns the games announced at an event, most-anticipated first.
func (c *Client) EventGames(ctx context.Context, eventID int) ([]Game, error) {
	events, err := c.queryEvents(ctx, fmt.Sprintf("fields games; where id = %d; limit 1;", eventID))
	if err != nil || len(events) == 0 {
		return nil, err
	}
	ids := events[0].gameIDs
	if len(ids) == 0 {
		return []Game{}, nil
	}
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = fmt.Sprintf("%d", id)
	}
	body := fmt.Sprintf("%s where id = (%s); sort hypes desc; limit 200;", gameFields, strings.Join(parts, ","))
	return c.queryGames(ctx, body)
}

func (c *Client) queryEvents(ctx context.Context, body string) ([]Event, error) {
	raw, err := c.post(ctx, eventsURL, body)
	if err != nil {
		return nil, err
	}
	var events []rawEvent
	if err := json.Unmarshal(raw, &events); err != nil {
		return nil, err
	}
	out := make([]Event, 0, len(events))
	for _, e := range events {
		out = append(out, e.normalize())
	}
	return out, nil
}

type rawEvent struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	StartTime int64  `json:"start_time"`
	EventLogo struct {
		ImageID string `json:"image_id"`
	} `json:"event_logo"`
	LiveStreamURL string `json:"live_stream_url"`
	Games         []int  `json:"games"`
}

func (r rawEvent) normalize() Event {
	e := Event{
		ID:            r.ID,
		Name:          r.Name,
		StartTime:     r.StartTime,
		LiveStreamURL: r.LiveStreamURL,
		GameCount:     len(r.Games),
		gameIDs:       r.Games,
	}
	if r.EventLogo.ImageID != "" {
		e.LogoURL = fmt.Sprintf(logoFmt, r.EventLogo.ImageID)
	}
	return e
}

func (c *Client) queryGames(ctx context.Context, body string) ([]Game, error) {
	data, err := c.post(ctx, gamesURL, body)
	if err != nil {
		return nil, err
	}
	var raw []rawGame
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	out := make([]Game, 0, len(raw))
	for _, r := range raw {
		out = append(out, r.normalize())
	}
	return out, nil
}

// post runs an Apicalypse query against an IGDB endpoint and returns the body.
func (c *Client) post(ctx context.Context, url, body string) ([]byte, error) {
	token, err := c.appToken(ctx)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(body))
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
	return io.ReadAll(resp.Body)
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
	Platforms []struct {
		Name string `json:"name"`
	} `json:"platforms"`
	Hypes             int `json:"hypes"`
	InvolvedCompanies []struct {
		Developer bool `json:"developer"`
		Company   struct {
			Name string `json:"name"`
		} `json:"company"`
	} `json:"involved_companies"`
	Videos []struct {
		VideoID string `json:"video_id"`
		Name    string `json:"name"`
	} `json:"videos"`
	Websites []struct {
		URL string `json:"url"`
	} `json:"websites"`
	SimilarGames []struct {
		ID    int    `json:"id"`
		Name  string `json:"name"`
		Cover struct {
			ImageID string `json:"image_id"`
		} `json:"cover"`
	} `json:"similar_games"`
}

// storeName classifies a website URL into a known storefront, or "" to skip.
func storeName(url string) string {
	switch {
	case strings.Contains(url, "store.steampowered.com"):
		return "Steam"
	case strings.Contains(url, "playstation.com"):
		return "PlayStation"
	case strings.Contains(url, "xbox.com"):
		return "Xbox"
	case strings.Contains(url, "nintendo.com"):
		return "Nintendo"
	case strings.Contains(url, "epicgames.com"):
		return "Epic"
	case strings.Contains(url, "gog.com"):
		return "GOG"
	default:
		return ""
	}
}

// shortPlatform trims IGDB's verbose platform names to compact badges.
var platformShort = strings.NewReplacer(
	"PC (Microsoft Windows)", "PC",
	"Xbox Series X|S", "Xbox Series",
	"PlayStation 5", "PS5",
	"PlayStation 4", "PS4",
	"Nintendo Switch 2", "Switch 2",
	"Nintendo Switch", "Switch",
)

func (r rawGame) normalize() Game {
	g := Game{IGDBID: r.ID, Name: r.Name, Summary: r.Summary, Score: int(r.TotalRating + 0.5)}
	if r.Cover.ImageID != "" {
		g.CoverURL = fmt.Sprintf(imageFmt, r.Cover.ImageID)
	}
	if r.FirstReleaseDate > 0 {
		y := time.Unix(r.FirstReleaseDate, 0).UTC().Year()
		g.ReleaseYear = &y
		g.ReleaseDate = r.FirstReleaseDate
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
	g.Hype = r.Hypes
	seen := map[string]bool{}
	for _, p := range r.Platforms {
		name := platformShort.Replace(p.Name)
		if !seen[name] {
			seen[name] = true
			g.Platforms = append(g.Platforms, name)
		}
	}
	for _, ic := range r.InvolvedCompanies {
		if ic.Developer {
			g.Developer = ic.Company.Name
			break
		}
	}
	for _, v := range r.Videos {
		if v.VideoID != "" {
			g.Videos = append(g.Videos, GameVideo{Name: v.Name, YoutubeID: v.VideoID})
		}
	}
	for _, w := range r.Websites {
		if store := storeName(w.URL); store != "" {
			g.Links = append(g.Links, GameLink{Store: store, URL: w.URL})
		}
	}
	for _, s := range r.SimilarGames {
		sg := SimilarGame{IGDBID: s.ID, Name: s.Name}
		if s.Cover.ImageID != "" {
			sg.CoverURL = fmt.Sprintf(coverSmallFmt, s.Cover.ImageID)
		}
		g.Similar = append(g.Similar, sg)
	}
	return g
}
