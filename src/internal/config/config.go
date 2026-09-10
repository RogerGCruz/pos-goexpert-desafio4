package config

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/rogergcruz/pos-goexpert-desafio4/internal/limiter"
)

// Config agrega toda a configuração da aplicação, carregada de variáveis
// de ambiente (ou arquivo .env na raiz do projeto).
type Config struct {
	ServerPort string

	RedisAddr     string
	RedisPassword string
	RedisDB       int

	Limiter limiter.Config
}

// Load lê o arquivo .env (se existir) e as variáveis de ambiente, retornando
// a configuração completa da aplicação com valores padrão quando aplicável.
func Load() (Config, error) {
	_ = godotenv.Load()

	cfg := Config{
		ServerPort:    getEnv("SERVER_PORT", "8080"),
		RedisAddr:     getEnv("REDIS_HOST", "localhost") + ":" + getEnv("REDIS_PORT", "6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       getEnvInt("REDIS_DB", 0),
	}

	cfg.Limiter = limiter.Config{
		IPLimit:                   getEnvInt64("RATE_LIMIT_IP_MAX_REQUESTS", 10),
		IPBlockDuration:           getEnvSeconds("RATE_LIMIT_IP_BLOCK_DURATION_SECONDS", 300),
		DefaultTokenLimit:         getEnvInt64("RATE_LIMIT_TOKEN_MAX_REQUESTS", 100),
		DefaultTokenBlockDuration: getEnvSeconds("RATE_LIMIT_TOKEN_BLOCK_DURATION_SECONDS", 300),
		TokenRules:                parseTokenRules(getEnv("RATE_LIMIT_TOKEN_RULES", "")),
	}

	return cfg, nil
}

// parseTokenRules interpreta regras específicas por token no formato
// "token:limite:bloqueioSegundos;token2:limite:bloqueioSegundos".
func parseTokenRules(raw string) map[string]limiter.TokenRule {
	rules := make(map[string]limiter.TokenRule)
	if raw == "" {
		return rules
	}
	for _, entry := range strings.Split(raw, ";") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		parts := strings.Split(entry, ":")
		if len(parts) != 3 {
			continue
		}
		token := strings.TrimSpace(parts[0])
		limit, err := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)
		if err != nil {
			continue
		}
		blockSeconds, err := strconv.ParseInt(strings.TrimSpace(parts[2]), 10, 64)
		if err != nil {
			continue
		}
		rules[token] = limiter.TokenRule{
			Limit:         limit,
			BlockDuration: time.Duration(blockSeconds) * time.Second,
		}
	}
	return rules
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}

func getEnvInt64(key string, fallback int64) int64 {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.ParseInt(v, 10, 64); err == nil {
			return i
		}
	}
	return fallback
}

func getEnvSeconds(key string, fallbackSeconds int64) time.Duration {
	return time.Duration(getEnvInt64(key, fallbackSeconds)) * time.Second
}
