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
	if cfg.Port != "9009" {
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
