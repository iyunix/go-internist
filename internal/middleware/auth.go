package middleware

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/iyunix/go-internist/internal/services/user_services"
)

const UserIDKey = "userID"

// NewJWTMiddleware creates middleware to validate JWT from cookie
func NewJWTMiddleware(authService *user_services.AuthService) func(http.Handler) http.Handler {
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
				// Clear invalid cookie
				http.SetCookie(w, &http.Cookie{
					Name:     "auth_token",
					Value:    "",
					Path:     "/",
					Expires:  time.Unix(0, 0),
					HttpOnly: true,
					Secure:   true, // Secure flag set to true
					SameSite: http.SameSiteLaxMode,
				})
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}

			ctx := context.WithValue(r.Context(), UserIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
