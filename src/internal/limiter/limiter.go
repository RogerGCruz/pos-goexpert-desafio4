package limiter

import (
	"context"
	"errors"
	"time"
)

// ErrLimiterUnavailable indica falha ao consultar a estratégia de persistência.
var ErrLimiterUnavailable = errors.New("rate limiter: persistence strategy unavailable")

// TokenRule define o limite e o tempo de bloqueio específicos de um token.
type TokenRule struct {
	Limit         int64
	BlockDuration time.Duration
}

// Config contém os parâmetros de negócio do Rate Limiter.
type Config struct {
	// IPLimit é o número máximo de requisições por segundo por IP.
	IPLimit int64
	// IPBlockDuration é o tempo de bloqueio aplicado a um IP que excede o limite.
	IPBlockDuration time.Duration

	// DefaultTokenLimit é usado quando o token não possui regra específica em TokenRules.
	DefaultTokenLimit int64
	// DefaultTokenBlockDuration é o bloqueio padrão para tokens sem regra específica.
	DefaultTokenBlockDuration time.Duration

	// TokenRules mapeia token -> regra específica de limite/bloqueio.
	// Regras de token sempre têm precedência sobre o limite por IP.
	TokenRules map[string]TokenRule
}

// Limiter concentra a lógica de negócio do Rate Limiter, desacoplada do
// transporte HTTP (middleware) e do mecanismo de persistência (Strategy).
type Limiter struct {
	strategy Strategy
	cfg      Config
}

// New cria um novo Limiter com a estratégia de persistência e configuração informadas.
func New(strategy Strategy, cfg Config) *Limiter {
	return &Limiter{strategy: strategy, cfg: cfg}
}

// Allow verifica se uma requisição identificada por ip e/ou token pode prosseguir.
// Quando token não é vazio, suas regras se sobrepõem às regras de IP (precedência).
func (l *Limiter) Allow(ctx context.Context, ip, token string) (bool, error) {
	key, limit, blockDuration := l.resolveRule(ip, token)

	blocked, err := l.strategy.IsBlocked(ctx, key)
	if err != nil {
		return false, ErrLimiterUnavailable
	}
	if blocked {
		return false, nil
	}

	count, err := l.strategy.Increment(ctx, key, time.Second)
	if err != nil {
		return false, ErrLimiterUnavailable
	}

	if count > limit {
		if err := l.strategy.Block(ctx, key, blockDuration); err != nil {
			return false, ErrLimiterUnavailable
		}
		return false, nil
	}

	return true, nil
}

// resolveRule determina a chave de controle e os parâmetros de limite/bloqueio
// aplicáveis, priorizando o token sobre o IP quando presente.
func (l *Limiter) resolveRule(ip, token string) (key string, limit int64, blockDuration time.Duration) {
	if token != "" {
		if rule, ok := l.cfg.TokenRules[token]; ok {
			return "token:" + token, rule.Limit, rule.BlockDuration
		}
		return "token:" + token, l.cfg.DefaultTokenLimit, l.cfg.DefaultTokenBlockDuration
	}
	return "ip:" + ip, l.cfg.IPLimit, l.cfg.IPBlockDuration
}
