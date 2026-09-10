package main

import (
	"log"
	"net/http"

	"github.com/redis/go-redis/v9"

	"github.com/rogergcruz/pos-goexpert-desafio4/internal/config"
	"github.com/rogergcruz/pos-goexpert-desafio4/internal/limiter"
	"github.com/rogergcruz/pos-goexpert-desafio4/internal/middleware"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})
	defer redisClient.Close()

	strategy := limiter.NewRedisStrategy(redisClient)
	rateLimiter := limiter.New(strategy, cfg.Limiter)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("request accepted"))
	})

	handler := middleware.Logging(middleware.RateLimiter(rateLimiter)(mux))

	addr := ":" + cfg.ServerPort
	log.Printf("server listening on %s", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
