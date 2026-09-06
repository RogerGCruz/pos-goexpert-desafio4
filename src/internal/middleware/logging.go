package middleware

import (
	"log"
	"net/http"
	"time"
)

// statusRecorder captura o status code e o tamanho da resposta para fins de log.
type statusRecorder struct {
	http.ResponseWriter
	statusCode int
	size       int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	n, err := r.ResponseWriter.Write(b)
	r.size += n
	return n, err
}

// Logging registra método, caminho, IP, token (se houver), status, tamanho
// da resposta e duração de cada requisição processada pela API.
func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(rec, r)

		log.Printf("method=%s path=%s ip=%s token=%s status=%d size=%d duration=%s",
			r.Method, r.URL.Path, clientIP(r), maskToken(r.Header.Get("API_KEY")), rec.statusCode, rec.size, time.Since(start))
	})
}

// maskToken evita expor o token completo nos logs.
func maskToken(token string) string {
	if token == "" {
		return ""
	}
	if len(token) <= 4 {
		return "****"
	}
	return token[:2] + "****" + token[len(token)-2:]
}
