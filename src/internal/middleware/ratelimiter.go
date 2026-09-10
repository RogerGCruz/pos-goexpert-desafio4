package middleware

import (
	"net"
	"net/http"
	"strings"

	"github.com/rogergcruz/pos-goexpert-desafio4/internal/limiter"
)

const tooManyRequestsMessage = "you have reached the maximum number of requests or actions allowed within a certain time frame"

// RateLimiter cria um middleware HTTP que aplica as regras de negócio de l
// a cada requisição, identificando o solicitante por token (header API_KEY)
// ou, na ausência deste, pelo IP de origem.
func RateLimiter(l *limiter.Limiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := clientIP(r)
			token := r.Header.Get("API_KEY")

			allowed, err := l.Allow(r.Context(), ip, token)
			if err != nil {
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
			if !allowed {
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = w.Write([]byte(tooManyRequestsMessage))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// clientIP extrai o IP do solicitante, considerando o header X-Forwarded-For
// quando presente (ex.: atrás de proxy/load balancer).
func clientIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		parts := strings.Split(fwd, ",")
		return strings.TrimSpace(parts[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
