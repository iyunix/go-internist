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
    templateCache     map[string]*template.Template
    templateCacheOnce sync.Once
)

// loadTemplateCache creates separate template sets for each page
func loadTemplateCache() {
    templateCache = make(map[string]*template.Template)
    
    // List of templates that need individual caching
    templates := []string{"login.html", "register.html", "chat.html"}
    
    for _, tmpl := range templates {
        ts := template.New(tmpl)
        
        // Parse layout first
        ts, err := ts.ParseFiles("web/templates/layout.html")
        if err != nil {
            log.Fatalf("Error parsing layout for %s: %v", tmpl, err)
        }
        
        // Parse the specific template
        ts, err = ts.ParseFiles("web/templates/" + tmpl)
        if err != nil {
            log.Fatalf("Error parsing %s: %v", tmpl, err)
        }
        
        templateCache[tmpl] = ts
    }
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

// renderTemplate uses individual template cache and injects CSRF/security headers
func renderTemplate(w http.ResponseWriter, tmpl string, data map[string]interface{}) {
    templateCacheOnce.Do(loadTemplateCache)
    addSecurityHeaders(w)

    // Add CSRF to template data
    if data == nil {
        data = make(map[string]interface{})
    }
    data["CSRFToken"] = generateCSRFToken()

    t, ok := templateCache[tmpl]
    if !ok {
        log.Printf("Template %s not found in cache", tmpl)
        http.Error(w, "Template not found", http.StatusInternalServerError)
        return
    }

    err := t.ExecuteTemplate(w, tmpl, data)
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
