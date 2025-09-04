// In: internal/middleware/recovery.go

package middleware

import (
	"log"
	"net/http"
)

func RecoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// This deferred function will execute after the handler,
		// or immediately if a panic occurs.
		defer func() {
			if err := recover(); err != nil {
				log.Printf("[PANIC] %v", err)

				// Print the stack trace here for detailed debugging
				// debug.PrintStack() 

				w.Header().Set("Connection", "close")
				http.Error(w, "Something went wrong on our end.", http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, r)
	})
}