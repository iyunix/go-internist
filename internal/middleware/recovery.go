// File: internal/middleware/recovery.go
package middleware

import (
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "os"
    "runtime/debug"
    "strings"
    "sync/atomic"
    "time"
)

var (
    panicCounter int64
    lastPanicTime time.Time
)

func RecoverPanic(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        defer func() {
            if err := recover(); err != nil {
                // Increment panic counter for metrics
                atomic.AddInt64(&panicCounter, 1)
                lastPanicTime = time.Now()
                
                // Get request ID and user context
                requestID := GetRequestID(r)
                if requestID == "" {
                    requestID = "unknown"
                }
                
                var userID uint
                var username string
                if id, ok := r.Context().Value(UserIDKey).(uint); ok {
                    userID = id
                }
                if name, ok := r.Context().Value(UsernameKey).(string); ok {
                    username = name
                }
                
                // Get client IP
                clientIP := r.RemoteAddr
                if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
                    clientIP = strings.Split(xff, ",")[0]
                }
                
                // Log panic with full context
                log.Printf("[PANIC_RECOVERY] request_id=%s path=%s method=%s user_id=%d username=%s remote_addr=%s error=%v timestamp=%s", 
                    requestID, r.URL.Path, r.Method, userID, username, clientIP, err, time.Now().UTC().Format(time.RFC3339))
                
                // Log stack trace in development or if configured
                if shouldLogStackTrace() {
                    log.Printf("[STACK_TRACE] request_id=%s\n%s", requestID, string(debug.Stack()))
                }
                
                // Security alert for potential attacks
                if isPotentialAttack(r, err) {
                    log.Printf("[SECURITY_ALERT] Potential attack detected - request_id=%s path=%s user_id=%d error=%v", 
                        requestID, r.URL.Path, userID, err)
                }
                
                // Set security headers
                w.Header().Set("Connection", "close")
                w.Header().Set("X-Request-ID", requestID)
                
                // Different responses for API vs web requests
                if isAPIRequest(r) {
                    // JSON response for API clients
                    w.Header().Set("Content-Type", "application/json")
                    w.WriteHeader(http.StatusInternalServerError)
                    
                    response := map[string]interface{}{
                        "error":      "internal_server_error",
                        "message":    "An unexpected error occurred. Please try again later.",
                        "request_id": requestID,
                        "timestamp":  time.Now().UTC().Format(time.RFC3339),
                    }
                    json.NewEncoder(w).Encode(response)
                } else {
                    // HTML response for web clients
                    w.WriteHeader(http.StatusInternalServerError)
                    fmt.Fprintf(w, `
                        <html>
                        <head><title>System Error - Internist AI</title></head>
                        <body style="font-family: Arial, sans-serif; text-align: center; padding: 50px;">
                            <h1>We're experiencing technical difficulties</h1>
                            <p>Our medical AI system encountered an unexpected error.</p>
                            <p>Please try again in a few moments.</p>
                            <p><small>Reference ID: %s</small></p>
                            <a href="/" style="color: #007bff;">Return to Home</a>
                        </body>
                        </html>
                    `, requestID)
                }
            }
        }()

        next.ServeHTTP(w, r)
    })
}

// Helper functions
func shouldLogStackTrace() bool {
    env := os.Getenv("GO_ENV")
    return env == "development" || os.Getenv("LOG_STACK_TRACES") == "true"
}

func isPotentialAttack(r *http.Request, err interface{}) bool {
    // Check for common attack patterns in panic messages
    errorStr := fmt.Sprintf("%v", err)
    attackPatterns := []string{
        "runtime error: index out of range",
        "reflect: call of reflect.Value",
        "interface conversion:",
    }
    
    for _, pattern := range attackPatterns {
        if strings.Contains(errorStr, pattern) {
            return true
        }
    }
    return false
}


// GetPanicMetrics returns panic statistics for monitoring
func GetPanicMetrics() (count int64, lastPanic time.Time) {
    return atomic.LoadInt64(&panicCounter), lastPanicTime
}
