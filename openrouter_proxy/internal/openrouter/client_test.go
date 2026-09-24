package openrouter

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

// stub replaces the transport of c so no request leaves the machine.
func stub(c *Client, fn roundTripFunc) {
	c.httpClient = &http.Client{Transport: fn}
}

func TestModelsReturnsStatusErrorWhenNonJSONBody(t *testing.T) {
	c := NewClient("k", 100, 0)
	stub(c, func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusBadGateway,
			Body:       io.NopCloser(strings.NewReader("bad gateway")),
			Header:     make(http.Header),
		}, nil
	})

	_, err := c.Models(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "returned status 502") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestModelsReturnsProviderErrorWhenAvailable(t *testing.T) {
	c := NewClient("k", 100, 0)
	stub(c, func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusTooManyRequests,
			Body:       io.NopCloser(strings.NewReader(`{"error":{"code":429,"message":"rate limited"}}`)),
			Header:     make(http.Header),
		}, nil
	})

	_, err := c.Models(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "rate limited") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// The card recognition sends the photo as a data-URI: it must reach OpenRouter
// as an image_url part next to the prompt, or the model reads nothing.
func TestChatSendsImagePart(t *testing.T) {
	c := NewClient("k", 4096, 0)
	var payload orRequest
	stub(c, func(req *http.Request) (*http.Response, error) {
		body, _ := io.ReadAll(req.Body)
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("unmarshal request: %v", err)
		}
		if got := req.Header.Get("Authorization"); got != "Bearer k" {
			t.Fatalf("unexpected Authorization header: %q", got)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body: io.NopCloser(strings.NewReader(
				`{"model":"google/gemini-3.1-flash-lite","choices":[{"message":{"content":"{}"}}],` +
					`"usage":{"prompt_tokens":3,"completion_tokens":1,"total_tokens":4}}`)),
			Header: make(http.Header),
		}, nil
	})

	res, err := c.Chat(context.Background(), &ChatRequest{
		Text:  "read the cards",
		Image: "data:image/jpeg;base64,AAAA",
		Model: "google/gemini-3.1-flash-lite",
	})
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}
	if res.Model != "google/gemini-3.1-flash-lite" || res.TotalTokens != 4 {
		t.Fatalf("unexpected result: %+v", res)
	}
	if payload.MaxTokens != 4096 {
		t.Fatalf("expected max_tokens from the client cap, got %d", payload.MaxTokens)
	}
	parts := payload.Messages[0].Content
	if len(parts) != 2 {
		t.Fatalf("expected text and image parts, got %d: %+v", len(parts), parts)
	}
	if parts[0].Type != "text" || parts[0].Text != "read the cards" {
		t.Fatalf("unexpected text part: %+v", parts[0])
	}
	if parts[1].Type != "image_url" || parts[1].ImageURL.URL != "data:image/jpeg;base64,AAAA" {
		t.Fatalf("unexpected image part: %+v", parts[1])
	}
}

func TestChatReportsUpstreamError(t *testing.T) {
	c := NewClient("k", 100, 0)
	stub(c, func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusPaymentRequired,
			Body:       io.NopCloser(strings.NewReader(`{"error":{"code":402,"message":"no credit"}}`)),
			Header:     make(http.Header),
		}, nil
	})

	_, err := c.Chat(context.Background(), &ChatRequest{Text: "hi", Model: "m1"})
	if err == nil || !strings.Contains(err.Error(), "no credit") {
		t.Fatalf("unexpected error: %v", err)
	}
}
