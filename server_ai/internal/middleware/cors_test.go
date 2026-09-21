package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewCORSUsesConfiguredOrigins(t *testing.T) {
	mw := NewCORS([]string{"https://allowed.example"})
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodOptions, "/api/chat", nil)
	req.Header.Set("Origin", "https://allowed.example")
	req.Header.Set("Access-Control-Request-Method", "POST")
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "https://allowed.example" {
		t.Fatalf("expected Access-Control-Allow-Origin to be configured origin, got %q", got)
	}
}
