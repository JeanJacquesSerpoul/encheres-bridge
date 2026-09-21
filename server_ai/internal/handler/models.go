package handler

import (
	"context"
	"errors"
	"log"
	"net/http"

	"server_ai/internal/openrouter"
)

// ModelsHandler handles GET /api/models.
type ModelsHandler struct {
	client aiClient
}

// NewModelsHandler creates a ModelsHandler over the OpenRouter client.
func NewModelsHandler(client aiClient) *ModelsHandler {
	return &ModelsHandler{client: client}
}

// ModelsResponse is the JSON envelope returned by GET /api/models.
type ModelsResponse struct {
	Success bool               `json:"success"`
	Count   int                `json:"count,omitempty"`
	Models  []openrouter.Model `json:"models,omitempty"`
	Error   string             `json:"error,omitempty"`
}

// ServeHTTP implements http.Handler. An optional ?provider= is tolerated as
// long as it names OPENROUTER.
func (h *ModelsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := checkProvider(r.URL.Query().Get("provider")); err != nil {
		writeJSON(w, http.StatusBadRequest, ModelsResponse{Success: false, Error: err.Error()})
		return
	}

	models, err := h.client.Models(r.Context())
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			writeJSON(w, http.StatusGatewayTimeout, ModelsResponse{
				Success: false,
				Error:   "upstream request timed out",
			})
			return
		}
		log.Printf("models upstream error: %v", err)
		writeJSON(w, http.StatusBadGateway, ModelsResponse{
			Success: false,
			Error:   "upstream request failed",
		})
		return
	}

	writeJSON(w, http.StatusOK, ModelsResponse{
		Success: true,
		Count:   len(models),
		Models:  models,
	})
}
