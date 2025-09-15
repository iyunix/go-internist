// File: internal/middleware/logger.go
package middleware

import (
    "context"
    "crypto/rand"
    "encoding/hex"
    "encoding/json"
    "log"
    "net/http"
    "os"
    "strings"
    "sync/atomic"
    "time"
)

// RequestIDKey is the context key for request IDs
const RequestIDKey contextKey = "request_id"

// HTTPLogEntry represents a structured HTTP request log entry
type HTTPLogEntry struct {
    Timestamp    string `json:"timestamp"`
    RequestID    string `json:"request_id"`
    Method       string `json:"method"`
    Path         string `json:"path"`
    Query        string `json:"query,omitempty"`
    RemoteAddr   string `json:"remote_addr"`
    UserAgent    string `json:"user_agent,omitempty"`
    StatusCode   int    `json:"status_code"`
    DurationMS   int64  `json:"duration_ms"`
    ResponseSize int64  `json:"response_size"`
    UserID       uint   `json:"user_id,omitempty"`
    Username     string `json:"username,omitempty"`
    IsAdmin      bool   `json:"is_admin,omitempty"`
    Error        string `json:"error,omitempty"`
}

// LoggingConfig configures the logging middleware behavior
type LoggingConfig struct {
    IncludeQuery     bool
    IncludeUserAgent bool
    FilterSensitive  bool
    LogSlowRequests  bool
    SlowThreshold    time.Duration
}

// DefaultLoggingConfig returns sensible defaults for production
func DefaultLoggingConfig() LoggingConfig {
    return LoggingConfig{
        IncludeQuery:     true,
        IncludeUserAgent: true,
        FilterSensitive:  true,
        LogSlowRequests:  true,
        SlowThreshold:    2 * time.Second,
    }
}

var (
    requestCounter int64
    
    // Sensitive query parameters to filter
    sensitiveParams = map[string]bool{
        "password":    true,
        "token":       true,
        "api_key":     true,
        "secret":      true,
        "code":        true,
        "otp":         true,
        "verification_code": true,
    }
)

// generateSecureRequestID creates a cryptographically secure request ID
func generateSecureRequestID() string {
    bytes := make([]byte, 8)
    if _, err := rand.Read(bytes); err != nil {
        // Fallback to counter-based ID if crypto/rand fails
        counter := atomic.AddInt64(&requestCounter, 1)
        return hex.EncodeToString([]byte{
            byte(counter >> 56), byte(counter >> 48), byte(counter >> 40), byte(counter >> 32),
            byte(counter >> 24), byte(counter >> 16), byte(counter >> 8), byte(counter),
        })
    }
    return hex.EncodeToString(bytes)
}

// filterSensitiveQuery removes sensitive parameters from query string
func filterSensitiveQuery(query string) string {
    if query == "" {
        return query
    }
    
    params := strings.Split(query, "&")
    var filtered []string
    
    for _, param := range params {
        if parts := strings.SplitN(param, "=", 2); len(parts) >= 1 {
            key := strings.ToLower(parts[0])
            if sensitiveParams[key] {
                if len(parts) == 2 {
                    filtered = append(filtered, parts[0]+"=[FILTERED]")
                } else {
                    filtered = append(filtered, param)
                }
            } else {
                filtered = append(filtered, param)
            }
        } else {
            filtered = append(filtered, param)
        }
    }
    
    return strings.Join(filtered, "&")
}

// getClientIP extracts the real client IP from headers or RemoteAddr
func getClientIP(r *http.Request) string {
    // Check for X-Forwarded-For (load balancers, proxies)
    if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
        ips := strings.Split(xff, ",")
        return strings.TrimSpace(ips[0])
    }
    
    // Check for X-Real-IP (Nginx proxy)
    if xri := r.Header.Get("X-Real-IP"); xri != "" {
        return strings.TrimSpace(xri)
    }
    
    // Check for X-Forwarded header
    if xf := r.Header.Get("X-Forwarded"); xf != "" {
        return strings.TrimSpace(xf)
    }
    
    // Fallback to RemoteAddr
    return r.RemoteAddr
}

// LoggingResponseWriter wraps http.ResponseWriter to capture response metadata
type LoggingResponseWriter struct {
    http.ResponseWriter
    statusCode   int
    responseSize int64
    wroteHeader  bool
}

