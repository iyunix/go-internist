// File: internal/middleware/auth.go
package middleware

import (
	"context"
	"net/http"

	"github.com/iyunix/go-internist/internal/auth"
)

type UserIDKey string
const userIDKey UserIDKey = "userID"

// NewJWTMiddleware is a factory that creates our JWT middleware.
func NewJWTMiddleware(secretKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 1. Try to get the token from the cookie.
			cookie, err := r.Cookie("auth_token")
			if err != nil {
				// If the cookie is not found, redirect to the login page.
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}
			tokenString := cookie.Value

			// 2. Validate the token.
			userID, err := auth.ValidateToken(tokenString, []byte(secretKey))
			if err != nil {
				// If the token is invalid, redirect to the login page.
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}

			// 3. Add the user ID to the request's context.
			ctx := context.WithValue(r.Context(), userIDKey, userID)
			
			// 4. Call the next handler in the chain.
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}