// File: internal/middleware/logger.go
package middleware

import (
	"log"
	"net/http"
	"time"
)

// LoggingMiddleware logs incoming HTTP request & response details.
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Record the start time of the request
		start := time.Now()

		// Call the next handler in the chain (e.g., our router).
		// This is the line that actually processes the request.
		next.ServeHTTP(w, r)

		// After the request has been handled, log the details.
		log.Printf(
			"Request: %s %s from %s | Duration: %v",
			r.Method,
			r.RequestURI,
			r.RemoteAddr,
			time.Since(start),
		)
	})
}