// File: internal/handlers/chat_handler.go
package handlers

import (
    "encoding/json"
    "log"
    "net/http"
    "strconv"
    "strings"

    "github.com/gorilla/mux"
    "github.com/iyunix/go-internist/internal/domain"
    "github.com/iyunix/go-internist/internal/middleware"
    "github.com/iyunix/go-internist/internal/services"
)

const (
    maxChatTitleLen    = 100
    maxMessageLen      = 2048
    defaultPageSize    = 20
    maxPageSize        = 100
)

type ChatHandler struct {
    UserService *services.UserService
    ChatService *services.ChatService
}

func NewChatHandler(userService *services.UserService, chatService *services.ChatService) *ChatHandler {
    return &ChatHandler{
        UserService: userService,
        ChatService: chatService,
    }
}

// Validate and sanitize chat title/message content
func sanitizeChatTitle(t string) string {
    t = strings.TrimSpace(t)
    if len(t) > maxChatTitleLen {
        t = t[:maxChatTitleLen]
    }
    return t
}
func sanitizeMessageContent(c string) string {
    c = strings.TrimSpace(c)
    if len(c) > maxMessageLen {
        c = c[:maxMessageLen]
    }
    return c
}

// GetUserChats returns paginated chats for the user
func (h *ChatHandler) GetUserChats(w http.ResponseWriter, r *http.Request) {
    userID, ok := r.Context().Value(middleware.UserIDKey("userID")).(uint)
    if !ok || userID == 0 {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }
    // Read pagination params
    pageStr := r.URL.Query().Get("page")
    pageSizeStr := r.URL.Query().Get("page_size")
    page, _ := strconv.Atoi(pageStr)
    if page < 1 {
        page = 1
    }
    pageSize, _ := strconv.Atoi(pageSizeStr)
    if pageSize < 1 || pageSize > maxPageSize {
        pageSize = defaultPageSize
    }

    allChats, err := h.ChatService.GetUserChats(r.Context(), userID)
    if err != nil {
        log.Printf("[ChatHandler] GetUserChats failed for user %d: %v", userID, err)
        http.Error(w, "Unable to fetch chats", http.StatusInternalServerError)
        return
    }
    startIdx := (page - 1) * pageSize
    endIdx := startIdx + pageSize
    if startIdx > len(allChats) {
        startIdx = len(allChats)
    }
    if endIdx > len(allChats) {
        endIdx = len(allChats)
    }
    pagedChats := allChats[startIdx:endIdx]

    json.NewEncoder(w).Encode(pagedChats)
}

// HandleChatMessage creates a message with input sanitation and audit logging
func (h *ChatHandler) HandleChatMessage(w http.ResponseWriter, r *http.Request) {
    userID, ok := r.Context().Value(middleware.UserIDKey("userID")).(uint)
    if !ok || userID == 0 {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }
    var req struct {
        ChatID   uint   `json:"chat_id"`
        Content  string `json:"content"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid data", http.StatusBadRequest)
        return
    }
    req.Content = sanitizeMessageContent(req.Content)
    if req.Content == "" {
        http.Error(w, "Message content required", http.StatusBadRequest)
        return
    }

    // Log the new message action (audit trail)
    log.Printf("[ChatHandler] User %d posted message to chat %d", userID, req.ChatID)

    response, err := h.ChatService.AddChatMessage(r.Context(), userID, req.ChatID, req.Content)
    if err != nil {
        log.Printf("[ChatHandler] AddChatMessage error user %d chat %d: %v", userID, req.ChatID, err)
        http.Error(w, "Failed to send message", http.StatusInternalServerError)
        return
    }

    json.NewEncoder(w).Encode(response)
}

// GetChatMessages returns paginated messages for a chat
func (h *ChatHandler) GetChatMessages(w http.ResponseWriter, r *http.Request) {
    userID, ok := r.Context().Value(middleware.UserIDKey("userID")).(uint)
    if !ok || userID == 0 {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }
    chatID, _ := strconv.Atoi(mux.Vars(r)["id"])
    pageStr := r.URL.Query().Get("page")
    pageSizeStr := r.URL.Query().Get("page_size")
    page, _ := strconv.Atoi(pageStr)
    if page < 1 {
        page = 1
    }
    pageSize, _ := strconv.Atoi(pageSizeStr)
    if pageSize < 1 || pageSize > maxPageSize {
        pageSize = defaultPageSize
    }

    messages, err := h.ChatService.GetChatMessages(r.Context(), userID, uint(chatID))
    if err != nil {
        log.Printf("[ChatHandler] GetChatMessages error user %d chat %d: %v", userID, chatID, err)
        http.Error(w, "Failed to fetch messages", http.StatusInternalServerError)
        return
    }
    startIdx := (page - 1) * pageSize
    endIdx := startIdx + pageSize
    if startIdx > len(messages) {
        startIdx = len(messages)
    }
    if endIdx > len(messages) {
        endIdx = len(messages)
    }
    pagedMessages := messages[startIdx:endIdx]

    json.NewEncoder(w).Encode(pagedMessages)
}

// DeleteChat validates ownership and logs deletion
func (h *ChatHandler) DeleteChat(w http.ResponseWriter, r *http.Request) {
    userID, ok := r.Context().Value(middleware.UserIDKey("userID")).(uint)
    if !ok || userID == 0 {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }
    chatID, _ := strconv.Atoi(mux.Vars(r)["id"])

    log.Printf("[ChatHandler] User %d deleting chat %d", userID, chatID)
    err := h.ChatService.DeleteChat(r.Context(), userID, uint(chatID))
    if err != nil {
        log.Printf("[ChatHandler] DeleteChat error user %d chat %d: %v", userID, chatID, err)
        http.Error(w, "Failed to delete chat", http.StatusInternalServerError)
        return
    }
    w.WriteHeader(http.StatusNoContent)
}
