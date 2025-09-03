// File: internal/handlers/auth_handlers.go
package handlers

import (
    "log"
    "net/http"
    "regexp"
    "strings"
    "time"

    "github.com/iyunix/go-internist/internal/domain"
    "github.com/iyunix/go-internist/internal/services"
)

var (
    usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]{3,20}$`)
    phoneRegex    = regexp.MustCompile(`^\+?[0-9]{7,15}$`)
    passwordMinLength = 8
)

// AuthHandler holds the dependencies for authentication handlers.
type AuthHandler struct {
    UserService *services.UserService
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(service *services.UserService) *AuthHandler {
    return &AuthHandler{UserService: service}
}

// validateInput performs basic input validation and sanitization.
func validateInput(username, phone, password string) (string, string, string, string) {
    username = strings.TrimSpace(username)
    phone = strings.TrimSpace(phone)
    password = strings.TrimSpace(password)

    var errMsg string
    if !usernameRegex.MatchString(username) {
        errMsg = "Username must be 3-20 characters, alphanumeric or underscore."
    } else if !phoneRegex.MatchString(phone) {
        errMsg = "Phone number format invalid."
    } else if len(password) < passwordMinLength {
        errMsg = "Password must be at least 8 characters."
    }
    return username, phone, password, errMsg
}

// Register handles user registration with enhanced validation.
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
    if err := r.ParseForm(); err != nil {
        http.Error(w, "Invalid form data", http.StatusBadRequest)
        return
    }

    username := r.FormValue("username")
    phone := r.FormValue("phone_number")
    password := r.FormValue("password")

    username, phone, password, errMsg := validateInput(username, phone, password)
    if errMsg != "" {
        data := map[string]string{"Error": errMsg}
        renderTemplate(w, "register.html", data)
        return
    }

    user := &domain.User{Username: username, PhoneNumber: phone}

    // Check for brute force/rate limit here (implementation dependent)
    // e.g., check IP-based attempts or CAPTCHA integration

    _, err := h.UserService.RegisterUser(r.Context(), user, password)
    if err != nil {
        log.Printf("Registration error: %v", err)
        data := map[string]string{"Error": err.Error()}
        renderTemplate(w, "register.html", data)
        return
    }

    http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// Login handles user login with simple validation.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
    if err := r.ParseForm(); err != nil {
        http.Error(w, "Invalid form data", http.StatusBadRequest)
        return
    }

    username := strings.TrimSpace(r.FormValue("username"))
    password := strings.TrimSpace(r.FormValue("password"))

    if username == "" || password == "" {
        data := map[string]string{"Error": "Username and password are required."}
        renderTemplate(w, "login.html", data)
        return
    }

    // Rate limiting or CAPTCHA can be enforced here to prevent brute force

    token, err := h.UserService.Login(r.Context(), username, password)
    if err != nil {
        log.Printf("Login error: %v", err)
        data := map[string]string{"Error": "Invalid username or password."}
        renderTemplate(w, "login.html", data)
        return
    }

    http.SetCookie(w, &http.Cookie{
        Name:     "auth_token",
        Value:    token,
        Expires:  time.Now().Add(24 * time.Hour),
        HttpOnly: true,
        Secure:   true,               // Add Secure for HTTPS only
        Path:     "/",
        SameSite: http.SameSiteLaxMode,
    })

    http.Redirect(w, r, "/chat", http.StatusSeeOther)
}
