package middleware

import (
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// RateLimiter is an in-memory token bucket per client IP. It is enough for a
// single host (PLAN §8); state resets on restart.
type RateLimiter struct {
	mu      sync.Mutex
	rate    float64 // tokens per second
	burst   float64
	buckets map[string]*bucket
	now     func() time.Time
	// OnLimited writes the 429 response; defaults to a JSON error body.
	OnLimited func(w http.ResponseWriter, r *http.Request, retryAfter time.Duration)
}

type bucket struct {
	tokens float64
	last   time.Time
}

// NewRateLimiter allows perMinute requests per minute with the given burst.
func NewRateLimiter(perMinute, burst int) *RateLimiter {
	if perMinute <= 0 {
		perMinute = 120
	}
	if burst <= 0 {
		burst = perMinute / 2
		if burst < 1 {
			burst = 1
		}
	}
	return &RateLimiter{
		rate:    float64(perMinute) / 60,
		burst:   float64(burst),
		buckets: map[string]*bucket{},
		now:     time.Now,
	}
}

// Allow takes one token for key and reports how long to wait if none is left.
func (l *RateLimiter) Allow(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	b, ok := l.buckets[key]
	if !ok {
		b = &bucket{tokens: l.burst, last: now}
		l.buckets[key] = b
	}
	b.tokens = math.Min(l.burst, b.tokens+now.Sub(b.last).Seconds()*l.rate)
	b.last = now
	if b.tokens >= 1 {
		b.tokens--
		return true, 0
	}
	wait := time.Duration((1 - b.tokens) / l.rate * float64(time.Second))
	return false, wait
}

// Sweep drops buckets that have refilled completely.
func (l *RateLimiter) Sweep() {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	full := time.Duration(l.burst / l.rate * float64(time.Second))
	for k, b := range l.buckets {
		if now.Sub(b.last) > full {
			delete(l.buckets, k)
		}
	}
}

// Middleware limits requests whose path starts with prefix.
func (l *RateLimiter) Middleware(prefix string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.HasPrefix(r.URL.Path, prefix) {
				next.ServeHTTP(w, r)
				return
			}
			ok, wait := l.Allow(IP(r))
			if ok {
				next.ServeHTTP(w, r)
				return
			}
			secs := int(math.Ceil(wait.Seconds()))
			if secs < 1 {
				secs = 1
			}
			w.Header().Set("Retry-After", strconv.Itoa(secs))
			if l.OnLimited != nil {
				l.OnLimited(w, r, wait)
				return
			}
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":{"code":"rate_limited","message":"Too many requests. Try again shortly."}}` + "\n"))
		})
	}
}
