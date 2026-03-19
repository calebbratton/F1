package middleware

import (
	"encoding/json"
	"net/http"
)

// APIKey returns a middleware that requires requests to include a valid
// X-API-Key header. If key is empty the middleware is a no-op (open access).
func APIKey(key string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if key == "" {
			return next // no key configured — allow all
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("X-API-Key") != key {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{"error": "invalid or missing API key"})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
