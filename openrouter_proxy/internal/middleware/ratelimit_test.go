package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func limitedHandler(rl *RateLimiter) http.Handler {
	return rl.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
}

func hit(h http.Handler, remoteAddr string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/chat", nil)
	req.RemoteAddr = remoteAddr
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr
}

func TestRateLimiterRefusesBeyondBurst(t *testing.T) {
	now := time.Unix(0, 0)
	h := limitedHandler(newRateLimiter(6, 2, func() time.Time { return now }))

	for i := 0; i < 2; i++ {
		if rr := hit(h, "203.0.113.1:1234"); rr.Code != http.StatusOK {
			t.Fatalf("request %d: expected 200, got %d", i, rr.Code)
		}
	}
	rr := hit(h, "203.0.113.1:1234")
	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", rr.Code)
	}
	if rr.Header().Get("Retry-After") != "10" {
		t.Fatalf("expected Retry-After 10, got %q", rr.Header().Get("Retry-After"))
	}

	// 6 per minute: one token back every 10 seconds.
	now = now.Add(10 * time.Second)
	if rr := hit(h, "203.0.113.1:1234"); rr.Code != http.StatusOK {
		t.Fatalf("expected 200 after refill, got %d", rr.Code)
	}
}

func TestRateLimiterCountsIPsSeparately(t *testing.T) {
	now := time.Unix(0, 0)
	h := limitedHandler(newRateLimiter(1, 1, func() time.Time { return now }))

	if rr := hit(h, "203.0.113.1:1"); rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	// Another port of the same host shares its bucket.
	if rr := hit(h, "203.0.113.1:2"); rr.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", rr.Code)
	}
	if rr := hit(h, "203.0.113.2:1"); rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for another IP, got %d", rr.Code)
	}
}

func TestRateLimiterSweepsIdleVisitors(t *testing.T) {
	now := time.Unix(0, 0)
	rl := newRateLimiter(1, 1, func() time.Time { return now })
	hit(limitedHandler(rl), "203.0.113.1:1")

	now = now.Add(idleTTL + time.Second)
	rl.sweep()

	if len(rl.visitors) != 0 {
		t.Fatalf("expected idle visitor to be swept, %d left", len(rl.visitors))
	}
}
