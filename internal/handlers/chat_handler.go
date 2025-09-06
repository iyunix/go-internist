// File: internal/handlers/chat_handler.go
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/iyunix/go-internist/internal/domain"
	"github.com/iyunix/go-internist/internal/middleware"
	"github.com/iyunix/go-internist/internal/services"
	"github.com/iyunix/go-internist/internal/services/user_services"
)

type ChatHandler struct {
	UserService *user_services.UserService
	ChatService *services.ChatService
}
func NewChatHandler(userService *user_services.UserService, chatService *services.ChatService) (*ChatHandler, error) { // Changed to return (*ChatHandler, error)
	// --- ADDED VALIDATION ---
	if userService == nil {
		return nil, fmt.Errorf("user service is required for chat handler")
	}
	if chatService == nil {
		return nil, fmt.Errorf("chat service is required for chat handler")
	}
	// --- END ADDED VALIDATION ---

	return &ChatHandler{
		UserService: userService,
		ChatService: chatService,
	}, nil // Return the handler and a nil error on success
}
type TotalCreditsProvider interface {
	GetTotalCredits(ctx context.Context, userID uint) (int, error)
}

const defaultTotalCredits = 2500

func (h *ChatHandler) GetUserBalance(w http.ResponseWriter, r *http.Request) {
	// FIXED: Removed parentheses and extra string
	userID, ok := r.Context().Value(middleware.UserIDKey).(uint)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	balance, err := h.UserService.GetCharacterBalance(r.Context(), userID)
	if err != nil {
		log.Printf("[ChatHandler] Error getting user balance: %v", err)
		http.Error(w, "Failed to get balance", http.StatusInternalServerError)
		return
	}

	totalCredits := defaultTotalCredits
	if provider, ok := interface{}(h.UserService).(TotalCreditsProvider); ok {
		if total, err := provider.GetTotalCredits(r.Context(), userID); err == nil && total > 0 {
			totalCredits = total
		}
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]int{
		"balance":      balance,
		"totalCredits": totalCredits,
	})
}

func (h *ChatHandler) CreateChat(w http.ResponseWriter, r *http.Request) {
	// FIXED: Removed parentheses and extra string
	userID, ok := r.Context().Value(middleware.UserIDKey).(uint)
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

func (h *ChatHandler) StreamChatSSE(w http.ResponseWriter, r *http.Request) {
	// FIXED: Removed parentheses and extra string
	userID, ok := r.Context().Value(middleware.UserIDKey).(uint)
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
	if len(prompt) > domain.MaxQuestionLength {
		http.Error(w, fmt.Sprintf("Question too long. Maximum %d characters allowed", domain.MaxQuestionLength), http.StatusBadRequest)
		return
	}

	canAsk, chargeAmount, err := h.UserService.CanUserAskQuestion(r.Context(), userID, len(prompt))
	if err != nil {
		log.Printf("[ChatHandler] Error checking user balance for user %d: %v", userID, err)
		http.Error(w, "Error checking balance", http.StatusInternalServerError)
		return
	}
	if !canAsk {
		http.Error(w, fmt.Sprintf("Insufficient character balance. Need %d characters", chargeAmount), http.StatusPaymentRequired) // 402
		return
	}

	actualCharge, err := h.UserService.DeductCharactersForQuestion(r.Context(), userID, len(prompt))
	if err != nil {
		log.Printf("[ChatHandler] Error deducting characters for user %d: %v", userID, err)
		http.Error(w, "Error processing payment", http.StatusInternalServerError)
		return
	}
	log.Printf("[ChatHandler] User %d charged %d characters for question of length %d", userID, actualCharge, len(prompt))

	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
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
		if len(tokenBuffer) == 0 {
			return
		}
		chunk := strings.Join(tokenBuffer, "")
		if _, err := fmt.Fprintf(w, "data: %s\n\n", chunk); err != nil {
			return
		}
		flusher.Flush()
		tokenBuffer = tokenBuffer[:0]
		bufferSize = 0
	}

	onDelta := func(token string) error {
		tokenBuffer = append(tokenBuffer, token)
		bufferSize += len(token)
		if strings.ContainsAny(token, ".!?\n") || bufferSize >= maxBufferSize || len(tokenBuffer) >= maxTokens {
			flushBuffer()
		}
		return nil
	}

	onSources := func(sources []string) {
		sourcesMutex.Lock()
		documentSources = sources
		sourcesMutex.Unlock()
		if len(sources) > 0 {
			payload := map[string]any{"type": "sources", "sources": sources}
			if b, err := json.Marshal(payload); err == nil {
				fmt.Fprintf(w, "event: metadata\ndata: %s\n\n", b)
				flusher.Flush()
			}
		}
	}

	if err := h.ChatService.StreamChatMessageWithSources(r.Context(), userID, chatID, prompt, onDelta, onSources); err != nil {
		log.Printf("[ChatHandler] StreamChatMessageWithSources error for user %d chat %d: %v", userID, chatID, err)
		return
	}

	flushBuffer()

	sourcesMutex.Lock()
	if len(documentSources) > 0 {
		final := map[string]any{"type": "final_sources", "sources": documentSources}
		if b, err := json.Marshal(final); err == nil {
			fmt.Fprintf(w, "event: metadata\ndata: %s\n\n", b)
			flusher.Flush()
		}
	}
	sourcesMutex.Unlock()

	fmt.Fprintf(w, "event: done\ndata: \n\n")
	flusher.Flush()
	log.Printf("[ChatHandler] Gracefully closed stream for user %d", userID)
}

func (h *ChatHandler) GetUserChats(w http.ResponseWriter, r *http.Request) {
	// FIXED: Removed parentheses and extra string
	userID, ok := r.Context().Value(middleware.UserIDKey).(uint)
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
	// FIXED: Removed parentheses and extra string
	userID, ok := r.Context().Value(middleware.UserIDKey).(uint)
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

	chatIDU64, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil || chatIDU64 == 0 {
		http.Error(w, "Invalid chat id", http.StatusBadRequest)
		return
	}

	messages, err := h.ChatService.GetChatMessages(r.Context(), userID, uint(chatIDU64))
	if err != nil {
		log.Printf("[ChatHandler] GetChatMessages service error: %v", err)
		http.Error(w, "Failed to get messages", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(messages)
}

func (h *ChatHandler) DeleteChat(w http.ResponseWriter, r *http.Request) {
	// FIXED: Removed parentheses and extra string
	userID, ok := r.Context().Value(middleware.UserIDKey).(uint)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	vars := mux.Vars(r)
	chatIDU64, err := strconv.ParseUint(vars["id"], 10, 64)
	if err != nil || chatIDU64 == 0 {
		http.Error(w, "Invalid chat id", http.StatusBadRequest)
		return
	}

	if err := h.ChatService.DeleteChat(r.Context(), userID, uint(chatIDU64)); err != nil {
		http.Error(w, "Failed to delete chat", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}