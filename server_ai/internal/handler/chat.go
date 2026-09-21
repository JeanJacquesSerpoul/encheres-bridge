package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"server_ai/internal/openrouter"
)

const maxChatRequestBodyBytes = 20 << 20 // 20 MiB (allows photo and PDF uploads)

// chatRequest is the inbound body of POST /api/chat.
type chatRequest struct {
	Text     string `json:"text"`
	Image    string `json:"image,omitempty"`    // HTTPS URL or base64 data-URI
	PDF      string `json:"pdf,omitempty"`      // base64 data-URI (data:application/pdf;base64,...)
	Model    string `json:"model,omitempty"`    // optional: the configured default model otherwise
	Provider string `json:"provider,omitempty"` // optional: OPENROUTER only
}

// ChatResponse is the JSON envelope always returned by POST /api/chat.
type ChatResponse struct {
	Success  bool   `json:"success"`
	Response string `json:"response,omitempty"`
	Model    string `json:"model,omitempty"`
	Usage    *Usage `json:"usage,omitempty"`
	Error    string `json:"error,omitempty"`
}

// Usage mirrors the token consumption reported by the upstream API.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ChatHandler handles POST /api/chat.
type ChatHandler struct {
	client       aiClient
	defaultModel string
}

// NewChatHandler creates a ChatHandler falling back to defaultModel whenever a
// request leaves the model out.
func NewChatHandler(client aiClient, defaultModel string) *ChatHandler {
	return &ChatHandler{client: client, defaultModel: defaultModel}
}

// ServeHTTP implements http.Handler.
func (h *ChatHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		writeJSON(w, http.StatusBadRequest, ChatResponse{
			Success: false,
			Error:   "Content-Type must be application/json",
		})
		return
	}

	var req chatRequest
	r.Body = http.MaxBytesReader(w, r.Body, maxChatRequestBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ChatResponse{
			Success: false,
			Error:   "invalid JSON body: " + err.Error(),
		})
		return
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		writeJSON(w, http.StatusBadRequest, ChatResponse{
			Success: false,
			Error:   "invalid JSON body: only one JSON object is allowed",
		})
		return
	}

	if req.Text == "" {
		writeJSON(w, http.StatusBadRequest, ChatResponse{Success: false, Error: "text field is required"})
		return
	}
	if err := checkProvider(req.Provider); err != nil {
		writeJSON(w, http.StatusBadRequest, ChatResponse{Success: false, Error: err.Error()})
		return
	}

	model := strings.TrimSpace(req.Model)
	if model == "" {
		model = h.defaultModel
	}

	result, err := h.client.Chat(r.Context(), &openrouter.ChatRequest{
		Text:  req.Text,
		Image: req.Image,
		PDF:   req.PDF,
		Model: model,
	})
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			writeJSON(w, http.StatusGatewayTimeout, ChatResponse{
				Success: false,
				Error:   "upstream request timed out",
			})
			return
		}
		log.Printf("chat upstream error: %v", err)
		writeJSON(w, http.StatusBadGateway, ChatResponse{
			Success: false,
			Error:   "upstream request failed",
		})
		return
	}

	writeJSON(w, http.StatusOK, ChatResponse{
		Success:  true,
		Response: result.Content,
		Model:    result.Model,
		Usage: &Usage{
			PromptTokens:     result.PromptTokens,
			CompletionTokens: result.CompletionTokens,
			TotalTokens:      result.TotalTokens,
		},
	})
}

// checkProvider accepts an absent provider and OPENROUTER, and refuses the
// rest rather than silently answering for a provider this server does not have.
func checkProvider(provider string) error {
	name := strings.ToUpper(strings.TrimSpace(provider))
	if name == "" || name == providerName {
		return nil
	}
	return fmt.Errorf("provider %q is not available: this server only serves %s", name, providerName)
}
