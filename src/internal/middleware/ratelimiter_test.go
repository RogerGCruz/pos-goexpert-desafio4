package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"github.com/rogergcruz/pos-goexpert-desafio4/internal/limiter"
	"github.com/rogergcruz/pos-goexpert-desafio4/internal/middleware"
)

const expectedBlockedBody = "you have reached the maximum number of requests or actions allowed within a certain time frame"

func newTestHandler(t *testing.T, cfg limiter.Config) http.Handler {
	t.Helper()

	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	t.Cleanup(mr.Close)

	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	l := limiter.New(limiter.NewRedisStrategy(client), cfg)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	return middleware.RateLimiter(l)(next)
}

func TestRateLimiterMiddleware_BlocksAfterLimitByIP(t *testing.T) {
	cfg := limiter.Config{IPLimit: 2, IPBlockDuration: 5 * time.Minute}
	handler := newTestHandler(t, cfg)

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "10.0.0.1:1234"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("request %d expected 200, got %d", i+1, rec.Code)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", rec.Code)
	}
	if rec.Body.String() != expectedBlockedBody {
		t.Fatalf("unexpected body: %q", rec.Body.String())
	}
}

func TestRateLimiterMiddleware_TokenOverridesIPLimit(t *testing.T) {
	cfg := limiter.Config{
		IPLimit:         1,
		IPBlockDuration: 5 * time.Minute,
		TokenRules: map[string]limiter.TokenRule{
			"super-token": {Limit: 3, BlockDuration: time.Minute},
		},
	}
	handler := newTestHandler(t, cfg)

	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "10.0.0.2:1234"
		req.Header.Set("API_KEY", "super-token")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("request %d with token expected 200, got %d", i+1, rec.Code)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.2:1234"
	req.Header.Set("API_KEY", "super-token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 after exceeding token limit, got %d", rec.Code)
	}
}
