package igdb

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// appToken returns a valid Twitch app access token, fetching a new one when the
// cached token is missing or about to expire.
func (c *Client) appToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.token != "" && time.Now().Before(c.expires) {
		return c.token, nil
	}

	params := url.Values{
		"client_id":     {c.clientID},
		"client_secret": {c.secret},
		"grant_type":    {"client_credentials"},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL+"?"+params.Encode(), nil)
	if err != nil {
		return "", err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("twitch token request returned %s", resp.Status)
	}

	var out struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}

	c.token = out.AccessToken
	// Refresh a minute early to avoid edge-of-expiry failures.
	c.expires = time.Now().Add(time.Duration(out.ExpiresIn-60) * time.Second)
	return c.token, nil
}
