// File: internal/handlers/chat_handler.go
package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/iyunix/go-internist/internal/middleware"
	"github.com/iyunix/go-internist/internal/services"
)

type ChatHandler struct {
	UserService *services.UserService
	ChatService *services.ChatService
}

func NewChatHandler(us *services.UserService, cs *services.ChatService) *ChatHandler {
	return &ChatHandler{
		UserService: us,
		ChatService: cs,
	}
}

// GetUserChats handles the request to retrieve all chat histories for a user.
func (h *ChatHandler) GetUserChats(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey("userID")).(uint)
	if !ok {
		writeError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	chats, err := h.ChatService.GetUserChats(r.Context(), userID)
	if err != nil {
		writeError(w, "Could not retrieve chats", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, chats)
}

// GetChatMessages handles the request to retrieve all messages for a specific chat.
func (h *ChatHandler) GetChatMessages(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey("userID")).(uint)
	if !ok {
		writeError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	chatID, err := strconv.ParseUint(vars["id"], 10, 32)
	if err != nil {
		writeError(w, "Invalid chat ID", http.StatusBadRequest)
		return
	}

	messages, err := h.ChatService.GetChatMessages(r.Context(), uint(chatID), userID)
	if err != nil {
		if err.Error() == "unauthorized" {
			writeError(w, "Unauthorized", http.StatusForbidden)
			return
		}
		writeError(w, "Could not retrieve messages", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, messages)
}

// File: internal/handlers/chat_handler.go
// ... (keep everything else, just replace this one function)
func (h *ChatHandler) HandleChatMessage(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey("userID")).(uint)
	if !ok {
		writeError(w, "Unauthorized", http.StatusUnauthorized); return
	}

	vars := mux.Vars(r)
	chatID, _ := strconv.ParseUint(vars["id"], 10, 32)

	var req struct { Message string `json:"message"` }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Message == "" {
		writeError(w, "Bad Request", http.StatusBadRequest); return
	}

	// Call the new high-level service function
	reply, chat, err := h.ChatService.GetResponse(r.Context(), userID, uint(chatID), req.Message)
	if err != nil {
		writeError(w, "Error processing chat: "+err.Error(), http.StatusInternalServerError); return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"reply": reply,
		"chat":  chat,
	})
}

// --- ADD THESE HELPER FUNCTIONS BACK ---

// writeJSON is a helper for sending JSON responses.
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// writeError is a helper for sending JSON error responses.
func writeError(w http.ResponseWriter, message string, status int) {
	writeJSON(w, status, map[string]string{"error": message})
}


// Add this function to chat_handler.go
func (h *ChatHandler) DeleteChat(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(middleware.UserIDKey("userID")).(uint)
	vars := mux.Vars(r)
	chatID, err := strconv.ParseUint(vars["id"], 10, 32)
	if err != nil {
		writeError(w, "Invalid chat ID", http.StatusBadRequest)
		return
	}

	if err := h.ChatService.DeleteChat(r.Context(), uint(chatID), userID); err != nil {
		writeError(w, "Could not delete chat", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent) // 204 No Content is a standard success response for DELETE
}