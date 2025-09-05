package handlers

import (
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "strconv"
    "strings"
    "sync"
    "time"

    "github.com/gorilla/mux"
    "github.com/iyunix/go-internist/internal/middleware"
    "github.com/iyunix/go-internist/internal/services"
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

// CreateChat handles creating a new chat record
func (h *ChatHandler) CreateChat(w http.ResponseWriter, r *http.Request) {
    userID, ok := r.Context().Value(middleware.UserIDKey("userID")).(uint)
    if !ok {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }

    var req struct {
        Title string `json:"title"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }

    chat, err := h.ChatService.CreateChat(r.Context(), userID, req.Title)
    if err != nil {
        log.Printf("[ChatHandler] Error calling CreateChat service: %v", err)
        http.Error(w, "Failed to create chat", http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json; charset=utf-8")
    w.Header().Set("Cache-Control", "no-store")
    w.WriteHeader(http.StatusCreated)
    _ = json.NewEncoder(w).Encode(chat)
}

// StreamChatSSE handles SSE chat streaming
func (h *ChatHandler) StreamChatSSE(w http.ResponseWriter, r *http.Request) {
    userID, ok := r.Context().Value(middleware.UserIDKey("userID")).(uint)
    if !ok {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }

    vars := mux.Vars(r)
    idStr, ok := vars["id"]
    if !ok {
        http.Error(w, "Missing chat id in URL", http.StatusBadRequest)
        return
    }
    id64, err := strconv.ParseUint(idStr, 10, 64)
    if err != nil || id64 == 0 {
        http.Error(w, "Invalid chat id", http.StatusBadRequest)
        return
    }
    chatID := uint(id64)

    prompt := r.URL.Query().Get("q")
    if prompt == "" {
        http.Error(w, "Missing query parameter: q", http.StatusBadRequest)
        return
    }

    // Set SSE headers
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")
    w.Header().Set("X-Accel-Buffering", "no")

    flusher, ok := w.(http.Flusher)
    if !ok {
        http.Error(w, "Streaming unsupported on this connection", http.StatusInternalServerError)
        return
    }

    done := r.Context().Done()
    hb := time.NewTicker(15 * time.Second)
    defer hb.Stop()

    // Heartbeat
    go func() {
        for {
            select {
            case <-hb.C:
                fmt.Fprint(w, ": ping\n\n")
                flusher.Flush()
            case <-done:
                return
            }
        }
    }()

    var documentSources []string
    var sourcesMutex sync.Mutex

    var tokenBuffer []string
    bufferSize := 0
    const maxBufferSize = 50
    const maxTokens = 5

    flushBuffer := func() {
        if len(tokenBuffer) > 0 {
            chunk := strings.Join(tokenBuffer, "")
            if _, err := fmt.Fprintf(w, "data: %s\n\n", chunk); err != nil {
                return
            }
            flusher.Flush()
            tokenBuffer = tokenBuffer[:0]
            bufferSize = 0
        }
    }

    onDelta := func(token string) error {
        tokenBuffer = append(tokenBuffer, token)
        bufferSize += len(token)

        if strings.ContainsAny(token, ".!?\n") ||
            bufferSize >= maxBufferSize ||
            len(tokenBuffer) >= maxTokens {
            flushBuffer()
        }
        return nil
    }

    onSources := func(sources []string) {
        sourcesMutex.Lock()
        documentSources = sources
        sourcesMutex.Unlock()

        if len(sources) > 0 {
            sourcesData := map[string]interface{}{
                "type":    "sources",
                "sources": sources,
            }
            sourcesJSON, _ := json.Marshal(sourcesData)
            fmt.Fprintf(w, "event: metadata\ndata: %s\n\n", sourcesJSON)
            flusher.Flush()
        }
    }

    if err := h.ChatService.StreamChatMessageWithSources(r.Context(), userID, chatID, prompt, onDelta, onSources); err != nil {
        log.Printf("[ChatHandler] StreamChatMessageWithSources error for user %d chat %d: %v", userID, chatID, err)
        return
    }

    flushBuffer()

    sourcesMutex.Lock()
    if len(documentSources) > 0 {
        finalData := map[string]interface{}{
            "type":    "final_sources",
            "sources": documentSources,
        }
        finalJSON, _ := json.Marshal(finalData)
        fmt.Fprintf(w, "event: metadata\ndata: %s\n\n", finalJSON)
        flusher.Flush()
    }
    sourcesMutex.Unlock()

    fmt.Fprintf(w, "event: done\ndata: \n\n")
    flusher.Flush()
    log.Printf("[ChatHandler] Gracefully closed stream for user %d", userID)
}

func (h *ChatHandler) GetUserChats(w http.ResponseWriter, r *http.Request) {
    userID, ok := r.Context().Value(middleware.UserIDKey("userID")).(uint)
    if !ok {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }
    chats, err := h.ChatService.GetUserChats(r.Context(), userID)
    if err != nil {
        http.Error(w, "Failed to get user chats", http.StatusInternalServerError)
        return
    }
    w.Header().Set("Content-Type", "application/json; charset=utf-8")
    w.Header().Set("Cache-Control", "no-store")
    _ = json.NewEncoder(w).Encode(chats)
}

func (h *ChatHandler) GetChatMessages(w http.ResponseWriter, r *http.Request) {
    userID, ok := r.Context().Value(middleware.UserIDKey("userID")).(uint)
    if !ok {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }
    vars := mux.Vars(r)
    chatID, _ := strconv.ParseUint(vars["id"], 10, 64)
    messages, err := h.ChatService.GetChatMessages(r.Context(), userID, uint(chatID))
    if err != nil {
        http.Error(w, "Failed to get messages", http.StatusInternalServerError)
        return
    }
    w.Header().Set("Content-Type", "application/json; charset=utf-8")
    w.Header().Set("Cache-Control", "no-store")
    _ = json.NewEncoder(w).Encode(messages)
}

func (h *ChatHandler) DeleteChat(w http.ResponseWriter, r *http.Request) {
    userID, ok := r.Context().Value(middleware.UserIDKey("userID")).(uint)
    if !ok {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }
    vars := mux.Vars(r)
    chatID, _ := strconv.ParseUint(vars["id"], 10, 64)
    if err := h.ChatService.DeleteChat(r.Context(), userID, uint(chatID)); err != nil {
        http.Error(w, "Failed to delete chat", http.StatusInternalServerError)
        return
    }
    w.WriteHeader(http.StatusNoContent)
}
