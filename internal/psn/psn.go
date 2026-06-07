// Package psn reads a user's PlayStation library via the (unofficial) My
// PlayStation internal API, authenticated with an NPSSO token.
package psn

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Public client credentials from the official PlayStation app (used by every
// community PSN library). Not secrets.
const (
	clientID     = "09515159-7237-4370-9b40-3806e67c0891"
	clientSecret = "ucPjka5tntB2KqsP"
	redirectURI  = "com.scee.psxandroid.scecompcall://redirect"
	scope        = "psn:mobile.v2.core psn:clientapp"

	authorizeURL = "https://ca.account.sony.com/api/authz/v3/oauth/authorize"
	tokenURL     = "https://ca.account.sony.com/api/authz/v3/oauth/token"
	gamelistURL  = "https://m.np.playstation.com/api/gamelist/v2/users/me/titles"
)

type Client struct {
	http       *http.Client
	noRedirect *http.Client
}

func NewClient() *Client {
	return &Client{
		http: &http.Client{Timeout: 20 * time.Second},
		noRedirect: &http.Client{
			Timeout:       20 * time.Second,
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		},
	}
}

type Tokens struct {
	AccessToken  string
	RefreshToken string
	AccountID    string
	ExpiresIn    int
}

// ExchangeNPSSO bootstraps a session from a fresh NPSSO token.
func (c *Client) ExchangeNPSSO(ctx context.Context, npsso string) (Tokens, error) {
	code, err := c.authCode(ctx, npsso)
	if err != nil {
		return Tokens{}, err
	}
	return c.token(ctx, url.Values{
		"code":         {code},
		"redirect_uri": {redirectURI},
		"grant_type":   {"authorization_code"},
		"token_format": {"jwt"},
	})
}

// Refresh gets a new access token from a stored refresh token (no NPSSO needed).
func (c *Client) Refresh(ctx context.Context, refreshToken string) (Tokens, error) {
	return c.token(ctx, url.Values{
		"refresh_token": {refreshToken},
		"grant_type":    {"refresh_token"},
		"token_format":  {"jwt"},
		"scope":         {scope},
	})
}

func (c *Client) authCode(ctx context.Context, npsso string) (string, error) {
	q := url.Values{
		"access_type":   {"offline"},
		"client_id":     {clientID},
		"redirect_uri":  {redirectURI},
		"response_type": {"code"},
		"scope":         {scope},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, authorizeURL+"?"+q.Encode(), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Cookie", "npsso="+npsso)

	resp, err := c.noRedirect.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Success is a redirect whose Location carries ?code=...
	loc, err := url.Parse(resp.Header.Get("Location"))
	if err != nil || loc.Query().Get("code") == "" {
		return "", fmt.Errorf("invalid or expired NPSSO — grab a fresh one")
	}
	return loc.Query().Get("code"), nil
}

func (c *Client) token(ctx context.Context, form url.Values) (Tokens, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return Tokens{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(clientID+":"+clientSecret)))

	resp, err := c.http.Do(req)
	if err != nil {
		return Tokens{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return Tokens{}, fmt.Errorf("PSN token request returned %s", resp.Status)
	}

	var t struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&t); err != nil {
		return Tokens{}, err
	}
	return Tokens{
		AccessToken:  t.AccessToken,
		RefreshToken: t.RefreshToken,
		AccountID:    accountIDFromJWT(t.AccessToken),
		ExpiresIn:    t.ExpiresIn,
	}, nil
}

// accountIDFromJWT pulls the stable account_id claim out of the access token.
func accountIDFromJWT(token string) string {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return ""
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return ""
	}
	var claims struct {
		AccountID string `json:"account_id"`
	}
	_ = json.Unmarshal(raw, &claims)
	return claims.AccountID
}
