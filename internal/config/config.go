package config

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds runtime configuration sourced from environment variables.
type Config struct {
	Port          int
	BaseURL       string // public base URL, used for Steam OpenID realm/return
	DBPath        string
	SessionSecret string
	SteamAPIKey   string // server-side key for reading public libraries
	AIBaseURL     string // OpenAI-compatible endpoint
	AIAPIKey      string // key for the AI endpoint (use any value for local Ollama)
	AIModel       string
	IGDBClientID  string // Twitch app credentials for IGDB
	IGDBSecret    string
	Dev           bool
}

// Load reads config from .env (if present) and the environment.
func Load() (*Config, error) {
	loadDotEnv(".env")

	cfg := &Config{
		Port:          envInt("PORT", 3000),
		BaseURL:       strings.TrimRight(env("BASE_URL", "http://localhost:3000"), "/"),
		DBPath:        env("DB_PATH", "./data/ludarium.db"),
		SessionSecret: os.Getenv("SESSION_SECRET"),
		SteamAPIKey:   os.Getenv("STEAM_API_KEY"),
		AIBaseURL:     strings.TrimRight(env("AI_BASE_URL", "https://api.openai.com/v1"), "/"),
		AIAPIKey:      os.Getenv("AI_API_KEY"),
		AIModel:       env("AI_MODEL", "gpt-4o-mini"),
		IGDBClientID:  os.Getenv("IGDB_CLIENT_ID"),
		IGDBSecret:    os.Getenv("IGDB_CLIENT_SECRET"),
		Dev:           env("ENV", "production") != "production",
	}

	if cfg.SessionSecret == "" {
		if !cfg.Dev {
			return nil, fmt.Errorf("SESSION_SECRET is required in production")
		}
		// Ephemeral secret for local dev so sessions survive within a run.
		buf := make([]byte, 32)
		_, _ = rand.Read(buf)
		cfg.SessionSecret = hex.EncodeToString(buf)
	}

	return cfg, nil
}

func (c *Config) Addr() string { return ":" + strconv.Itoa(c.Port) }

func env(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	if v, ok := os.LookupEnv(key); ok {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
