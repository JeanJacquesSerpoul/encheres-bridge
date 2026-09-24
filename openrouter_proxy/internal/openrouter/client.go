// Package openrouter is the sole upstream of this server: the OpenRouter chat
// and models APIs, called with the key held by the process.
package openrouter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const baseURL = "https://openrouter.ai/api/v1"

// Headers OpenRouter shows on its activity page to name the calling app.
const (
	refererHeader = "https://github.com/jjserpoul/encheres-bridge"
	titleHeader   = "encheres-bridge"
)

// ChatRequest is one prompt, with the optional image the card recognition
// relies on and the optional PDF OpenRouter also accepts.
type ChatRequest struct {
	Text  string // prompt
	Image string // base64 data-URI
	PDF   string // base64 data-URI (data:application/pdf;base64,...)
	Model string // OpenRouter model ID
}

// ChatResult holds the successful response data returned to the handler.
type ChatResult struct {
	Content          string
	Model            string
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

// Model describes a single AI model available on OpenRouter.
type Model struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Description   string `json:"description,omitempty"`
	ContextLength int    `json:"context_length,omitempty"`
	PricingPrompt string `json:"pricing_prompt,omitempty"`
	PricingOutput string `json:"pricing_completion,omitempty"`
}

// --- OpenRouter wire types ---

type orContentPart struct {
	Type     string      `json:"type"`
	Text     string      `json:"text,omitempty"`
	ImageURL *orImageURL `json:"image_url,omitempty"`
	File     *orFile     `json:"file,omitempty"`
}

type orImageURL struct {
	URL    string `json:"url"`
	Detail string `json:"detail,omitempty"`
}

type orFile struct {
	Filename string `json:"filename"`
	FileData string `json:"file_data"`
}

type orMessage struct {
	Role    string          `json:"role"`
	Content []orContentPart `json:"content"`
}

type orRequest struct {
	Model     string      `json:"model"`
	Messages  []orMessage `json:"messages"`
	MaxTokens int         `json:"max_tokens"`
}

type orChoice struct {
	Message struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"message"`
	FinishReason string `json:"finish_reason"`
}

type orUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type orError struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

type orResponse struct {
	ID      string     `json:"id"`
	Model   string     `json:"model"`
	Choices []orChoice `json:"choices"`
	Usage   orUsage    `json:"usage"`
	Error   *orError   `json:"error,omitempty"`
}

type orModelsResponse struct {
	Data  []Model  `json:"data"`
	Error *orError `json:"error,omitempty"`
}

// Client sends requests to the OpenRouter API.
type Client struct {
	apiKey     string
	maxTokens  int
	httpClient *http.Client
}

// NewClient creates a Client with the given API key, token cap, and timeout.
func NewClient(apiKey string, maxTokens int, timeout time.Duration) *Client {
	return &Client{
		apiKey:    apiKey,
		maxTokens: maxTokens,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// Models fetches the list of available models from OpenRouter.
func (c *Client) Models(ctx context.Context) ([]Model, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/models", nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	c.setHeaders(req.Header)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	var result orModelsResponse
	if resp.StatusCode != http.StatusOK {
		if err := json.Unmarshal(body, &result); err == nil && result.Error != nil {
			return nil, fmt.Errorf("openrouter models error %d: %s", result.Error.Code, result.Error.Message)
		}
		return nil, fmt.Errorf("openrouter models returned status %d", resp.StatusCode)
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("unmarshal models response: %w", err)
	}

	if result.Error != nil {
		return nil, fmt.Errorf("openrouter error %d: %s", result.Error.Code, result.Error.Message)
	}

	return result.Data, nil
}

// Chat sends a prompt (with optional image or PDF) to OpenRouter and returns the result.
func (c *Client) Chat(ctx context.Context, req *ChatRequest) (*ChatResult, error) {
	// Build content parts
	parts := []orContentPart{
		{Type: "text", Text: req.Text},
	}
	if req.Image != "" {
		parts = append(parts, orContentPart{
			Type:     "image_url",
			ImageURL: &orImageURL{URL: req.Image, Detail: "auto"},
		})
	}
	if req.PDF != "" {
		parts = append(parts, orContentPart{
			Type: "file",
			File: &orFile{Filename: "document.pdf", FileData: req.PDF},
		})
	}

	payload := orRequest{
		Model:     req.Model,
		MaxTokens: c.maxTokens,
		Messages: []orMessage{
			{Role: "user", Content: parts},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	c.setHeaders(httpReq.Header)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err // includes context deadline exceeded (timeout)
	}
	defer resp.Body.Close()

	// Limit response to 10 MB
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	var orResp orResponse
	if err := json.Unmarshal(respBody, &orResp); err != nil {
		return nil, fmt.Errorf("unmarshal response (status %d): %w", resp.StatusCode, err)
	}

	if orResp.Error != nil {
		return nil, fmt.Errorf("openrouter error %d: %s", orResp.Error.Code, orResp.Error.Message)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("openrouter returned status %d", resp.StatusCode)
	}

	if len(orResp.Choices) == 0 {
		return nil, fmt.Errorf("openrouter returned no choices")
	}

	return &ChatResult{
		Content:          orResp.Choices[0].Message.Content,
		Model:            orResp.Model,
		PromptTokens:     orResp.Usage.PromptTokens,
		CompletionTokens: orResp.Usage.CompletionTokens,
		TotalTokens:      orResp.Usage.TotalTokens,
	}, nil
}

func (c *Client) setHeaders(h http.Header) {
	h.Set("Authorization", "Bearer "+c.apiKey)
	h.Set("HTTP-Referer", refererHeader)
	h.Set("X-Title", titleHeader)
}
