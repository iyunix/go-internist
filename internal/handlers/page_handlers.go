// File: internal/handlers/page_handlers.go
package handlers

import (
	"bytes"
	"encoding/json"
	"html/template"
	"log"
	"os"
	"net/http"
    "path/filepath"
	"strconv"
	"sync"

	"github.com/iyunix/go-internist/internal/domain"
	"github.com/iyunix/go-internist/internal/middleware"
	"github.com/iyunix/go-internist/internal/services"
	"github.com/iyunix/go-internist/internal/services/admin_services"
	"github.com/iyunix/go-internist/internal/services/user_services"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
)

var (
	templates map[string]*template.Template
	once      sync.Once
)

var funcMap = template.FuncMap{
    "subtract": func(a, b int) int {
        return a - b
    },
    "percentage": func(a, b int) float64 {
        if b == 0 { return 0 }
        return (float64(a) / float64(b)) * 100
    },
    "json": func(v interface{}) (template.JS, error) {
        b, err := json.Marshal(v)
        if err != nil {
            return "", err
        }
        return template.JS(b), nil
    },
}


type RenderedMessage struct {
	domain.Message
	RenderedContent template.HTML
}

func loadTemplates() {
    templateDir := findTemplateDir()
    layoutFile := "layout.html"
    partialsDir := filepath.Join(templateDir, "partials")
    
    templates = make(map[string]*template.Template)

    partials, err := filepath.Glob(filepath.Join(partialsDir, "*.html"))
    if err != nil {
        log.Printf("Warning: could not find partial templates: %v", err)
    }

    pages, err := filepath.Glob(filepath.Join(templateDir, "*.html"))
    if err != nil {
        log.Fatalf("Error finding page templates: %v", err)
    }
    if len(pages) == 0 {
        log.Fatalf("No page templates found in %s directory", templateDir)
    }
    
    layoutPath := filepath.Join(templateDir, layoutFile)

	for _, pagePath := range pages {
		filename := filepath.Base(pagePath)
		if filename == layoutFile {
			continue
		}
		filesToParse := []string{pagePath, layoutPath}
		filesToParse = append(filesToParse, partials...)

		ts, err := template.New(filename).Funcs(funcMap).ParseFiles(filesToParse...)
		if err != nil {
			log.Fatalf("Error parsing template %s: %v", filename, err)
		}
		templates[filename] = ts
	}
	log.Printf("Successfully loaded and cached %d page templates with %d partials.", len(templates), len(partials))
}


func RenderTemplate(w http.ResponseWriter, tmplName string, data map[string]interface{}) {
	once.Do(loadTemplates)
	t, ok := templates[tmplName]
	if !ok {
		http.Error(w, "The template "+tmplName+" does not exist.", http.StatusInternalServerError)
		return
	}
	err := t.ExecuteTemplate(w, "layout.html", data)
	if err != nil {
		log.Printf("Error executing template %s: %v", tmplName, err)
		http.Error(w, "Error rendering template", http.StatusInternalServerError)
	}
}

type PageHandler struct {
	UserService  *user_services.UserService
	ChatService  *services.ChatService
	AdminService *admin_services.AdminService
}

func NewPageHandler(us *user_services.UserService, cs *services.ChatService, as *admin_services.AdminService) *PageHandler {
	return &PageHandler{
		UserService:  us,
		ChatService:  cs,
		AdminService: as,
	}
}

// ... (The rest of your handler functions remain correct and unchanged) ...
func (h *PageHandler) ShowIndexPage(w http.ResponseWriter, r *http.Request) {
	RenderTemplate(w, "index.html", nil)
}

func (h *PageHandler) ShowLoginPage(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Verified":     r.URL.Query().Get("verified") == "true",
		"Error":        r.URL.Query().Get("error"),
		"ResetSuccess": r.URL.Query().Get("reset") == "success",
	}
	RenderTemplate(w, "login.html", data)
}

func (h *PageHandler) ShowRegisterPage(w http.ResponseWriter, r *http.Request) {
	RenderTemplate(w, "register.html", nil)
}

