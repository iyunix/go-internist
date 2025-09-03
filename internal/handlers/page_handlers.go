// File: internal/handlers/page_handlers.go
package handlers

import (
    "html/template"
    "log"
    "net/http"
    "sync"

    "github.com/iyunix/go-internist/internal/middleware"
)

// Template cache to avoid parsing templates on every request
var (
    templates     *template.Template
    templatesOnce sync.Once
)

// loadTemplates loads and caches all templates (call once, thread-safe)
func loadTemplates() {
    templates = template.Must(template.ParseGlob("web/templates/*.html"))
}

// Adds basic security headers for all responses
func addSecurityHeaders(w http.ResponseWriter) {
    w.Header().Set("Content-Security-Policy", "default-src 'self'")
    w.Header().Set("X-Frame-Options", "DENY")
    w.Header().Set("X-Content-Type-Options", "nosniff")
    w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
}

// Dummy CSRF token generator (in production use a proper CSRF library)
func generateCSRFToken() string {
    // Replace with cryptographically secure random generator in real deployment
    return "csrf-token-placeholder"
}

type PageHandler struct{}

func NewPageHandler() *PageHandler {
    return &PageHandler{}
}

// renderTemplate uses template cache and injects CSRF/security headers
func renderTemplate(w http.ResponseWriter, tmpl string, data map[string]interface{}) {
    templatesOnce.Do(loadTemplates)
    addSecurityHeaders(w)

    // Add CSRF to template data
    if data == nil {
        data = make(map[string]interface{})
    }
    data["CSRFToken"] = generateCSRFToken()

    err := templates.ExecuteTemplate(w, tmpl, data)
    if err != nil {
        log.Printf("Template render error for %s: %v", tmpl, err)
        http.Error(w, "Error rendering page", http.StatusInternalServerError)
    }
}

// ShowLoginPage calls the helper function
func (h *PageHandler) ShowLoginPage(w http.ResponseWriter, r *http.Request) {
    renderTemplate(w, "login.html", nil)
}

// ShowRegisterPage calls the helper function
func (h *PageHandler) ShowRegisterPage(w http.ResponseWriter, r *http.Request) {
    renderTemplate(w, "register.html", nil)
}

// ShowChatPage renders the main chat interface with user context
func (h *PageHandler) ShowChatPage(w http.ResponseWriter, r *http.Request) {
    userID := r.Context().Value(middleware.UserIDKey("userID"))
    data := map[string]interface{}{
        "UserID": userID,
    }
    renderTemplate(w, "chat.html", data)
}
