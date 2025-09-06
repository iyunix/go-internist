// File: internal/middleware/auth.go
package middleware

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/iyunix/go-internist/internal/auth"
)

// The key is now a simple constant string for easier use across packages.
const UserIDKey = "userID"

// NewJWTMiddleware creates middleware to validate JWT from cookie.
func NewJWTMiddleware(secretKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("auth_token")
			if err != nil {
				log.Printf("[AuthMiddleware] Missing auth_token cookie: %v", err)
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}
			tokenString := cookie.Value

			userID, err := auth.ValidateToken(tokenString, []byte(secretKey))
			if err != nil {
				log.Printf("[AuthMiddleware] Invalid token: %v", err)
				// Clear potentially invalid cookie.
				// We set Secure: false for local http development, to match the login handler.
				http.SetCookie(w, &http.Cookie{
					Name:     "auth_token",
					Value:    "",
					Path:     "/",
					Expires:  time.Unix(0, 0),
					HttpOnly: true,
					Secure:   false, 
					SameSite: http.SameSiteLaxMode,
				})
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}

			// Use the updated string constant for the key.
			ctx := context.WithValue(r.Context(), UserIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}