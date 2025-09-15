// File: internal/middleware/constants.go
package middleware

// Context keys for middleware communication
type contextKey string

const (
    UserIDKey   contextKey = "user_id"
    UserKey     contextKey = "user"
    IsAdminKey  contextKey = "is_admin"
    UsernameKey contextKey = "username"
    PhoneKey    contextKey = "phone"
)
