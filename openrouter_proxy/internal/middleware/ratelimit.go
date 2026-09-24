package middleware

import (
	"math"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// idleTTL is how long a client's bucket survives without a request before the
// sweeper forgets it; long enough that a full bucket is all it loses.
const idleTTL = 10 * time.Minute

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// RateLimiter keeps one token bucket per client IP, so that a single caller
// cannot drain the OpenRouter credit behind the key.
type RateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitor
	limit    rate.Limit
	burst    int
	now      func() time.Time
}

// NewRateLimiter allows perMinute requests per minute and IP, with bursts of
// up to burst requests. It starts a sweeper that lives as long as the process.
func NewRateLimiter(perMinute, burst int) *RateLimiter {
	rl := newRateLimiter(perMinute, burst, time.Now)
	go func() {
		for range time.Tick(time.Minute) {
			rl.sweep()
		}
	}()
	return rl
}

func newRateLimiter(perMinute, burst int, now func() time.Time) *RateLimiter {
	return &RateLimiter{
		visitors: make(map[string]*visitor),
		limit:    rate.Limit(float64(perMinute) / 60),
		burst:    burst,
		now:      now,
	}
}

// Handler refuses with 429 the requests beyond the client's allowance.
func (rl *RateLimiter) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		res := rl.reserve(clientIP(r))
		if delay := res.DelayFrom(rl.now()); delay > 0 {
			res.CancelAt(rl.now())
			w.Header().Set("Retry-After", strconv.Itoa(int(math.Ceil(delay.Seconds()))))
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"success":false,"error":"rate limit exceeded"}`)) //nolint:errcheck
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (rl *RateLimiter) reserve(ip string) *rate.Reservation {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	now := rl.now()
	v, ok := rl.visitors[ip]
	if !ok {
		v = &visitor{limiter: rate.NewLimiter(rl.limit, rl.burst)}
		rl.visitors[ip] = v
	}
	v.lastSeen = now
	return v.limiter.ReserveN(now, 1)
}

func (rl *RateLimiter) sweep() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	cutoff := rl.now().Add(-idleTTL)
	for ip, v := range rl.visitors {
		if v.lastSeen.Before(cutoff) {
			delete(rl.visitors, ip)
		}
	}
}

// clientIP is RemoteAddr without its port. Behind a trusted proxy, chi's
// RealIP has already replaced RemoteAddr with the forwarded address.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
