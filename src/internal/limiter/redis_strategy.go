package limiter

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

const blockedKeySuffix = ":blocked"

// RedisStrategy implementa Strategy usando Redis. É a estratégia obrigatória
// deste desafio, mas pode ser substituída por qualquer outra implementação
// de Strategy sem alterar a lógica de negócio em Limiter.
type RedisStrategy struct {
	client *redis.Client
}

// NewRedisStrategy cria uma estratégia de persistência baseada em Redis.
func NewRedisStrategy(client *redis.Client) *RedisStrategy {
	return &RedisStrategy{client: client}
}

// Increment implementa um contador de janela fixa: o TTL só é definido no
// primeiro incremento da janela, garantindo que a contagem expire após window.
func (r *RedisStrategy) Increment(ctx context.Context, key string, window time.Duration) (int64, error) {
	count, err := r.client.Incr(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	if count == 1 {
		if err := r.client.Expire(ctx, key, window).Err(); err != nil {
			return 0, err
		}
	}
	return count, nil
}

// IsBlocked verifica a existência da chave de bloqueio associada.
func (r *RedisStrategy) IsBlocked(ctx context.Context, key string) (bool, error) {
	exists, err := r.client.Exists(ctx, key+blockedKeySuffix).Result()
	if err != nil {
		return false, err
	}
	return exists > 0, nil
}

// Block cria a chave de bloqueio com expiração igual ao tempo de bloqueio.
func (r *RedisStrategy) Block(ctx context.Context, key string, duration time.Duration) error {
	return r.client.Set(ctx, key+blockedKeySuffix, 1, duration).Err()
}
