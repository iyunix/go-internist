// File: internal/middleware/logger.go
package middleware

import (
    "encoding/json"
    "log"
    "math/rand"
    "net/http"
    "time"
)

type logEntry struct {
    Timestamp  string `json:"timestamp"`
    Method     string `json:"method"`
    Path       string `json:"path"`
    RemoteAddr string `json:"remote_addr"`
    RequestID  string `json:"request_id"`
    StatusCode int    `json:"status_code"`
    DurationMS int64  `json:"duration_ms"`
}

func generateRequestID() string {
    letters := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
    b := make([]rune, 12)
    for i := range b {
        b[i] = letters[rand.Intn(len(letters))]
    }
    return string(b)
}

func LoggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        requestID := generateRequestID()

        lrw := &loggingResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
        next.ServeHTTP(lrw, r)

        entry := logEntry{
            Timestamp:  start.Format(time.RFC3339Nano),
            Method:     r.Method,
            Path:       r.URL.Path,
            RemoteAddr: r.RemoteAddr,
            RequestID:  requestID,
            StatusCode: lrw.statusCode,
            DurationMS: time.Since(start).Milliseconds(),
        }
        b, _ := json.Marshal(entry)
        log.Println(string(b))
    })
}

type loggingResponseWriter struct {
    http.ResponseWriter
    statusCode int
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
    lrw.statusCode = code
    lrw.ResponseWriter.WriteHeader(code)
}

// Forward Flush to the underlying writer to support streaming (SSE).
func (lrw *loggingResponseWriter) Flush() {
    if f, ok := lrw.ResponseWriter.(http.Flusher); ok {
        f.Flush()
    }
}
