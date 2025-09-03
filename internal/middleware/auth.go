// File: internal/middleware/auth.go
package middleware

import (
    "context"
    "log"
    "net/http"
    "time"

    "github.com/iyunix/go-internist/internal/auth"
)

type UserIDKey string

const userIDKey UserIDKey = "userID"

// NewJWTMiddleware creates middleware to validate JWT from cookie with secure cookie options and logging.
func NewJWTMiddleware(secretKey string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            cookie, err := r.Cookie("auth_token")
            if err != nil {
                log.Printf("[AuthMiddleware] Missing auth_token cookie: %v from %s", err, r.RemoteAddr)
                http.Redirect(w, r, "/login", http.StatusSeeOther)
                return
            }
            tokenString := cookie.Value

            userID, err := auth.ValidateToken(tokenString, []byte(secretKey))
            if err != nil {
                log.Printf("[AuthMiddleware] Invalid token: %v from %s", err, r.RemoteAddr)
                // Clear potentially invalid cookie by setting expired cookie with secure flags
                expiredCookie := &http.Cookie{
                    Name:     "auth_token",
                    Value:    "",
                    Path:     "/",
                    Expires:  time.Unix(0, 0),
                    HttpOnly: true,
                    Secure:   true,            // Ensure cookie sent on HTTPS
                    SameSite: http.SameSiteLaxMode,
                }
                http.SetCookie(w, expiredCookie)
                http.Redirect(w, r, "/login", http.StatusSeeOther)
                return
            }

            ctx := context.WithValue(r.Context(), userIDKey, userID)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}
