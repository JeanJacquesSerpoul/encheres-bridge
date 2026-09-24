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

	// AllowedModels bounds what a caller may spend the key on: the default
	// model is always in it, ALLOWED_MODELS adds the others.
	AllowedModels []string
	// RateLimitPerMin caps /api/* requests per client IP; 0 disables it.
	RateLimitPerMin int // RATE_LIMIT_PER_MIN
	RateLimitBurst  int // RATE_LIMIT_BURST
	// TrustProxy lets X-Forwarded-For / X-Real-IP name the client. Only set it
	// behind a reverse proxy that overwrites them, or any caller picks its IP.
	TrustProxy bool // TRUST_PROXY
	// EnableModelsEndpoint exposes GET /api/models, which the client never calls.
	EnableModelsEndpoint bool // ENABLE_MODELS_ENDPOINT
}

// Load reads environment variables and returns a validated Config.
func Load() (*Config, error) {
	port := getEnvOrDefault("PORT", "9013")

	openRouterAPIKey := strings.TrimSpace(os.Getenv("OPENROUTER_API_KEY"))
	if openRouterAPIKey == "" {
		return nil, errors.New("OPENROUTER_API_KEY is required")
	}

	defaultModel := strings.TrimSpace(getEnvOrDefault("DEFAULT_MODEL", DefaultModel))
	if defaultModel == "" {
		return nil, errors.New("DEFAULT_MODEL must not be empty")
	}

	allowedModels := []string{defaultModel}
	if v := strings.TrimSpace(os.Getenv("ALLOWED_MODELS")); v != "" {
		for _, m := range parseCSV(v) {
			if m != defaultModel && m != "*" {
				allowedModels = append(allowedModels, m)
			}
		}
	}

	rateLimitPerMin, err := getNonNegativeInt("RATE_LIMIT_PER_MIN", 20)
	if err != nil {
		return nil, err
	}
	rateLimitBurst, err := getNonNegativeInt("RATE_LIMIT_BURST", 5)
	if err != nil {
		return nil, err
	}
	if rateLimitPerMin > 0 && rateLimitBurst == 0 {
		return nil, errors.New("RATE_LIMIT_BURST must be greater than 0 when RATE_LIMIT_PER_MIN is set")
	}

	trustProxy, err := getBool("TRUST_PROXY", false)
	if err != nil {
		return nil, err
	}
	enableModels, err := getBool("ENABLE_MODELS_ENDPOINT", false)
	if err != nil {
		return nil, err
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

		AllowedModels:        allowedModels,
		RateLimitPerMin:      rateLimitPerMin,
		RateLimitBurst:       rateLimitBurst,
		TrustProxy:           trustProxy,
		EnableModelsEndpoint: enableModels,
	}, nil
}

func getEnvOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getNonNegativeInt(key string, def int) (int, error) {
	v := os.Getenv(key)
	if v == "" {
		return def, nil
	}
	parsed, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer: %w", key, err)
	}
	if parsed < 0 {
		return 0, fmt.Errorf("%s must not be negative", key)
	}
	return parsed, nil
}

func getBool(key string, def bool) (bool, error) {
	v := os.Getenv(key)
	if v == "" {
		return def, nil
	}
	parsed, err := strconv.ParseBool(v)
	if err != nil {
		return false, fmt.Errorf("%s must be a boolean: %w", key, err)
	}
	return parsed, nil
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
