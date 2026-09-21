// Command server_ai proxies the card-recognition calls of the bridge client to
// OpenRouter: the vision model reads the cards on a photo, and the API key
// stays here instead of travelling to the browser.
package main

import (
	"log"

	"github.com/joho/godotenv"

	"server_ai/internal/config"
	"server_ai/internal/server"
)

func main() {
	// Load .env if present; in production env vars are injected directly.
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("configuration error: %v", err)
	}

	if err := server.Run(cfg); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