func (h *PageHandler) ShowVerifySMSPage(w http.ResponseWriter, r *http.Request) {
	phone := r.URL.Query().Get("phone")
	action := r.URL.Query().Get("action") // <-- ADD THIS LINE

	data := map[string]interface{}{
		"PhoneNumber": phone,
		"Action":      action, // <-- AND PASS IT HERE
	}
	RenderTemplate(w, "verify_sms.html", data)
}

func (h *PageHandler) ShowForgotPasswordPage(w http.ResponseWriter, r *http.Request) {
	RenderTemplate(w, "forgot_password.html", nil)
}

func (h *PageHandler) ShowResetPasswordPage(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Phone": r.URL.Query().Get("phone"),
		"Code":  r.URL.Query().Get("code"),
		"Error": r.URL.Query().Get("error"),
	}
	RenderTemplate(w, "reset_password.html", data)
}

func (h *PageHandler) ShowAdminPage(w http.ResponseWriter, r *http.Request) {
	users, _, err := h.AdminService.GetAllUsers(r.Context(), 1, 10, "")
	if err != nil {
		log.Printf("Error fetching users for admin page: %v", err)
		users = []domain.User{}
	}
	data := map[string]interface{}{
		"Users": users,
	}
	RenderTemplate(w, "admin.html", data)
}

func (h *PageHandler) ShowChatPage(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(uint)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	user, err := h.UserService.GetUserByID(r.Context(), userID)
	if err != nil {
		http.Redirect(w, r, "/login?error=user_not_found", http.StatusSeeOther)
		return
	}
	chats, err := h.ChatService.GetUserChats(r.Context(), userID)
	if err != nil {
		log.Printf("Error fetching chats for user %d: %v", userID, err)
		chats = []domain.Chat{}
	}
	activeChatIDStr := r.URL.Query().Get("id")
	var activeChatID uint64
	var renderedMessages []RenderedMessage
	if activeChatIDStr != "" {
		activeChatID, _ = strconv.ParseUint(activeChatIDStr, 10, 64)
		messages, err := h.ChatService.GetChatMessages(r.Context(), userID, uint(activeChatID))
		if err != nil {
			log.Printf("Error fetching messages for chat %d: %v", activeChatID, err)
		} else {
			mdParser := goldmark.New(
				goldmark.WithExtensions(extension.GFM),
				goldmark.WithRendererOptions(html.WithHardWraps()),
			)
			for _, msg := range messages {
				var buf bytes.Buffer
				if msg.MessageType == "assistant" {
					if err := mdParser.Convert([]byte(msg.Content), &buf); err == nil {
						renderedMessages = append(renderedMessages, RenderedMessage{
							Message:         msg,
							RenderedContent: template.HTML(buf.String()),
						})
					}
				} else {
					renderedMessages = append(renderedMessages, RenderedMessage{
						Message:         msg,
						RenderedContent: template.HTML(template.HTMLEscapeString(msg.Content)),
					})
				}
			}
		}
	}
	data := map[string]interface{}{
		"User":         user,
		"Chats":        chats,
		"Messages":     renderedMessages,
		"ActiveChatID": uint(activeChatID),
	}
	RenderTemplate(w, "chat.html", data)
}

func (h *PageHandler) ShowErrorPage(w http.ResponseWriter, code, message, description string) {
	w.WriteHeader(http.StatusNotFound)
	data := map[string]interface{}{
		"Code":        code,
		"Message":     message,
		"Description": description,
	}
	RenderTemplate(w, "error.html", data)
}


// findTemplateDir looks for web/templates in multiple possible locations
func findTemplateDir() string {
    possiblePaths := []string{
        "web/templates",           // Current directory
        "../web/templates",        // Parent directory  
        "../../web/templates",     // Two levels up (for cmd/server)
    }
    
    for _, path := range possiblePaths {
        if _, err := os.Stat(path); err == nil {
            log.Printf("Found templates directory at: %s", path)
            return path
        }
    }
    
    // Fallback to default
    log.Printf("Warning: using fallback template path - templates may not load")
    return "web/templates"
}
