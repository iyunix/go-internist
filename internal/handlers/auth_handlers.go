// File: internal/handlers/auth_handlers.go
package handlers

import (
	"log"
	"net/http"
	"time"

	"github.com/iyunix/go-internist/internal/domain"
	"github.com/iyunix/go-internist/internal/services"
)

// AuthHandler holds the dependencies for authentication handlers.
type AuthHandler struct {
	UserService *services.UserService
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(service *services.UserService) *AuthHandler {
	return &AuthHandler{UserService: service}
}

// Register handles the user registration form submission.
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	username := r.FormValue("username")
	phone := r.FormValue("phone_number")
	password := r.FormValue("password")

	user := &domain.User{Username: username, PhoneNumber: phone}

	_, err := h.UserService.RegisterUser(r.Context(), user, password)
	if err != nil {
		log.Printf("Registration error: %v", err)
		data := map[string]string{"Error": err.Error()}
		renderTemplate(w, "register.html", data) // Uses the helper from the other file
		return
	}

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// Login handles the user login form submission.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	token, err := h.UserService.Login(r.Context(), username, password)
	if err != nil {
		log.Printf("Login error: %v", err)
		data := map[string]string{"Error": "Invalid username or password."}
		renderTemplate(w, "login.html", data) // Uses the helper from the other file
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    token,
		Expires:  time.Now().Add(24 * time.Hour),
		HttpOnly: true,
		Path:     "/",
	})

	http.Redirect(w, r, "/chat", http.StatusSeeOther)
}