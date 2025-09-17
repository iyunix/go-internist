// File: internal/ratelimit/ratelimit.go
package ratelimit

import (
    "net"
    "net/http"
    "sync"
    "time"
    "strings"
)

// Config holds rate limiting configuration
type Config struct {
    WindowSize    time.Duration // Time window for rate limiting
    MaxAttempts   int          // Maximum attempts per window
    CleanupPeriod time.Duration // How often to clean up old entries
    BanDuration   time.Duration // How long to ban after exceeding limit
}

// DefaultAuthConfig returns sensible defaults for auth endpoints
func DefaultAuthConfig() *Config {
    return &Config{
        WindowSize:    15 * time.Minute, // 15-minute window
        MaxAttempts:   5,                // 5 attempts per window
        CleanupPeriod: 30 * time.Minute, // Clean up every 30 minutes
        BanDuration:   30 * time.Minute, // Ban for 30 minutes after limit
    }
}

// StrictAuthConfig returns stricter limits for sensitive operations
func StrictAuthConfig() *Config {
    return &Config{
        WindowSize:    10 * time.Minute, // 10-minute window
        MaxAttempts:   3,                // Only 3 attempts
        CleanupPeriod: 20 * time.Minute,
        BanDuration:   60 * time.Minute, // Ban for 1 hour
    }
}

// attemptRecord tracks attempts for an IP/identifier
type attemptRecord struct {
    Count     int
    FirstSeen time.Time
    LastSeen  time.Time
    BannedAt  *time.Time
}

// MemoryRateLimiter implements in-memory rate limiting
type MemoryRateLimiter struct {
    config   *Config
    attempts map[string]*attemptRecord
    mu       sync.RWMutex
    stopCh   chan struct{}
}

// NewMemoryRateLimiter creates a new in-memory rate limiter
func NewMemoryRateLimiter(config *Config) *MemoryRateLimiter {
    limiter := &MemoryRateLimiter{
        config:   config,
        attempts: make(map[string]*attemptRecord),
        stopCh:   make(chan struct{}),
    }
    
    // Start cleanup goroutine
    go limiter.cleanupLoop()
    
    return limiter
}

// Allow checks if a request should be allowed
func (rl *MemoryRateLimiter) Allow(identifier string) (bool, *RateLimitInfo) {
    rl.mu.Lock()
    defer rl.mu.Unlock()
    
    now := time.Now()
    record, exists := rl.attempts[identifier]
    
    // If no record exists, create one and allow
    if !exists {
        rl.attempts[identifier] = &attemptRecord{
            Count:     1,
            FirstSeen: now,
            LastSeen:  now,
        }
        return true, &RateLimitInfo{
            Allowed:    true,
            Remaining:  rl.config.MaxAttempts - 1,
            ResetTime:  now.Add(rl.config.WindowSize),
            RetryAfter: 0,
        }
    }
    
    // Check if currently banned
    if record.BannedAt != nil && now.Sub(*record.BannedAt) < rl.config.BanDuration {
        remainingBan := rl.config.BanDuration - now.Sub(*record.BannedAt)
        return false, &RateLimitInfo{
            Allowed:    false,
            Remaining:  0,
            ResetTime:  record.BannedAt.Add(rl.config.BanDuration),
            RetryAfter: remainingBan,
            Banned:     true,
        }
    }
    
    // Check if window has reset
    if now.Sub(record.FirstSeen) > rl.config.WindowSize {
        // Reset the window
        record.Count = 1
        record.FirstSeen = now
        record.LastSeen = now
        record.BannedAt = nil
        return true, &RateLimitInfo{
            Allowed:    true,
            Remaining:  rl.config.MaxAttempts - 1,
            ResetTime:  now.Add(rl.config.WindowSize),
            RetryAfter: 0,
        }
    }
    
    // Increment counter
    record.Count++
    record.LastSeen = now
    
    // Check if limit exceeded
    if record.Count > rl.config.MaxAttempts {
        // Ban the identifier
        banTime := now
        record.BannedAt = &banTime
        return false, &RateLimitInfo{
            Allowed:    false,
            Remaining:  0,
            ResetTime:  now.Add(rl.config.BanDuration),
            RetryAfter: rl.config.BanDuration,
            Banned:     true,
        }
    }
    
    // Still within limits
    remaining := rl.config.MaxAttempts - record.Count
    return true, &RateLimitInfo{
        Allowed:    true,
        Remaining:  remaining,
        ResetTime:  record.FirstSeen.Add(rl.config.WindowSize),
        RetryAfter: 0,
    }
}

// RecordSuccess records a successful authentication (resets attempts)
func (rl *MemoryRateLimiter) RecordSuccess(identifier string) {
    rl.mu.Lock()
    defer rl.mu.Unlock()
    
    // Reset attempts on successful authentication
    delete(rl.attempts, identifier)
}

// RateLimitInfo contains information about rate limit status
type RateLimitInfo struct {
    Allowed    bool
    Remaining  int
    ResetTime  time.Time
    RetryAfter time.Duration
    Banned     bool
}

// cleanupLoop periodically removes old records
func (rl *MemoryRateLimiter) cleanupLoop() {
    ticker := time.NewTicker(rl.config.CleanupPeriod)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            rl.cleanup()
        case <-rl.stopCh:
            return
        }
    }
}

// cleanup removes expired records
func (rl *MemoryRateLimiter) cleanup() {
    rl.mu.Lock()
    defer rl.mu.Unlock()
    
    now := time.Now()
    for identifier, record := range rl.attempts {
        // Remove if window has expired and not banned, or ban has expired
        windowExpired := now.Sub(record.FirstSeen) > rl.config.WindowSize
        banExpired := record.BannedAt != nil && now.Sub(*record.BannedAt) > rl.config.BanDuration
        
        if (windowExpired && record.BannedAt == nil) || banExpired {
            delete(rl.attempts, identifier)
        }
    }
}

// Close stops the cleanup goroutine
func (rl *MemoryRateLimiter) Close() {
    close(rl.stopCh)
}

// GetClientIP extracts the real client IP from request
func GetClientIP(r *http.Request) string {
    // Check for forwarded IP (behind proxy/load balancer)
    forwarded := r.Header.Get("X-Forwarded-For")
    if forwarded != "" {
        // Take the first IP in case of multiple
        if ip := parseFirstIP(forwarded); ip != "" {
            return ip
        }
    }
    
    // Check for real IP header
    realIP := r.Header.Get("X-Real-IP")
    if realIP != "" {
        return realIP
    }
    
    // Fall back to remote address
    ip, _, err := net.SplitHostPort(r.RemoteAddr)
    if err != nil {
        return r.RemoteAddr
    }
    return ip
}

// parseFirstIP extracts the first valid IP from a comma-separated list
func parseFirstIP(forwarded string) string {
    if forwarded == "" {
        return ""
    }
    
    // Split by comma and take first
    ips := strings.Split(forwarded, ",")
    if len(ips) > 0 {
        return strings.TrimSpace(ips[0])
    }
    return ""
}
