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

const (
	// The client shrinks its photos to 2560 px before sending them: a few MiB
	// once base64-encoded.
	maxChatRequestBodyBytes = 10 << 20 // 10 MiB
	// The card prompts are well under a kilobyte; this only bounds abuse.
	maxTextBytes = 16 << 10 // 16 KiB
)

// Images must travel inline: a remote URL would have OpenRouter fetch whatever
// a caller names, on this server's key.
var allowedImagePrefixes = []string{
	"data:image/jpeg;base64,",
	"data:image/png;base64,",
	"data:image/webp;base64,",
	"data:image/gif;base64,",
}

const pdfPrefix = "data:application/pdf;base64,"

// chatRequest is the inbound body of POST /api/chat.
type chatRequest struct {
	Text     string `json:"text"`
	Image    string `json:"image,omitempty"`    // base64 data-URI
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
	client        aiClient
	defaultModel  string
	allowedModels map[string]bool
}

// NewChatHandler creates a ChatHandler falling back to defaultModel whenever a
// request leaves the model out, and refusing any model outside allowedModels
// (defaultModel is always allowed).
func NewChatHandler(client aiClient, defaultModel string, allowedModels []string) *ChatHandler {
	allowed := map[string]bool{defaultModel: true}
	for _, m := range allowedModels {
		allowed[m] = true
	}
	return &ChatHandler{client: client, defaultModel: defaultModel, allowedModels: allowed}
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
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			writeJSON(w, http.StatusRequestEntityTooLarge, ChatResponse{
				Success: false,
				Error:   "request body too large",
			})
			return
		}
		log.Printf("chat invalid JSON body: %v", err)
		writeJSON(w, http.StatusBadRequest, ChatResponse{
			Success: false,
			Error:   "invalid JSON body",
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
	if len(req.Text) > maxTextBytes {
		writeJSON(w, http.StatusBadRequest, ChatResponse{
			Success: false,
			Error:   fmt.Sprintf("text field exceeds %d bytes", maxTextBytes),
		})
		return
	}
	if req.Image != "" && !hasAnyPrefix(req.Image, allowedImagePrefixes) {
		writeJSON(w, http.StatusBadRequest, ChatResponse{
			Success: false,
			Error:   "image must be a base64 data-URI (jpeg, png, webp or gif)",
		})
		return
	}
	if req.PDF != "" && !strings.HasPrefix(req.PDF, pdfPrefix) {
		writeJSON(w, http.StatusBadRequest, ChatResponse{
			Success: false,
			Error:   "pdf must be a base64 data-URI (" + pdfPrefix + "...)",
		})
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
	if !h.allowedModels[model] {
		writeJSON(w, http.StatusBadRequest, ChatResponse{
			Success: false,
			Error:   fmt.Sprintf("model %q is not allowed", model),
		})
		return
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

func hasAnyPrefix(s string, prefixes []string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}
