// File: internal/middleware/ratelimit.go
package middleware

import (
	"encoding/json"
	"log"
	"net/http"
	"fmt"

	"github.com/iyunix/go-internist/internal/ratelimit"
)

// RateLimitMiddleware creates a rate limiting middleware
func RateLimitMiddleware(limiter *ratelimit.MemoryRateLimiter, name string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get client IP
			clientIP := ratelimit.GetClientIP(r)

			// Create identifier
			identifier := clientIP // You can still prefix with name if needed: fmt.Sprintf("%s:%s", name, clientIP)

			// Check rate limit
			allowed, info := limiter.Allow(identifier)

			// ✅ FIXED: Calculate limit header properly
			limit := info.Remaining
			if info.Allowed {
				limit++ // if allowed, remaining + 1 = original limit
			}
			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", limit))
			w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", info.Remaining))
			w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", info.ResetTime.Unix()))

			if !allowed {
				// ✅ FIXED: Use if/else for log message
				statusMsg := "RATE LIMITED"
				if info.Banned {
					statusMsg = "BANNED"
				}
				log.Printf("[RateLimit] Blocked %s request from %s - %s",
					name, clientIP, statusMsg)

				// Set retry-after header
				if info.RetryAfter > 0 {
					w.Header().Set("Retry-After", fmt.Sprintf("%.0f", info.RetryAfter.Seconds()))
				}

				// Return rate limit exceeded error
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)

				errorMsg := "Too many attempts. Please try again later."
				if info.Banned {
					errorMsg = fmt.Sprintf("Too many failed attempts. Account temporarily locked for security. Try again in %d minutes.",
						int(info.RetryAfter.Minutes()))
				}

				json.NewEncoder(w).Encode(map[string]interface{}{
					"error":      errorMsg,
					"retryAfter": int(info.RetryAfter.Seconds()),
					"banned":     info.Banned,
				})
				return
			}

			// Request allowed, proceed
			next.ServeHTTP(w, r)
		})
	}
}

// AuthSuccessMiddleware creates middleware to record successful authentications
func AuthSuccessMiddleware(limiter *ratelimit.MemoryRateLimiter, name string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Use a custom response writer to capture status code
			wrapper := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			// Call next handler
			next.ServeHTTP(wrapper, r)

			// If response was successful (2xx), reset rate limit
			if wrapper.statusCode >= 200 && wrapper.statusCode < 300 {
				clientIP := ratelimit.GetClientIP(r)
				identifier := clientIP // or fmt.Sprintf("%s:%s", name, clientIP)
				limiter.RecordSuccess(identifier)
				log.Printf("[RateLimit] Reset attempts for %s from %s (successful auth)", name, clientIP)
			}
		})
	}
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}