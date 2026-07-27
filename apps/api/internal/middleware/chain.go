package middleware

import (
	"net/http"
	"time"

	"github.com/artumont/dotslashstream/internal/platform"
)

// Chain wraps the given handler with a standard set of global middleware.
func Chain(next http.Handler, redis platform.RedisClient) http.Handler {
	// Rate limit: 100 requests per 60-second sliding window per user/IP.
	next = RateLimit(redis, 100, 60*time.Second)(next)

	/*
		NOTE: RequestLogger should ALWAYS run last as it needs to see the final
		response state after all other middleware have processed the request
	*/

	next = RequestLogger(next)

	return next
}
