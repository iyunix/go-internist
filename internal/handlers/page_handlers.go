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

	// CHANGED: Added "verify_sms.html" to the list of templates to be cached.
	templates := []string{"index.html", "login.html", "register.html", "chat.html", "error.html", "verify_sms.html"}

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

// ShowIndexPage renders the landing page
func (h *PageHandler) ShowIndexPage(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "index.html", nil)
}

// ShowLoginPage renders the login page
func (h *PageHandler) ShowLoginPage(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "login.html", nil)
}

// ShowRegisterPage renders the registration page
func (h *PageHandler) ShowRegisterPage(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "register.html", nil)
}

// NEW: This is the handler method our main.go needs to call.
// ShowVerifySMSPage renders the page for SMS code verification.
func (h *PageHandler) ShowVerifySMSPage(w http.ResponseWriter, r *http.Request) {
	phone := r.URL.Query().Get("phone")
	data := map[string]interface{}{"PhoneNumber": phone}
	renderTemplate(w, "verify_sms.html", data)
}

// ShowChatPage renders the main chat interface with user context
func (h *PageHandler) ShowChatPage(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(middleware.UserIDKey("userID"))
	data := map[string]interface{}{
		"UserID": userID,
	}
	renderTemplate(w, "chat.html", data)
}

// ShowErrorPage renders the error page with custom code/message/description
func (h *PageHandler) ShowErrorPage(w http.ResponseWriter, code, message, description string) {
	data := map[string]interface{}{
		"Code":        code,
		"Message":     message,
		"Description": description,
	}
	renderTemplate(w, "error.html", data)
}