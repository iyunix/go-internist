package middleware

import (
    "context"
    "log"
    "net/http"
    "time"

    "github.com/iyunix/go-internist/internal/services/user_services"
)

const (
    UserIDKey    = "userID"
    UserAdminKey = "isAdmin"
)

// NewJWTMiddleware creates middleware to validate JWT and check admin status
func NewJWTMiddleware(
    authService *user_services.AuthService,
    userService *user_services.UserService,
    adminPhone string,
) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            cookie, err := r.Cookie("auth_token")
            if err != nil {
                log.Printf("[AuthMiddleware] Missing auth_token cookie: %v", err)
                http.Redirect(w, r, "/login", http.StatusSeeOther)
                return
            }

            userID, err := authService.ValidateJWTToken(cookie.Value)
            if err != nil {
                log.Printf("[AuthMiddleware] Invalid token: %v", err)
                clearAuthCookie(w)
                http.Redirect(w, r, "/login?error=session_expired", http.StatusSeeOther)
                return
            }

            // Use UserService to fetch user details
            user, err := userService.GetUserByID(r.Context(), userID)
            if err != nil {
                log.Printf("[AuthMiddleware] User %d not found: %v", userID, err)
                clearAuthCookie(w)
                http.Redirect(w, r, "/login?error=account_not_found", http.StatusSeeOther)
                return
            }

            // Check if user is admin based on phone number
            isAdmin := (user.PhoneNumber == adminPhone)
            
            log.Printf("[AuthMiddleware] User authenticated: ID=%d, Phone=%s, Admin=%t", 
                user.ID, user.PhoneNumber, isAdmin)

            // Add both userID and admin status to context
            ctx := context.WithValue(r.Context(), UserIDKey, userID)
            ctx = context.WithValue(ctx, UserAdminKey, isAdmin)
            
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}

// Helper function to clear auth cookie
func clearAuthCookie(w http.ResponseWriter) {
    http.SetCookie(w, &http.Cookie{
        Name:     "auth_token",
        Value:    "",
        Path:     "/",
        Expires:  time.Unix(0, 0),
        HttpOnly: true,
        Secure:   true,
        SameSite: http.SameSiteLaxMode,
    })
}