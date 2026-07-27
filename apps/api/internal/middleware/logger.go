package middleware

import (
	"log"
	"net/http"
	"time"
)

type responseRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

// requestLogger records sanitized request metadata for every route.
// Request bodies, query strings, authorization headers, and tokens are never logged.
func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		recorder := &responseRecorder{ResponseWriter: w}

		next.ServeHTTP(recorder, r)

		status := recorder.status
		if status == 0 {
			status = http.StatusOK
		}
		log.Printf(
			"HTTP request: method=%s path=%q status=%d bytes=%d duration=%s remote=%q",
			r.Method,
			r.URL.Path,
			status,
			recorder.bytes,
			time.Since(started).Round(time.Microsecond),
			r.RemoteAddr,
		)
	})
}
