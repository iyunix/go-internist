// File: internal/middleware/auth.go
package middleware

import (
    "context"
    "encoding/json"
    "log"
    "net/http"
    "strings"
    "sync"
    "time"

    "github.com/iyunix/go-internist/internal/services/user_services"
)

// UserCache stores user information to avoid database hits
type UserCache struct {
    mu    sync.RWMutex
    cache map[uint]UserCacheEntry
}

type UserCacheEntry struct {
    ID          uint
    Username    string
    PhoneNumber string
    IsAdmin     bool
    ExpiresAt   time.Time
}

var (
    userCache = &UserCache{
        cache: make(map[uint]UserCacheEntry),
    }
    userCacheExpiry = 5 * time.Minute // User info cache for 5 minutes
)

// getUserCached gets user information with caching
func (uc *UserCache) getUserCached(userID uint, userService *user_services.UserService, ctx context.Context) (*UserCacheEntry, error) {
    uc.mu.RLock()
    entry, exists := uc.cache[userID]
    uc.mu.RUnlock()

    // Return cached result if valid
    if exists && time.Now().Before(entry.ExpiresAt) {
        return &entry, nil
    }

    // Cache miss or expired - fetch from database
    user, err := userService.GetUserByID(ctx, userID)
    if err != nil {
        return nil, err
    }

    // Create cache entry
    cacheEntry := UserCacheEntry{
        ID:          user.ID,
        Username:    user.Username,
        PhoneNumber: user.PhoneNumber,
        IsAdmin:     user.IsAdmin, // Use database field, not phone comparison
        ExpiresAt:   time.Now().Add(userCacheExpiry),
    }

    // Update cache
    uc.mu.Lock()
    uc.cache[userID] = cacheEntry
    uc.mu.Unlock()

    return &cacheEntry, nil
}

// ClearUserCache clears the user cache for a specific user
func ClearUserCache(userID uint) {
    userCache.mu.Lock()
    delete(userCache.cache, userID)
    userCache.mu.Unlock()
    
    // Also clear admin cache to keep them in sync
    ClearAdminCache(userID)
}

// ClearAllUserCache clears the entire user cache
func ClearAllUserCache() {
    userCache.mu.Lock()
    userCache.cache = make(map[uint]UserCacheEntry)
    userCache.mu.Unlock()
    
    // Also clear admin cache
    ClearAllAdminCache()
}

// isAPIRequest determines if the request is from an API client
func isAPIRequest(r *http.Request) bool {
    // Check for API path prefix
    if strings.HasPrefix(r.URL.Path, "/api/") {
        return true
    }
    
    // Check Accept header for JSON
    accept := r.Header.Get("Accept")
    if strings.Contains(accept, "application/json") {
        return true
    }
    
    // Check Content-Type for JSON
    contentType := r.Header.Get("Content-Type")
    if strings.Contains(contentType, "application/json") {
        return true
    }
    
    return false
}

// sendAPIError sends JSON error response for API requests
func sendAPIError(w http.ResponseWriter, code int, errorType, message string) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(code)
    
    response := map[string]interface{}{
        "error":     errorType,
        "message":   message,
        "timestamp": time.Now().UTC().Format(time.RFC3339),
    }
    
    json.NewEncoder(w).Encode(response)
}

// AuthMetrics tracks authentication metrics
type AuthMetrics struct {
    mu                    sync.RWMutex
    AuthAttempts         int64
    AuthSuccesses        int64
    AuthFailures         int64
    TokenValidations     int64
    CacheHits           int64
    CacheMisses         int64
}

var authMetrics = &AuthMetrics{}

// GetAuthMetrics returns current authentication metrics
func GetAuthMetrics() AuthMetrics {
    authMetrics.mu.RLock()
    defer authMetrics.mu.RUnlock()
    return *authMetrics
}

