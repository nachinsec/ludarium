package auth

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/nachinsec/ludarium/internal/config"
)

const (
	steamOpenIDEndpoint = "https://steamcommunity.com/openid/login"
	steamIdentifier     = "http://specs.openid.net/auth/2.0/identifier_select"
	openIDNamespace     = "http://specs.openid.net/auth/2.0"
)

var steamIDRe = regexp.MustCompile(`^https?://steamcommunity\.com/openid/id/(\d+)$`)

// Steam handles OpenID login and Steam Web API calls.
type Steam struct {
	cfg    *config.Config
	client *http.Client
}

func NewSteam(cfg *config.Config) *Steam {
	return &Steam{cfg: cfg, client: &http.Client{Timeout: 10 * time.Second}}
}

// AuthURL is where to redirect the user to start Steam login.
func (s *Steam) AuthURL() string {
	returnTo := s.cfg.BaseURL + "/api/auth/steam/callback"
	q := url.Values{
		"openid.ns":         {openIDNamespace},
		"openid.mode":       {"checkid_setup"},
		"openid.return_to":  {returnTo},
		"openid.realm":      {s.cfg.BaseURL},
		"openid.identity":   {steamIdentifier},
		"openid.claimed_id": {steamIdentifier},
	}
	return steamOpenIDEndpoint + "?" + q.Encode()
}

// Verify confirms the signed OpenID callback with Steam and returns the Steam ID.
func (s *Steam) Verify(ctx context.Context, params url.Values) (string, error) {
	claimed := params.Get("openid.claimed_id")
	m := steamIDRe.FindStringSubmatch(claimed)
	if m == nil {
		return "", fmt.Errorf("invalid claimed_id: %q", claimed)
	}
	steamID := m[1]

	// Echo the params back with mode=check_authentication for Steam to validate.
	check := url.Values{}
	for k, v := range params {
		check[k] = v
	}
	check.Set("openid.mode", "check_authentication")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, steamOpenIDEndpoint,
		strings.NewReader(check.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if !strings.Contains(string(body), "is_valid:true") {
		return "", fmt.Errorf("steam did not validate the assertion")
	}
	return steamID, nil
}

type OwnedGame struct {
	AppID           int
	Name            string
	PlaytimeMinutes int
}

// OwnedGames uses the official Web API when a key is set, else the public XML.
// Both need the target profile's game details to be public.
func (s *Steam) OwnedGames(ctx context.Context, steamID string) ([]OwnedGame, error) {
	if s.cfg.SteamAPIKey != "" {
		return s.ownedGamesAPI(ctx, steamID)
	}
	return s.ownedGamesPublic(ctx, steamID)
}

func (s *Steam) ownedGamesAPI(ctx context.Context, steamID string) ([]OwnedGame, error) {
	endpoint := fmt.Sprintf(
		"https://api.steampowered.com/IPlayerService/GetOwnedGames/v1/"+
			"?key=%s&steamid=%s&include_appinfo=1&include_played_free_games=1&format=json",
		url.QueryEscape(s.cfg.SteamAPIKey), url.QueryEscape(steamID))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var payload struct {
		Response struct {
			Games []struct {
				AppID           int    `json:"appid"`
				Name            string `json:"name"`
				PlaytimeForever int    `json:"playtime_forever"`
			} `json:"games"`
		} `json:"response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	out := make([]OwnedGame, 0, len(payload.Response.Games))
	for _, g := range payload.Response.Games {
		out = append(out, OwnedGame{AppID: g.AppID, Name: g.Name, PlaytimeMinutes: g.PlaytimeForever})
	}
	return out, nil
}

func (s *Steam) ownedGamesPublic(ctx context.Context, steamID string) ([]OwnedGame, error) {
	endpoint := fmt.Sprintf(
		"https://steamcommunity.com/profiles/%s/games?tab=all&xml=1", url.PathEscape(steamID))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "ludarium/0.1")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Steam now redirects unauthenticated requests here to login, so this rarely works.
	if strings.Contains(resp.Request.URL.Path, "/login") || !strings.Contains(string(body), "<gamesList>") {
		return nil, fmt.Errorf(
			"Steam no longer allows keyless library access — set STEAM_API_KEY on the server (see README)")
	}

	var doc struct {
		Games []struct {
			AppID int    `xml:"appID"`
			Name  string `xml:"name"`
			Hours string `xml:"hoursOnRecord"`
		} `xml:"games>game"`
	}
	if err := xml.Unmarshal(body, &doc); err != nil {
		return nil, fmt.Errorf("could not parse Steam games list: %w", err)
	}

	out := make([]OwnedGame, 0, len(doc.Games))
	for _, g := range doc.Games {
		out = append(out, OwnedGame{
			AppID:           g.AppID,
			Name:            g.Name,
			PlaytimeMinutes: parseHoursToMinutes(g.Hours),
		})
	}
	return out, nil
}

// parseHoursToMinutes parses Steam's "1,234.5" hours string.
func parseHoursToMinutes(s string) int {
	s = strings.ReplaceAll(strings.TrimSpace(s), ",", "")
	if s == "" {
		return 0
	}
	h, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return int(h * 60)
}

type Profile struct {
	Name   string
	Avatar string
}

// FetchProfile returns the public name/avatar, or a zero Profile if no key is set.
func (s *Steam) FetchProfile(ctx context.Context, steamID string) (Profile, error) {
	if s.cfg.SteamAPIKey == "" {
		return Profile{}, nil
	}
	endpoint := fmt.Sprintf(
		"https://api.steampowered.com/ISteamUser/GetPlayerSummaries/v2/?key=%s&steamids=%s",
		url.QueryEscape(s.cfg.SteamAPIKey), url.QueryEscape(steamID))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return Profile{}, err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return Profile{}, err
	}
	defer resp.Body.Close()

	var payload struct {
		Response struct {
			Players []struct {
				PersonaName string `json:"personaname"`
				AvatarFull  string `json:"avatarfull"`
			} `json:"players"`
		} `json:"response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return Profile{}, err
	}
	if len(payload.Response.Players) == 0 {
		return Profile{}, nil
	}
	p := payload.Response.Players[0]
	return Profile{Name: p.PersonaName, Avatar: p.AvatarFull}, nil
}
