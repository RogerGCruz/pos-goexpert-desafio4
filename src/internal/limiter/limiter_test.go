package limiter_test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"github.com/rogerioperoni/pos-goexpert-desafio4/internal/limiter"
)

// newTestLimiter cria um Limiter com RedisStrategy apontando para um Redis
// em memória (miniredis), isolado por teste.
func newTestLimiter(t *testing.T, cfg limiter.Config) (*limiter.Limiter, *miniredis.Miniredis) {
	t.Helper()

	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	t.Cleanup(mr.Close)

	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	strategy := limiter.NewRedisStrategy(client)
	return limiter.New(strategy, cfg), mr
}

func TestAllow_IPWithinLimit(t *testing.T) {
	cfg := limiter.Config{IPLimit: 3, IPBlockDuration: 5 * time.Minute}
	l, _ := newTestLimiter(t, cfg)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		allowed, err := l.Allow(ctx, "1.1.1.1", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !allowed {
			t.Fatalf("request %d should be allowed", i+1)
		}
	}
}

func TestAllow_IPExceedsLimit(t *testing.T) {
	cfg := limiter.Config{IPLimit: 2, IPBlockDuration: 5 * time.Minute}
	l, _ := newTestLimiter(t, cfg)
	ctx := context.Background()

	for i := 0; i < 2; i++ {
		if allowed, err := l.Allow(ctx, "2.2.2.2", ""); err != nil || !allowed {
			t.Fatalf("request %d should be allowed, got allowed=%v err=%v", i+1, allowed, err)
		}
	}

	allowed, err := l.Allow(ctx, "2.2.2.2", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if allowed {
		t.Fatal("3rd request should be blocked")
	}
}

func TestAllow_IPBlockedUntilDurationExpires(t *testing.T) {
	cfg := limiter.Config{IPLimit: 1, IPBlockDuration: 5 * time.Minute}
	l, mr := newTestLimiter(t, cfg)
	ctx := context.Background()

	if allowed, _ := l.Allow(ctx, "3.3.3.3", ""); !allowed {
		t.Fatal("1st request should be allowed")
	}
	if allowed, _ := l.Allow(ctx, "3.3.3.3", ""); allowed {
		t.Fatal("2nd request should be blocked")
	}

	// Ainda dentro do período de bloqueio: continua bloqueado mesmo
	// avançando o tempo de janela de contagem (1s).
	mr.FastForward(2 * time.Second)
	if allowed, _ := l.Allow(ctx, "3.3.3.3", ""); allowed {
		t.Fatal("request during block period should remain blocked")
	}

	// Após expirar o bloqueio, novas requisições voltam a ser permitidas.
	mr.FastForward(5 * time.Minute)
	if allowed, err := l.Allow(ctx, "3.3.3.3", ""); err != nil || !allowed {
		t.Fatalf("request after block expiration should be allowed, got allowed=%v err=%v", allowed, err)
	}
}

func TestAllow_TokenPrecedenceOverIP(t *testing.T) {
	cfg := limiter.Config{
		IPLimit:         1,
		IPBlockDuration: 5 * time.Minute,
		TokenRules: map[string]limiter.TokenRule{
			"abc123": {Limit: 5, BlockDuration: time.Minute},
		},
	}
	l, _ := newTestLimiter(t, cfg)
	ctx := context.Background()

	// Mesmo IP (limite 1) e um token com limite 5: usando o token, deve
	// permitir mais de 1 requisição, comprovando a precedência token > IP.
	for i := 0; i < 5; i++ {
		allowed, err := l.Allow(ctx, "4.4.4.4", "abc123")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !allowed {
			t.Fatalf("request %d with token should be allowed under token limit", i+1)
		}
	}

	allowed, err := l.Allow(ctx, "4.4.4.4", "abc123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if allowed {
		t.Fatal("6th request with token should exceed token limit and be blocked")
	}
}

func TestAllow_TokenWithoutSpecificRuleUsesDefault(t *testing.T) {
	cfg := limiter.Config{
		IPLimit:                   1,
		IPBlockDuration:           5 * time.Minute,
		DefaultTokenLimit:         2,
		DefaultTokenBlockDuration: time.Minute,
		TokenRules:                map[string]limiter.TokenRule{},
	}
	l, _ := newTestLimiter(t, cfg)
	ctx := context.Background()

	for i := 0; i < 2; i++ {
		if allowed, err := l.Allow(ctx, "5.5.5.5", "unknown-token"); err != nil || !allowed {
			t.Fatalf("request %d should be allowed under default token limit", i+1)
		}
	}

	if allowed, _ := l.Allow(ctx, "5.5.5.5", "unknown-token"); allowed {
		t.Fatal("3rd request should exceed default token limit")
	}
}

func TestAllow_DifferentIPsAreIndependent(t *testing.T) {
	cfg := limiter.Config{IPLimit: 1, IPBlockDuration: 5 * time.Minute}
	l, _ := newTestLimiter(t, cfg)
	ctx := context.Background()

	if allowed, _ := l.Allow(ctx, "6.6.6.6", ""); !allowed {
		t.Fatal("IP A first request should be allowed")
	}
	if allowed, _ := l.Allow(ctx, "6.6.6.6", ""); allowed {
		t.Fatal("IP A second request should be blocked")
	}
	if allowed, _ := l.Allow(ctx, "7.7.7.7", ""); !allowed {
		t.Fatal("IP B should not be affected by IP A block")
	}
}