// NewJWTMiddleware creates middleware to validate JWT and check admin status
func NewJWTMiddleware(
    authService *user_services.AuthService,
    userService *user_services.UserService,
    adminPhone string, // Keep for backward compatibility, but won't be used
) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            startTime := time.Now()
            
            // Increment auth attempts metric
            authMetrics.mu.Lock()
            authMetrics.AuthAttempts++
            authMetrics.mu.Unlock()

            // Get auth token from cookie
            cookie, err := r.Cookie("auth_token")
            if err != nil {
                log.Printf("[AuthMiddleware] Missing auth_token cookie from %s for path %s: %v", 
                    r.RemoteAddr, r.URL.Path, err)
                
                authMetrics.mu.Lock()
                authMetrics.AuthFailures++
                authMetrics.mu.Unlock()
                
                logAuthEvent("auth_cookie_missing", 0, "", r.URL.Path, r.RemoteAddr)
                
                if isAPIRequest(r) {
                    sendAPIError(w, http.StatusUnauthorized, "authentication_required", "Authentication token required")
                    return
                }
                
                http.Redirect(w, r, "/login", http.StatusSeeOther)
                return
            }

            // Validate JWT token
            authMetrics.mu.Lock()
            authMetrics.TokenValidations++
            authMetrics.mu.Unlock()
            
            userID, err := authService.ValidateJWTToken(cookie.Value)
            if err != nil {
                log.Printf("[AuthMiddleware] Invalid token from %s for path %s: %v", 
                    r.RemoteAddr, r.URL.Path, err)
                
                authMetrics.mu.Lock()
                authMetrics.AuthFailures++
                authMetrics.mu.Unlock()
                
                logAuthEvent("token_invalid", 0, "", r.URL.Path, r.RemoteAddr)
                clearAuthCookie(w)
                
                if isAPIRequest(r) {
                    sendAPIError(w, http.StatusUnauthorized, "invalid_token", "Authentication token is invalid or expired")
                    return
                }
                
                http.Redirect(w, r, "/login?error=session_expired", http.StatusSeeOther)
                return
            }

            // Get user information with caching
            user, err := userCache.getUserCached(userID, userService, r.Context())
            if err != nil {
                log.Printf("[AuthMiddleware] User %d not found from %s for path %s: %v", 
                    userID, r.RemoteAddr, r.URL.Path, err)
                
                authMetrics.mu.Lock()
                authMetrics.AuthFailures++
                authMetrics.mu.Unlock()
                
                logAuthEvent("user_not_found", userID, "", r.URL.Path, r.RemoteAddr)
                clearAuthCookie(w)
                
                if isAPIRequest(r) {
                    sendAPIError(w, http.StatusUnauthorized, "user_not_found", "User account not found")
                    return
                }
                
                http.Redirect(w, r, "/login?error=account_not_found", http.StatusSeeOther)
                return
            }

            // Update cache hit metrics
            authMetrics.mu.Lock()
            authMetrics.CacheHits++
            authMetrics.mu.Unlock()

            // Success - update metrics
            authMetrics.mu.Lock()
            authMetrics.AuthSuccesses++
            authMetrics.mu.Unlock()

            duration := time.Since(startTime)
            log.Printf("[AuthMiddleware] User authenticated: ID=%d, Username=%s, Phone=%s, Admin=%t (auth_time: %v)", 
                user.ID, user.Username, user.PhoneNumber, user.IsAdmin, duration)

            logAuthEvent("auth_success", user.ID, user.Username, r.URL.Path, r.RemoteAddr)

            // Add comprehensive user info to context
            ctx := context.WithValue(r.Context(), UserIDKey, user.ID)
            ctx = context.WithValue(ctx, UsernameKey, user.Username)
            ctx = context.WithValue(ctx, PhoneKey, user.PhoneNumber)
            ctx = context.WithValue(ctx, IsAdminKey, user.IsAdmin)
            ctx = context.WithValue(ctx, UserKey, user) // Full user object for convenience
            
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}

// logAuthEvent logs authentication events for security monitoring
func logAuthEvent(eventType string, userID uint, username, path, remoteAddr string) {
    log.Printf("[AUTH_EVENT] type=%s user_id=%d username=%s path=%s remote_addr=%s timestamp=%s", 
        eventType, userID, username, path, remoteAddr, time.Now().UTC().Format(time.RFC3339))
}

// Enhanced cookie clearing with proper security settings
func clearAuthCookie(w http.ResponseWriter) {
    http.SetCookie(w, &http.Cookie{
        Name:     "auth_token",
        Value:    "",
        Path:     "/",
        Expires:  time.Unix(0, 0),
        MaxAge:   -1,
        HttpOnly: true,
        Secure:   true, // Only send over HTTPS in production
        SameSite: http.SameSiteStrictMode, // Enhanced CSRF protection
    })
}

// GetUserFromContext retrieves user information from request context
func GetUserFromContext(r *http.Request) (userID uint, username, phone string, isAdmin bool, ok bool) {
    userID, ok1 := r.Context().Value(UserIDKey).(uint)
    username, ok2 := r.Context().Value(UsernameKey).(string)
    phone, ok3 := r.Context().Value(PhoneKey).(string)
    isAdmin, ok4 := r.Context().Value(IsAdminKey).(bool)
    
    ok = ok1 && ok2 && ok3 && ok4
    return userID, username, phone, isAdmin, ok
}

// GetFullUserFromContext retrieves the full user object from context
func GetFullUserFromContext(r *http.Request) (*UserCacheEntry, bool) {
    user, ok := r.Context().Value(UserKey).(*UserCacheEntry)
    return user, ok
}

// AuthConfig allows configuration of the auth middleware
type AuthConfig struct {
    CacheExpiry         time.Duration
    EnableCache         bool
    SecurityLogging     bool
    StrictCookieSettings bool
}

// NewJWTMiddlewareWithConfig creates auth middleware with custom configuration
func NewJWTMiddlewareWithConfig(
    authService *user_services.AuthService,
    userService *user_services.UserService,
    adminPhone string,
    config AuthConfig,
) func(http.Handler) http.Handler {
    if config.CacheExpiry > 0 {
        userCacheExpiry = config.CacheExpiry
    }
    
    return NewJWTMiddleware(authService, userService, adminPhone)
}

// RequireAuth is a helper middleware that can be used for routes that need authentication
func RequireAuth(authService *user_services.AuthService, userService *user_services.UserService, adminPhone string) func(http.Handler) http.Handler {
    return NewJWTMiddleware(authService, userService, adminPhone)
}
