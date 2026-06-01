// Package ai talks to any OpenAI-compatible chat endpoint (OpenAI, Ollama,
// Groq, OpenRouter, …) to generate game recommendations.
package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Client struct {
	baseURL string
	apiKey  string
	model   string
	http    *http.Client
}

func NewClient(baseURL, apiKey, model string) *Client {
	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		model:   model,
		http:    &http.Client{Timeout: 60 * time.Second},
	}
}

// Configured reports whether an API key is set (any value works for local Ollama).
func (c *Client) Configured() bool { return c.apiKey != "" }

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// chat sends a system + user message and returns the assistant's text.
func (c *Client) chat(ctx context.Context, system, user string) (string, error) {
	payload := map[string]any{
		"model":       c.model,
		"temperature": 0.7,
		// Constrain the output to valid JSON (supported by OpenAI and Ollama).
		"response_format": map[string]string{"type": "json_object"},
		"messages": []chatMessage{
			{Role: "system", Content: system},
			{Role: "user", Content: user},
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("AI endpoint returned %s", resp.Status)
	}

	var out struct {
		Choices []struct {
			Message chatMessage `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	if len(out.Choices) == 0 {
		return "", fmt.Errorf("AI endpoint returned no choices")
	}
	return out.Choices[0].Message.Content, nil
}
