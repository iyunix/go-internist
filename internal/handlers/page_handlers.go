// File: internal/handlers/page_handlers.go
package handlers

import (
	"html/template"
	"net/http"

	"github.com/iyunix/go-internist/internal/middleware"
)

// PageHandler is the struct for handlers that render HTML pages.
type PageHandler struct{}

// NewPageHandler creates a new PageHandler.
func NewPageHandler() *PageHandler {
	return &PageHandler{}
}

// renderTemplate is now a package-level helper function.
func renderTemplate(w http.ResponseWriter, tmpl string, data interface{}) {
	t, err := template.ParseFiles("web/templates/layout.html", "web/templates/"+tmpl)
	if err != nil {
		http.Error(w, "Error parsing templates", http.StatusInternalServerError)
		return
	}

	err = t.ExecuteTemplate(w, "layout.html", data)
	if err != nil {
		http.Error(w, "Error rendering page", http.StatusInternalServerError)
	}
}

// ShowLoginPage now calls the helper function.
func (h *PageHandler) ShowLoginPage(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "login.html", nil)
}

// ShowRegisterPage also calls the helper.
func (h *PageHandler) ShowRegisterPage(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "register.html", nil)
}

// ShowChatPage renders the main chat interface.
func (h *PageHandler) ShowChatPage(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(middleware.UserIDKey("userID"))
	data := map[string]interface{}{
		"UserID": userID,
	}
	renderTemplate(w, "chat.html", data)
}