// NewLoggingResponseWriter creates a new logging response writer
func NewLoggingResponseWriter(w http.ResponseWriter) *LoggingResponseWriter {
    return &LoggingResponseWriter{
        ResponseWriter: w,
        statusCode:     http.StatusOK,
    }
}

// WriteHeader captures the HTTP status code
func (lrw *LoggingResponseWriter) WriteHeader(code int) {
    if !lrw.wroteHeader {
        lrw.statusCode = code
        lrw.wroteHeader = true
    }
    lrw.ResponseWriter.WriteHeader(code)
}

// Write captures the response size and writes data
func (lrw *LoggingResponseWriter) Write(data []byte) (int, error) {
    if !lrw.wroteHeader {
        lrw.WriteHeader(http.StatusOK)
    }
    
    n, err := lrw.ResponseWriter.Write(data)
    lrw.responseSize += int64(n)
    return n, err
}

// Flush implements http.Flusher for SSE and streaming responses
func (lrw *LoggingResponseWriter) Flush() {
    if !lrw.wroteHeader {
        lrw.WriteHeader(http.StatusOK)
    }
    
    if f, ok := lrw.ResponseWriter.(http.Flusher); ok {
        f.Flush()
    }
}

// LoggingMiddleware creates HTTP request logging middleware with default config
func LoggingMiddleware(next http.Handler) http.Handler {
    return LoggingMiddlewareWithConfig(next, DefaultLoggingConfig())
}

// LoggingMiddlewareWithConfig creates configurable HTTP request logging middleware
func LoggingMiddlewareWithConfig(next http.Handler, config LoggingConfig) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Record start time
        startTime := time.Now()
        
        // Generate secure request ID
        requestID := generateSecureRequestID()
        
        // Add request ID to context and response headers
        ctx := context.WithValue(r.Context(), RequestIDKey, requestID)
        r = r.WithContext(ctx)
        w.Header().Set("X-Request-ID", requestID)
        
        // Wrap response writer to capture metadata
        lrw := NewLoggingResponseWriter(w)
        
        // Extract user information from context (set by auth middleware)
        var userID uint
        var username string
        var isAdmin bool
        
        if id, ok := r.Context().Value(UserIDKey).(uint); ok {
            userID = id
        }
        if name, ok := r.Context().Value(UsernameKey).(string); ok {
            username = name
        }
        if admin, ok := r.Context().Value(IsAdminKey).(bool); ok {
            isAdmin = admin
        }
        
        // Process the request
        next.ServeHTTP(lrw, r)
        
        // Calculate request duration
        duration := time.Since(startTime)
        
        // Build structured log entry
        entry := HTTPLogEntry{
            Timestamp:    startTime.UTC().Format(time.RFC3339),
            RequestID:    requestID,
            Method:       r.Method,
            Path:         r.URL.Path,
            RemoteAddr:   getClientIP(r),
            StatusCode:   lrw.statusCode,
            DurationMS:   duration.Milliseconds(),
            ResponseSize: lrw.responseSize,
        }
        
        // Add query parameters (filtered if sensitive)
        if config.IncludeQuery && r.URL.RawQuery != "" {
            if config.FilterSensitive {
                entry.Query = filterSensitiveQuery(r.URL.RawQuery)
            } else {
                entry.Query = r.URL.RawQuery
            }
        }
        
        // Add user agent
        if config.IncludeUserAgent && r.UserAgent() != "" {
            entry.UserAgent = r.UserAgent()
        }
        
        // Add authenticated user information
        if userID > 0 {
            entry.UserID = userID
            entry.Username = username
            entry.IsAdmin = isAdmin
        }
        
        // Add error information for failed requests
        if lrw.statusCode >= 400 {
            entry.Error = http.StatusText(lrw.statusCode)
        }
        
        // Log the request
        logHTTPRequest(entry)
        
        // Log slow requests if enabled
        if config.LogSlowRequests && duration > config.SlowThreshold {
            logSlowRequest(requestID, r.URL.Path, duration, userID, username)
        }
        
        // Log security events - FIXED FUNCTION CALL
        logHTTPSecurityEvent(lrw.statusCode, requestID, r.URL.Path, getClientIP(r), userID, username)
    })
}


