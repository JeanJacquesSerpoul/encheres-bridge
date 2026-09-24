package config

import "testing"

// helper: the API key is the one mandatory variable, so every test needs it.
func setAPIKey(t *testing.T) {
	t.Helper()
	t.Setenv("OPENROUTER_API_KEY", "sk-or-test")
}

func TestLoadRequiresAPIKey(t *testing.T) {
	t.Setenv("OPENROUTER_API_KEY", "")

	if _, err := Load(); err == nil {
		t.Fatal("expected error for missing OPENROUTER_API_KEY")
	}
}

func TestLoadDefaults(t *testing.T) {
	setAPIKey(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Port != "9013" {
		t.Fatalf("unexpected port: %q", cfg.Port)
	}
	if cfg.DefaultModel != DefaultModel {
		t.Fatalf("expected default model %q, got %q", DefaultModel, cfg.DefaultModel)
	}
	if cfg.MaxTokens != 10000 {
		t.Fatalf("unexpected MaxTokens: %d", cfg.MaxTokens)
	}
	if cfg.Timeout.Seconds() != 60 {
		t.Fatalf("unexpected Timeout: %v", cfg.Timeout)
	}
}

func TestLoadOverridesDefaultModel(t *testing.T) {
	setAPIKey(t)
	t.Setenv("DEFAULT_MODEL", "google/gemini-3.1-pro")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.DefaultModel != "google/gemini-3.1-pro" {
		t.Fatalf("unexpected default model: %q", cfg.DefaultModel)
	}
}

func TestLoadRejectsInvalidTimeout(t *testing.T) {
	setAPIKey(t)
	t.Setenv("TIMEOUT", "0")

	if _, err := Load(); err == nil {
		t.Fatal("expected error for TIMEOUT=0")
	}
}

func TestLoadRejectsInvalidMaxTokens(t *testing.T) {
	setAPIKey(t)
	t.Setenv("MAX_TOKENS", "-1")

	if _, err := Load(); err == nil {
		t.Fatal("expected error for MAX_TOKENS=-1")
	}
}

func TestLoadParsesCORSOrigins(t *testing.T) {
	setAPIKey(t)
	t.Setenv("CORS_ORIGINS", "https://a.example, https://b.example")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(cfg.CORSAllowedOrigins) != 2 ||
		cfg.CORSAllowedOrigins[0] != "https://a.example" ||
		cfg.CORSAllowedOrigins[1] != "https://b.example" {
		t.Fatalf("unexpected origins: %#v", cfg.CORSAllowedOrigins)
	}
}

func TestLoadAcceptsAiproxyCORSName(t *testing.T) {
	setAPIKey(t)
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://c.example")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(cfg.CORSAllowedOrigins) != 1 || cfg.CORSAllowedOrigins[0] != "https://c.example" {
		t.Fatalf("unexpected origins: %#v", cfg.CORSAllowedOrigins)
	}
}

func TestLoadSecurityDefaults(t *testing.T) {
	setAPIKey(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(cfg.AllowedModels) != 1 || cfg.AllowedModels[0] != DefaultModel {
		t.Fatalf("unexpected allowed models: %#v", cfg.AllowedModels)
	}
	if cfg.RateLimitPerMin != 20 || cfg.RateLimitBurst != 5 {
		t.Fatalf("unexpected rate limit: %d/min, burst %d", cfg.RateLimitPerMin, cfg.RateLimitBurst)
	}
	if cfg.TrustProxy {
		t.Fatal("TrustProxy must default to false")
	}
	if cfg.EnableModelsEndpoint {
		t.Fatal("EnableModelsEndpoint must default to false")
	}
}

func TestLoadParsesAllowedModels(t *testing.T) {
	setAPIKey(t)
	t.Setenv("ALLOWED_MODELS", "openai/gpt-4o, "+DefaultModel+", *")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	// The default comes first, is not repeated, and "*" grants nothing.
	if len(cfg.AllowedModels) != 2 ||
		cfg.AllowedModels[0] != DefaultModel ||
		cfg.AllowedModels[1] != "openai/gpt-4o" {
		t.Fatalf("unexpected allowed models: %#v", cfg.AllowedModels)
	}
}

func TestLoadParsesSecurityFlags(t *testing.T) {
	setAPIKey(t)
	t.Setenv("RATE_LIMIT_PER_MIN", "0")
	t.Setenv("TRUST_PROXY", "true")
	t.Setenv("ENABLE_MODELS_ENDPOINT", "1")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.RateLimitPerMin != 0 || !cfg.TrustProxy || !cfg.EnableModelsEndpoint {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}

func TestLoadRejectsInvalidSecurityValues(t *testing.T) {
	for key, value := range map[string]string{
		"RATE_LIMIT_PER_MIN":     "-1",
		"RATE_LIMIT_BURST":       "abc",
		"TRUST_PROXY":            "maybe",
		"ENABLE_MODELS_ENDPOINT": "yes please",
	} {
		t.Run(key, func(t *testing.T) {
			setAPIKey(t)
			t.Setenv(key, value)

			if _, err := Load(); err == nil {
				t.Fatalf("expected error for %s=%s", key, value)
			}
		})
	}
}

func TestLoadRejectsZeroBurstWithRateLimit(t *testing.T) {
	setAPIKey(t)
	t.Setenv("RATE_LIMIT_BURST", "0")

	if _, err := Load(); err == nil {
		t.Fatal("expected error for RATE_LIMIT_BURST=0 with a rate limit")
	}
}
