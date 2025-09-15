// File: internal/middleware/admin_middleware.go
package middleware

import (
    "context"
    "log"
    "net/http"
    "sync"
    "time"

    "github.com/iyunix/go-internist/internal/repository/user"
)

// AdminCache stores admin status to avoid database hits
type AdminCache struct {
    mu    sync.RWMutex
    cache map[uint]AdminCacheEntry
}

type AdminCacheEntry struct {
    IsAdmin   bool
    Username  string
    ExpiresAt time.Time
}

var (
    adminCache = &AdminCache{
        cache: make(map[uint]AdminCacheEntry),
    }
    cacheExpiry = 10 * time.Minute // Admin status cache for 10 minutes
)

// isAdminCached checks admin status with caching
func (ac *AdminCache) isAdminCached(userID uint, userRepo user.UserRepository, ctx context.Context) (bool, string, error) {
    ac.mu.RLock()
    entry, exists := ac.cache[userID]
    ac.mu.RUnlock()

    // Return cached result if valid
    if exists && time.Now().Before(entry.ExpiresAt) {
        return entry.IsAdmin, entry.Username, nil
    }

    // Cache miss or expired - fetch from database
    user, err := userRepo.FindByID(ctx, userID)
    if err != nil {
        return false, "", err
    }

    // Update cache
    ac.mu.Lock()
    ac.cache[userID] = AdminCacheEntry{
        IsAdmin:   user.IsAdmin,
        Username:  user.Username,
        ExpiresAt: time.Now().Add(cacheExpiry),
    }
    ac.mu.Unlock()

    return user.IsAdmin, user.Username, nil
}

// ClearAdminCache clears the admin cache for a specific user
func ClearAdminCache(userID uint) {
    adminCache.mu.Lock()
    delete(adminCache.cache, userID)
    adminCache.mu.Unlock()
}

// ClearAllAdminCache clears the entire admin cache
func ClearAllAdminCache() {
    adminCache.mu.Lock()
    adminCache.cache = make(map[uint]AdminCacheEntry)
    adminCache.mu.Unlock()
}

// RequireAdmin is a middleware that checks if the authenticated user has admin privileges.
// It MUST be used AFTER the standard JWT authentication middleware.
func RequireAdmin(userRepo user.UserRepository) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            startTime := time.Now()

            // 1. Get the userID from the context. The JWT middleware should have already placed it there.
            userID, ok := r.Context().Value(UserIDKey).(uint)
            if !ok || userID == 0 {
                // This indicates a problem with the auth setup or the token is missing claims.
                log.Printf("[AdminMiddleware] SECURITY_ALERT: Invalid authentication context for admin route %s from %s", 
                    r.URL.Path, r.RemoteAddr)
                
                w.Header().Set("Content-Type", "application/json")
                w.WriteHeader(http.StatusForbidden)
                w.Write([]byte(`{"error":"Authentication required","code":"AUTH_REQUIRED"}`))
                return
            }

            // 2. Check admin status with caching
            isAdmin, username, err := adminCache.isAdminCached(userID, userRepo, r.Context())
            if err != nil {
                // This could happen if the user was deleted after their token was issued.
                log.Printf("[AdminMiddleware] SECURITY_ALERT: Could not verify user ID %d for admin access to %s. Error: %v", 
                    userID, r.URL.Path, err)
                
                w.Header().Set("Content-Type", "application/json")
                w.WriteHeader(http.StatusForbidden)
                w.Write([]byte(`{"error":"Access verification failed","code":"USER_NOT_FOUND"}`))
                return
            }

            // 3. The core logic: check the IsAdmin flag.
            if !isAdmin {
                // The user is logged in, but they are NOT an admin.
                log.Printf("[AdminMiddleware] SECURITY_ALERT: UNAUTHORIZED_ADMIN_ACCESS - Non-admin user '%s' (ID: %d) from %s attempted to access admin route: %s", 
                    username, userID, r.RemoteAddr, r.URL.Path)
                
                // Log security event for monitoring
                logSecurityEvent("unauthorized_admin_access", userID, username, r.URL.Path, r.RemoteAddr)
                
                w.Header().Set("Content-Type", "application/json")
                w.WriteHeader(http.StatusForbidden)
                w.Write([]byte(`{"error":"Insufficient privileges","message":"You do not have permission to access this resource","code":"INSUFFICIENT_PRIVILEGES"}`))
                return
            }

            // 4. Add admin user info to context for downstream handlers
            ctx := context.WithValue(r.Context(), IsAdminKey, true)
            ctx = context.WithValue(ctx, UsernameKey, username)
            r = r.WithContext(ctx)

            // 5. If we reach here, the user is a verified admin. Allow the request to proceed.
            duration := time.Since(startTime)
            log.Printf("[AdminMiddleware] SUCCESS: Admin user '%s' (ID: %d) accessed admin route: %s (auth_time: %v)", 
                username, userID, r.URL.Path, duration)
            
            // Log successful admin access for security monitoring
            logSecurityEvent("admin_access_granted", userID, username, r.URL.Path, r.RemoteAddr)
            
            next.ServeHTTP(w, r)
        })
    }
}

// logSecurityEvent logs security-related events for monitoring and alerting
func logSecurityEvent(eventType string, userID uint, username, path, remoteAddr string) {
    // This can be enhanced to send to security monitoring systems
    // For now, we log with a specific format that monitoring tools can parse
    log.Printf("[SECURITY_EVENT] type=%s user_id=%d username=%s path=%s remote_addr=%s timestamp=%s", 
        eventType, userID, username, path, remoteAddr, time.Now().UTC().Format(time.RFC3339))
}

// AdminMiddlewareConfig allows configuration of the admin middleware
type AdminMiddlewareConfig struct {
    CacheExpiry     time.Duration
    EnableCache     bool
    SecurityLogging bool
}

// RequireAdminWithConfig creates an admin middleware with custom configuration
func RequireAdminWithConfig(userRepo user.UserRepository, config AdminMiddlewareConfig) func(http.Handler) http.Handler {
    if config.CacheExpiry > 0 {
        cacheExpiry = config.CacheExpiry
    }
    
    return RequireAdmin(userRepo)
}

// GetAdminFromContext retrieves admin user information from request context
func GetAdminFromContext(r *http.Request) (userID uint, username string, isAdmin bool) {
    if id, ok := r.Context().Value(UserIDKey).(uint); ok {
        userID = id
    }
    
    if name, ok := r.Context().Value(UsernameKey).(string); ok {
        username = name
    }
    
    if admin, ok := r.Context().Value(IsAdminKey).(bool); ok {
        isAdmin = admin
    }
    
    return userID, username, isAdmin
}

// MiddlewareMetrics provides metrics for monitoring
type MiddlewareMetrics struct {
    AdminAccessAttempts   int64
    AdminAccessGranted    int64
    AdminAccessDenied     int64
    CacheHits            int64
    CacheMisses          int64
}

var metrics = &MiddlewareMetrics{}

// GetMetrics returns current middleware metrics
func GetMetrics() MiddlewareMetrics {
    return *metrics
}

// ResetMetrics resets all metrics counters
func ResetMetrics() {
    metrics = &MiddlewareMetrics{}
}
