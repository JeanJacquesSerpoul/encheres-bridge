package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"server_ai/internal/openrouter"
)

const testModel = "google/gemini-3.1-flash-lite"

// testAllowedModels are the models the tests may name besides the default.
var testAllowedModels = []string{"m1", "openai/gpt-4o"}

type mockAIClient struct {
	lastRequest *openrouter.ChatRequest
	chatResult  *openrouter.ChatResult
	chatErr     error
	models      []openrouter.Model
	modelsErr   error
}

func (m *mockAIClient) Chat(_ context.Context, req *openrouter.ChatRequest) (*openrouter.ChatResult, error) {
	m.lastRequest = req
	return m.chatResult, m.chatErr
}

func (m *mockAIClient) Models(context.Context) ([]openrouter.Model, error) {
	return m.models, m.modelsErr
}

// postChat runs one JSON body through a ChatHandler backed by c.
func postChat(t *testing.T, c *mockAIClient, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/chat", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	NewChatHandler(c, testModel, testAllowedModels).ServeHTTP(rr, req)
	return rr
}

func TestChatHandlerValidRequest(t *testing.T) {
	c := &mockAIClient{chatResult: &openrouter.ChatResult{Content: "ok", Model: "m1"}}

	rr := postChat(t, c, `{"text":"hello","model":"m1"}`)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
	var resp ChatResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if !resp.Success || resp.Response != "ok" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

// The card-recognition client may leave the model out: the configured default
// is what makes the request work anyway.
func TestChatHandlerFallsBackToDefaultModel(t *testing.T) {
	c := &mockAIClient{chatResult: &openrouter.ChatResult{Content: "{}", Model: testModel}}

	rr := postChat(t, c, `{"text":"read the cards","image":"data:image/jpeg;base64,AAAA"}`)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
	if c.lastRequest.Model != testModel {
		t.Fatalf("expected default model %q, got %q", testModel, c.lastRequest.Model)
	}
	if c.lastRequest.Image != "data:image/jpeg;base64,AAAA" {
		t.Fatalf("image not forwarded: %q", c.lastRequest.Image)
	}
}

func TestChatHandlerKeepsRequestedModel(t *testing.T) {
	c := &mockAIClient{chatResult: &openrouter.ChatResult{Content: "ok", Model: "openai/gpt-4o"}}

	postChat(t, c, `{"text":"hi","model":"openai/gpt-4o"}`)

	if c.lastRequest.Model != "openai/gpt-4o" {
		t.Fatalf("unexpected model: %q", c.lastRequest.Model)
	}
}

func TestChatHandlerRequiresText(t *testing.T) {
	rr := postChat(t, &mockAIClient{}, `{"model":"m1"}`)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

// An aiproxy client naming OPENROUTER must keep working; any other provider is
// refused rather than silently served by OpenRouter.
func TestChatHandlerAcceptsOpenRouterProvider(t *testing.T) {
	c := &mockAIClient{chatResult: &openrouter.ChatResult{Content: "ok", Model: "m1"}}

	rr := postChat(t, c, `{"text":"hi","model":"m1","provider":"openrouter"}`)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
}

func TestChatHandlerRejectsOtherProvider(t *testing.T) {
	rr := postChat(t, &mockAIClient{}, `{"text":"hi","model":"m1","provider":"OLLAMA"}`)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "OLLAMA") {
		t.Fatalf("expected the refused provider to be named: %s", rr.Body.String())
	}
}

func TestChatHandlerMasksUpstreamError(t *testing.T) {
	c := &mockAIClient{chatErr: errors.New("sensitive upstream detail")}

	rr := postChat(t, c, `{"text":"hi","model":"m1"}`)

	if rr.Code != http.StatusBadGateway {
		t.Fatalf("expected status %d, got %d", http.StatusBadGateway, rr.Code)
	}
	if strings.Contains(rr.Body.String(), "sensitive upstream detail") {
		t.Fatalf("response leaked upstream details: %s", rr.Body.String())
	}
}

func TestChatHandlerReportsTimeout(t *testing.T) {
	c := &mockAIClient{chatErr: context.DeadlineExceeded}

	rr := postChat(t, c, `{"text":"hi","model":"m1"}`)

	if rr.Code != http.StatusGatewayTimeout {
		t.Fatalf("expected status %d, got %d", http.StatusGatewayTimeout, rr.Code)
	}
}

func TestChatHandlerRejectsLargeBody(t *testing.T) {
	tooLarge := strings.Repeat("a", maxChatRequestBodyBytes+1)

	rr := postChat(t, &mockAIClient{}, `{"text":"`+tooLarge+`","model":"m1"}`)

	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected status %d, got %d", http.StatusRequestEntityTooLarge, rr.Code)
	}
}

