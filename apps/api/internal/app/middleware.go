package app

import (
	"net/http"
	"time"

	"github.com/artumont/dotslashstream/internal/middleware"
	"github.com/artumont/dotslashstream/internal/platform"
)

// Chain wraps the given handler with a standard set of global middleware.
func (app *App) Chain(next http.Handler, redis platform.RedisClient) http.Handler {
	// Rate limit: 100 requests per 60-second sliding window per user/IP.
	// Only enabled when on production environment
	if app.Config.Environment == "prod" {
		next = middleware.RateLimit(redis, 100, 60*time.Second)(next)
	}

	/*
		NOTE: RequestLogger should ALWAYS run last as it needs to see the final
		response state after all other middleware have processed the request
	*/

	next = middleware.RequestLogger(next)

	return next
}
