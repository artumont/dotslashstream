package middleware

import (
	"log"
	"net/http"
	"time"

	"github.com/artumont/dotslashstream/internal/auth"
	"github.com/artumont/dotslashstream/internal/httpx"
	"github.com/artumont/dotslashstream/internal/platform"
)

// RateLimit returns a dependency that enforces a sliding-window rate limit
// per key using Redis sorted sets. The key is derived from the authenticated
// user ID when present, falling back to the client IP.
//
// Fail-open: Redis errors are logged and the request proceeds.
func RateLimit(redis platform.RedisClient, limit int, window time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := "rate:" + rateLimitKey(r)

			allowed, err := redis.Allow(r.Context(), key, limit, window)
			if err != nil {
				log.Printf("rate limit check failed: %v", err)
				next.ServeHTTP(w, r)
				return
			}

			if !allowed {
				httpx.WriteError(w, http.StatusTooManyRequests, "rate limit exceeded")
				return
			}

			if _, err := redis.IncrAndExpire(r.Context(), key, window); err != nil {
				log.Printf("rate limit increment failed: %v", err)
			}

			next.ServeHTTP(w, r)
		})
	}
}

// rateLimitKey builds a rate-limit key from the request. If a user is
// present in context (AuthRequired ran first), the key is the user ID;
// otherwise it falls back to RemoteAddr.
func rateLimitKey(r *http.Request) string {
	if user := auth.UserFromContext(r); user != nil {
		return "user:" + user.ID.String()
	}
	return "ip:" + r.RemoteAddr
}