// The key must not pay for whatever model a stranger names.
func TestChatHandlerRejectsModelOutsideAllowList(t *testing.T) {
	c := &mockAIClient{}

	rr := postChat(t, c, `{"text":"hi","model":"anthropic/claude-opus"}`)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
	if c.lastRequest != nil {
		t.Fatal("refused model still reached the upstream")
	}
}

func TestChatHandlerRejectsLongText(t *testing.T) {
	long := strings.Repeat("a", maxTextBytes+1)

	rr := postChat(t, &mockAIClient{}, `{"text":"`+long+`","model":"m1"}`)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

// A remote URL would have OpenRouter fetch any address a caller names.
func TestChatHandlerRejectsRemoteImage(t *testing.T) {
	for _, image := range []string{
		"https://example.com/hand.jpg",
		"http://169.254.169.254/latest/meta-data",
		"data:text/html;base64,AAAA",
	} {
		rr := postChat(t, &mockAIClient{}, `{"text":"hi","image":"`+image+`"}`)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("image %q: expected status %d, got %d", image, http.StatusBadRequest, rr.Code)
		}
	}
}

func TestChatHandlerAcceptsPNGImage(t *testing.T) {
	c := &mockAIClient{chatResult: &openrouter.ChatResult{Content: "{}", Model: testModel}}

	rr := postChat(t, c, `{"text":"hi","image":"data:image/png;base64,AAAA"}`)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
}

func TestChatHandlerRejectsNonDataURIPDF(t *testing.T) {
	rr := postChat(t, &mockAIClient{}, `{"text":"hi","pdf":"https://example.com/a.pdf"}`)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

// The decoder's own message names fields and offsets; the client gets a
// generic one.
func TestChatHandlerHidesJSONDecodeDetail(t *testing.T) {
	rr := postChat(t, &mockAIClient{}, `{"text":"hi","secret_field":1}`)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
	if strings.Contains(rr.Body.String(), "secret_field") {
		t.Fatalf("response leaked decoder detail: %s", rr.Body.String())
	}
}

func TestModelsHandlerReturnsModels(t *testing.T) {
	c := &mockAIClient{models: []openrouter.Model{{ID: testModel, Name: "Gemini 3.1 Flash Lite"}}}

	rr := httptest.NewRecorder()
	NewModelsHandler(c).ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/models", nil))

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
	var resp ModelsResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Count != 1 || resp.Models[0].ID != testModel {
		t.Fatalf("unexpected models response: %+v", resp)
	}
}

func TestModelsHandlerRejectsOtherProvider(t *testing.T) {
	rr := httptest.NewRecorder()
	NewModelsHandler(&mockAIClient{}).ServeHTTP(rr,
		httptest.NewRequest(http.MethodGet, "/api/models?provider=DEEPSEEK", nil))

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestModelsHandlerMasksUpstreamError(t *testing.T) {
	c := &mockAIClient{modelsErr: errors.New("internal provider info")}

	rr := httptest.NewRecorder()
	NewModelsHandler(c).ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/models", nil))

	if rr.Code != http.StatusBadGateway {
		t.Fatalf("expected status %d, got %d", http.StatusBadGateway, rr.Code)
	}
	if strings.Contains(rr.Body.String(), "internal provider info") {
		t.Fatalf("response leaked upstream details: %s", rr.Body.String())
	}
}
