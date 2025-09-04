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
    usernameRegex     = regexp.MustCompile(`^[a-zA-Z0-9_]{3,20}$`)
    phoneRegex        = regexp.MustCompile(`^\+?[0-9]{7,15}$`)
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

// validateInput ensures that username, phone, and password meet basic rules.
func validateInput(username, phone, password string) (string, string, string, string) {
    username = strings.TrimSpace(username)
    phone = strings.TrimSpace(phone)
    password = strings.TrimSpace(password)

    var errMsg string
    switch {
    case !usernameRegex.MatchString(username):
        errMsg = "Username must be 3-20 characters, alphanumeric or underscore."
    case !phoneRegex.MatchString(phone):
        errMsg = "Phone number format invalid."
    case len(password) < passwordMinLength:
        errMsg = "Password must be at least 8 characters."
    }
    return username, phone, password, errMsg
}

// convertToInterfaceMap transforms map[string]string into map[string]interface{}.
// As noted on StackOverflow, Go doesn't support casting here—looping is the idiomatic way. :contentReference[oaicite:0]{index=0}
func convertToInterfaceMap(strMap map[string]string) map[string]interface{} {
    ifaceMap := make(map[string]interface{}, len(strMap))
    for k, v := range strMap {
        ifaceMap[k] = v
    }
    return ifaceMap
}

// Register handles new user registrations, including form validation and rendering.
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
    if err := r.ParseForm(); err != nil {
        http.Error(w, "Invalid form data", http.StatusBadRequest)
        return
    }

    username, phone, password, errMsg := validateInput(
        r.FormValue("username"),
        r.FormValue("phone_number"),
        r.FormValue("password"),
    )
    if errMsg != "" {
        data := convertToInterfaceMap(map[string]string{"Error": errMsg})
        renderTemplate(w, "register.html", data)
        return
    }

    user := &domain.User{Username: username, PhoneNumber: phone}
    if _, err := h.UserService.RegisterUser(r.Context(), user, password); err != nil {
        log.Printf("Registration error: %v", err)
        data := convertToInterfaceMap(map[string]string{"Error": err.Error()})
        renderTemplate(w, "register.html", data)
        return
    }

    http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// Login validates user credentials, sets auth cookies, and redirects to chat.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
    if err := r.ParseForm(); err != nil {
        http.Error(w, "Invalid form data", http.StatusBadRequest)
        return
    }

    username := strings.TrimSpace(r.FormValue("username"))
    password := strings.TrimSpace(r.FormValue("password"))
    if username == "" || password == "" {
        data := convertToInterfaceMap(map[string]string{"Error": "Username and password are required."})
        renderTemplate(w, "login.html", data)
        return
    }

    token, err := h.UserService.Login(r.Context(), username, password)
    if err != nil {
        log.Printf("Login error: %v", err)
        data := convertToInterfaceMap(map[string]string{"Error": "Invalid username or password."})
        renderTemplate(w, "login.html", data)
        return
    }

    http.SetCookie(w, &http.Cookie{
        Name:     "auth_token",
        Value:    token,
        Expires:  time.Now().Add(24 * time.Hour),
        HttpOnly: true,
        Secure:   true,
        Path:     "/",
        SameSite: http.SameSiteLaxMode,
    })
    http.Redirect(w, r, "/chat", http.StatusSeeOther)
}
