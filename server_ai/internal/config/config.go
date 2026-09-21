package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Provider is the single AI provider this server talks to. It is not a choice:
// the server exists to expose the OpenRouter API and nothing else.
const Provider = "OPENROUTER"

// DefaultModel is the vision model used when a request does not name one. A
// small "flash" model reads cards well enough and answers in a few seconds;
// override it with DEFAULT_MODEL.
const DefaultModel = "google/gemini-3.1-flash-lite"

// Config holds all runtime configuration loaded from environment variables.
type Config struct {
	Port               string
	MaxTokens          int
	Timeout            time.Duration
	DefaultModel       string   // DEFAULT_MODEL
	OpenRouterAPIKey   string   // OPENROUTER_API_KEY
	CORSAllowedOrigins []string // CORS_ORIGINS (or CORS_ALLOWED_ORIGINS), comma-separated
}

// Load reads environment variables and returns a validated Config.
func Load() (*Config, error) {
	port := getEnvOrDefault("PORT", "9009")

	openRouterAPIKey := strings.TrimSpace(os.Getenv("OPENROUTER_API_KEY"))
	if openRouterAPIKey == "" {
		return nil, errors.New("OPENROUTER_API_KEY is required")
	}

	defaultModel := strings.TrimSpace(getEnvOrDefault("DEFAULT_MODEL", DefaultModel))
	if defaultModel == "" {
		return nil, errors.New("DEFAULT_MODEL must not be empty")
	}

	maxTokens := 10000
	if v := os.Getenv("MAX_TOKENS"); v != "" {
		parsed, err := strconv.Atoi(v)
		if err != nil {
			return nil, fmt.Errorf("MAX_TOKENS must be an integer: %w", err)
		}
		if parsed <= 0 {
			return nil, errors.New("MAX_TOKENS must be greater than 0")
		}
		maxTokens = parsed
	}

	timeoutSec := 60
	if v := os.Getenv("TIMEOUT"); v != "" {
		parsed, err := strconv.Atoi(v)
		if err != nil {
			return nil, fmt.Errorf("TIMEOUT must be an integer (seconds): %w", err)
		}
		if parsed <= 0 {
			return nil, errors.New("TIMEOUT must be greater than 0 seconds")
		}
		timeoutSec = parsed
	}

	// CORS_ORIGINS is the name the bidding server already uses in this repo;
	// CORS_ALLOWED_ORIGINS is aiproxy's, kept so an existing .env still works.
	corsRaw := getEnvOrDefault("CORS_ORIGINS", getEnvOrDefault("CORS_ALLOWED_ORIGINS", "*"))

	return &Config{
		Port:               port,
		MaxTokens:          maxTokens,
		Timeout:            time.Duration(timeoutSec) * time.Second,
		DefaultModel:       defaultModel,
		OpenRouterAPIKey:   openRouterAPIKey,
		CORSAllowedOrigins: parseCSV(corsRaw),
	}, nil
}

func getEnvOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func parseCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	if len(out) == 0 {
		return []string{"*"}
	}
	return out
}
