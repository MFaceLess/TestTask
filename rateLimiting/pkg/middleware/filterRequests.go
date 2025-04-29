package middleware

import (
	"errors"
	"net/http"
	"rateLimiting/pkg/db"
	"rateLimiting/pkg/token"

	"rateLimiting/pkg/response"
)

var (
	ErrTooManyRequests = errors.New("too many requests")
)

// Middleware для всех входящих соединений, было принято решение использовать IP в качестве
// уникального идентификатора, также можно было генерировать jwt key с соответствующими полями
func RateLimitMiddleware(rateLimiter *token.RateLimiter, capacity, refillRate float64, db *db.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			clientIP := getClientIP(r)
			db.UpdateOrInsertClient(clientIP, capacity, refillRate)
			if !rateLimiter.AllowRequest(clientIP, capacity, refillRate) {
				w.WriteHeader(http.StatusTooManyRequests)
				response.ResponseJSON(w, http.StatusTooManyRequests, ErrTooManyRequests.Error())
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func getClientIP(r *http.Request) string {
	ip := r.Header.Get("X-Real-IP")
	if ip == "" {
		ip = r.Header.Get("X-Forwarded-For")
	}
	if ip == "" {
		ip = r.RemoteAddr
	}
	return ip
}