// logHTTPRequest outputs the HTTP request log entry
func logHTTPRequest(entry HTTPLogEntry) {
    // Convert to JSON
    jsonData, err := json.Marshal(entry)
    if err != nil {
        log.Printf("[LOG_ERROR] Failed to marshal HTTP log entry: %v (request_id=%s, path=%s)", 
            err, entry.RequestID, entry.Path)
        return
    }
    
    // Determine log level based on status code
    prefix := "[INFO]"
    if entry.StatusCode >= 500 {
        prefix = "[ERROR]"
    } else if entry.StatusCode >= 400 {
        prefix = "[WARN]"
    }
    
    // Log to stdout (container/production friendly)
    if shouldLogToStdout() {
        log.Printf("%s %s", prefix, string(jsonData))
    } else {
        // Development logging (more readable)
        log.Printf("%s [%s] %s %s - %d (%dms) [User: %d] Size: %dB", 
            prefix, entry.RequestID[:8], entry.Method, entry.Path, 
            entry.StatusCode, entry.DurationMS, entry.UserID, entry.ResponseSize)
    }
}

// logSlowRequest logs performance warnings for slow requests
func logSlowRequest(requestID, path string, duration time.Duration, userID uint, username string) {
    log.Printf("[PERFORMANCE_ALERT] Slow request detected - ID: %s, Path: %s, Duration: %s, User: %d (%s)", 
        requestID, path, duration.String(), userID, username)
}

// logSecurityEvent logs security-related HTTP events
func logHTTPSecurityEvent(statusCode int, requestID, path, remoteAddr string, userID uint, username string) {
    switch statusCode {
    case 401:
        log.Printf("[SECURITY_EVENT] Unauthorized access - ID: %s, Path: %s, IP: %s, User: %d (%s)", 
            requestID, path, remoteAddr, userID, username)
    case 403:
        log.Printf("[SECURITY_EVENT] Forbidden access - ID: %s, Path: %s, IP: %s, User: %d (%s)", 
            requestID, path, remoteAddr, userID, username)
    case 429:
        log.Printf("[SECURITY_EVENT] Rate limit exceeded - ID: %s, Path: %s, IP: %s, User: %d (%s)", 
            requestID, path, remoteAddr, userID, username)
    }
}

// shouldLogToStdout determines output format based on environment
func shouldLogToStdout() bool {
    env := os.Getenv("GO_ENV")
    return env == "production" || os.Getenv("LOG_FORMAT") == "json"
}

// GetRequestID retrieves the request ID from the request context
func GetRequestID(r *http.Request) string {
    if requestID, ok := r.Context().Value(RequestIDKey).(string); ok {
        return requestID
    }
    return ""
}

// LoggingMetrics provides basic metrics about HTTP requests
type LoggingMetrics struct {
    TotalRequests int64
    ErrorCount    int64
    SlowCount     int64
    AvgDuration   time.Duration
}

// Simple in-memory metrics (for production, use proper metrics system)
var globalMetrics = struct {
    requests int64
    errors   int64
    slow     int64
}{0, 0, 0}

// GetMetrics returns current logging metrics
func GetLoggingMetrics() LoggingMetrics {
    return LoggingMetrics{
        TotalRequests: globalMetrics.requests,
        ErrorCount:    globalMetrics.errors,
        SlowCount:     globalMetrics.slow,
    }
}

// Helper function to check if path should be excluded from logging
func shouldSkipLogging(path string) bool {
    skipPaths := []string{
        "/health",
        "/favicon.ico",
        "/robots.txt",
    }
    
    for _, skipPath := range skipPaths {
        if path == skipPath {
            return true
        }
    }
    return false
}

// Enhanced logging middleware that can skip certain paths
func LoggingMiddlewareWithSkip(next http.Handler, config LoggingConfig, skipPaths []string) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Check if we should skip logging for this path
        for _, skipPath := range skipPaths {
            if r.URL.Path == skipPath {
                next.ServeHTTP(w, r)
                return
            }
        }
        
        // Use regular logging middleware
        LoggingMiddlewareWithConfig(next, config).ServeHTTP(w, r)
    })
}
