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

	// --- ADD "admin.html" TO THIS LIST ---
	templates := []string{"index.html", "login.html", "register.html", "chat.html", "error.html", "verify_sms.html", "admin.html"}

	for _, tmpl := range templates {
		ts := template.New(tmpl)

		ts, err := ts.ParseFiles("web/templates/layout.html")
		if err != nil {
			log.Fatalf("Error parsing layout for %s: %v", tmpl, err)
		}

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

	err := t.ExecuteTemplate(w, "layout.html", data)
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

// Dummy CSRF token generator
func generateCSRFToken() string {
	return "csrf-token-placeholder"
}

type PageHandler struct{}

func NewPageHandler() *PageHandler {
	return &PageHandler{}
}

func (h *PageHandler) ShowIndexPage(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "index.html", nil)
}

func (h *PageHandler) ShowLoginPage(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "login.html", nil)
}

func (h *PageHandler) ShowRegisterPage(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "register.html", nil)
}

func (h *PageHandler) ShowVerifySMSPage(w http.ResponseWriter, r *http.Request) {
	phone := r.URL.Query().Get("phone")
	data := map[string]interface{}{"PhoneNumber": phone}
	renderTemplate(w, "verify_sms.html", data)
}

// ShowAdminPage renders the admin panel.
func (h *PageHandler) ShowAdminPage(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "admin.html", nil)
}

func (h *PageHandler) ShowChatPage(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(middleware.UserIDKey).(uint)
	data := map[string]interface{}{
		"UserID": userID,
	}
	renderTemplate(w, "chat.html", data)
}

func (h *PageHandler) ShowErrorPage(w http.ResponseWriter, code, message, description string) {
	data := map[string]interface{}{
		"Code":        code,
		"Message":     message,
		"Description": description,
	}
	renderTemplate(w, "error.html", data)
}