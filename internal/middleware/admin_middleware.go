// File: internal/middleware/admin_middleware.go
package middleware

import (
	"log"
	"net/http"

	"github.com/iyunix/go-internist/internal/repository"
)

// RequireAdmin is a middleware that checks if the authenticated user has admin privileges.
// It MUST be used AFTER the standard JWT authentication middleware.
func RequireAdmin(userRepo repository.UserRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 1. Get the userID from the context. The JWT middleware should have already placed it there.
			userID, ok := r.Context().Value(UserIDKey).(uint)
			if !ok || userID == 0 {
				// This indicates a problem with the auth setup or the token is missing claims.
				// The user is not properly authenticated to even check for admin status.
				log.Printf("[AdminMiddleware] Forbidden: Could not find valid userID in context for path %s", r.URL.Path)
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			// 2. Fetch the full user object from the database to check their status.
			user, err := userRepo.FindByID(r.Context(), userID)
			if err != nil {
				// This could happen if the user was deleted after their token was issued.
				log.Printf("[AdminMiddleware] Forbidden: Could not find user with ID %d from token. Error: %v", userID, err)
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			// 3. The core logic: check the IsAdmin flag.
			if !user.IsAdmin {
				// The user is logged in, but they are NOT an admin.
				log.Printf("[AdminMiddleware] FORBIDDEN: Non-admin user '%s' (ID: %d) attempted to access admin route: %s", user.Username, user.ID, r.URL.Path)
				http.Error(w, "Forbidden: You do not have permission to access this page.", http.StatusForbidden)
				return
			}

			// 4. If we reach here, the user is a verified admin. Allow the request to proceed.
			log.Printf("[AdminMiddleware] GRANTED: Admin user '%s' (ID: %d) accessed admin route: %s", user.Username, user.ID, r.URL.Path)
			next.ServeHTTP(w, r)
		})
	}
}