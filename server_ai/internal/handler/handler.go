// Package handler exposes the two JSON endpoints of the server: the chat call
// that reads the cards on a photo, and the model list behind it.
package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"server_ai/internal/openrouter"
)

// aiClient is the slice of the OpenRouter client the handlers need. An
// interface keeps them testable without a network round-trip.
type aiClient interface {
	Chat(ctx context.Context, req *openrouter.ChatRequest) (*openrouter.ChatResult, error)
	Models(ctx context.Context) ([]openrouter.Model, error)
}

// providerName is the only provider this server speaks to. Requests may still
// name it -- aiproxy clients do -- but nothing else is accepted.
const providerName = "OPENROUTER"

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}
