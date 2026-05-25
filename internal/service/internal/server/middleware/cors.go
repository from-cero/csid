package middleware

import (
	"net/http"
	"strings"

	"github.com/from-cero/csid/service/internal/config"
)

// CORS returns a middleware that sets Access-Control-Allow-* headers for matching origins.
// If cfg.AllowedOrigins is empty the middleware is a no-op (CORS disabled).
// Preflight OPTIONS requests are responded to immediately with 204 No Content.
func CORS(cfg *config.CORSConfig) func(http.Handler) http.Handler {
	if len(cfg.AllowedOrigins) == 0 {
		return func(next http.Handler) http.Handler { return next }
	}

	methods := strings.Join(cfg.AllowedMethods, ", ")

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin == "" || !isAllowedOrigin(cfg.AllowedOrigins, origin) {
				next.ServeHTTP(w, r)
				return
			}

			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", methods)
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-Id")
			w.Header().Add("Vary", "Origin")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func isAllowedOrigin(allowed []string, origin string) bool {
	for _, a := range allowed {
		if a == "*" || a == origin {
			return true
		}
	}
	return false
}
