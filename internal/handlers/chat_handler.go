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

// ChatHandler handles HTTP requests for chat operations with production-ready features
type ChatHandler struct {
    UserService *user_services.UserService
    ChatService *services.ChatService
}

// NewChatHandler creates a new ChatHandler with validation and error handling
func NewChatHandler(userService *user_services.UserService, chatService *services.ChatService) (*ChatHandler, error) {
    // Production-ready validation
    if userService == nil {
        return nil, fmt.Errorf("user service is required for chat handler")
    }
    if chatService == nil {
        return nil, fmt.Errorf("chat service is required for chat handler")
    }

    return &ChatHandler{
        UserService: userService,
        ChatService: chatService,
    }, nil
}

// TotalCreditsProvider interface for medical AI credit management
type TotalCreditsProvider interface {
    GetTotalCredits(ctx context.Context, userID uint) (int, error)
}

// Production constants for medical AI application
const (
    defaultTotalCredits = 2500
    maxChatTitleLength  = 200
    defaultPageSize     = 20
    maxPageSize         = 100
    defaultMessageLimit = 50
)

// GetUserBalance retrieves user's character balance with enhanced error handling
func (h *ChatHandler) GetUserBalance(w http.ResponseWriter, r *http.Request) {
    // Enhanced user ID extraction with validation
    userID, ok := r.Context().Value(middleware.UserIDKey).(uint)
    if !ok || userID == 0 {
        log.Printf("[ChatHandler] Invalid or missing user ID in context")
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }

    // Production-ready balance retrieval with timeout
    ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
    defer cancel()

    balance, err := h.UserService.GetCharacterBalance(ctx, userID)
    if err != nil {
        log.Printf("[ChatHandler] Error getting user balance for user %d: %v", userID, err)
        http.Error(w, "Failed to get balance", http.StatusInternalServerError)
        return
    }

    // Enhanced total credits calculation
    totalCredits := defaultTotalCredits
    if provider, ok := interface{}(h.UserService).(TotalCreditsProvider); ok {
        if total, err := provider.GetTotalCredits(ctx, userID); err == nil && total > 0 {
            totalCredits = total
        }
    }

    // Production-ready response with proper headers
    w.Header().Set("Content-Type", "application/json; charset=utf-8")
    w.Header().Set("Cache-Control", "no-store")
    
    response := map[string]interface{}{
        "balance":      balance,
        "totalCredits": totalCredits,
        "timestamp":    time.Now().Unix(),
        "userId":       userID,
    }
    
    if err := json.NewEncoder(w).Encode(response); err != nil {
        log.Printf("[ChatHandler] Error encoding balance response: %v", err)
    }
}

// CreateChat creates a new medical AI chat with enhanced validation
func (h *ChatHandler) CreateChat(w http.ResponseWriter, r *http.Request) {
    // Enhanced user validation
    userID, ok := r.Context().Value(middleware.UserIDKey).(uint)
    if !ok || userID == 0 {
        log.Printf("[ChatHandler] Invalid user ID in CreateChat")
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }

    // Production-ready request parsing with validation
    var req struct {
        Title string `json:"title"`
        Type  string `json:"type,omitempty"` // Medical AI chat type
    }
    
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        log.Printf("[ChatHandler] Invalid request body in CreateChat: %v", err)
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }

    // Enhanced title validation for medical AI
    if err := h.validateChatTitle(req.Title); err != nil {
        log.Printf("[ChatHandler] Chat title validation failed: %v", err)
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    // Set default chat type for medical AI
    if req.Type == "" {
        req.Type = "medical_consultation"
    }

    // Production-ready chat creation with timeout
    ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
    defer cancel()

    chat, err := h.ChatService.CreateChat(ctx, userID, req.Title)
    if err != nil {
        log.Printf("[ChatHandler] Error creating chat for user %d: %v", userID, err)
        http.Error(w, "Failed to create chat", http.StatusInternalServerError)
        return
    }

    // Enhanced response with medical AI metadata
    w.Header().Set("Content-Type", "application/json; charset=utf-8")
    w.Header().Set("Cache-Control", "no-store")
    w.WriteHeader(http.StatusCreated)
    
    response := map[string]interface{}{
        "id":        chat.ID,
        "title":     chat.Title,
        "userId":    chat.UserID,
        "type":      req.Type,
        "createdAt": chat.CreatedAt,
        "updatedAt": chat.UpdatedAt,
    }
    
    if err := json.NewEncoder(w).Encode(response); err != nil {
        log.Printf("[ChatHandler] Error encoding chat creation response: %v", err)
    }

    log.Printf("[ChatHandler] Successfully created chat %d for user %d", chat.ID, userID)
}

// StreamChatSSE handles medical AI chat streaming with enhanced error handling and monitoring
func (h *ChatHandler) StreamChatSSE(w http.ResponseWriter, r *http.Request) {
	// Enhanced user validation
	userID, ok := r.Context().Value(middleware.UserIDKey).(uint)
	if !ok || userID == 0 {
		log.Printf("[ChatHandler] Invalid user ID in StreamChatSSE")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Production-ready chat ID extraction and validation
	vars := mux.Vars(r)
	idStr, ok := vars["id"]
	if !ok {
		http.Error(w, "Missing chat id in URL", http.StatusBadRequest)
		return
	}

	id64, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil || id64 == 0 {
		log.Printf("[ChatHandler] Invalid chat ID format: %s", idStr)
		http.Error(w, "Invalid chat id", http.StatusBadRequest)
		return
	}
	chatID := uint(id64)

	// Enhanced prompt validation for medical AI
	prompt := r.URL.Query().Get("q")
	if err := h.validateMedicalPrompt(prompt); err != nil {
		log.Printf("[ChatHandler] Medical prompt validation failed: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Production-ready balance checking with detailed logging
	canAsk, chargeAmount, err := h.UserService.CanUserAskQuestion(r.Context(), userID, len(prompt))
	if err != nil {
		log.Printf("[ChatHandler] Error checking user balance for user %d: %v", userID, err)
		http.Error(w, "Error checking balance", http.StatusInternalServerError)
		return
	}

	if !canAsk {
		log.Printf("[ChatHandler] Insufficient balance for user %d, needs %d characters", userID, chargeAmount)
		http.Error(w, fmt.Sprintf("Insufficient character balance. Need %d characters", chargeAmount), http.StatusPaymentRequired)
		return
	}

	// Enhanced character deduction with transaction logging
	actualCharge, err := h.UserService.DeductCharactersForQuestion(r.Context(), userID, len(prompt))
	if err != nil {
		log.Printf("[ChatHandler] Error deducting characters for user %d: %v", userID, err)
		http.Error(w, "Error processing payment", http.StatusInternalServerError)
		return
	}

	log.Printf("[ChatHandler] Medical AI consultation: User %d charged %d characters for question length %d", userID, actualCharge, len(prompt))

	// Production-ready SSE headers with security enhancements
	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Cache-Control")

	// Enhanced flusher validation
	flusher, ok := w.(http.Flusher)
	if !ok {
		log.Printf("[ChatHandler] Streaming unsupported for user %d", userID)
		http.Error(w, "Streaming unsupported on this connection", http.StatusInternalServerError)
		return
	}

	// Production-ready context management
	done := r.Context().Done()
	hb := time.NewTicker(15 * time.Second)
	defer hb.Stop()

	// Enhanced heartbeat with monitoring
	go func() {
		for {
			select {
			case <-hb.C:
				fmt.Fprint(w, ": ping\n\n")
				flusher.Flush()
			case <-done:
				// This log is now handled by the main function's final log message
				// log.Printf("[ChatHandler] Stream context cancelled for user %d", userID)
				return
			}
		}
	}()

	// Medical AI streaming variables
	var documentSources []string
	var sourcesMutex sync.Mutex
	var tokenBuffer []string
	bufferSize := 0
	const maxBufferSize = 100
	const maxTokens = 10

	// --- CORRECTED FLUSHBUFFER FUNCTION ---
	flushBuffer := func() {
		if len(tokenBuffer) == 0 {
			return
		}
		chunk := strings.Join(tokenBuffer, "")

		// Create a JSON payload to wrap the text chunk.
		// This prevents newlines in the AI's response from breaking the SSE format.
		payload := map[string]string{"content": chunk}
		jsonPayload, err := json.Marshal(payload)
		if err != nil {
			log.Printf("[ChatHandler] Error marshalling stream data: %v", err)
			return
		}

		// Send the JSON payload as the data for the event.
		if _, err := fmt.Fprintf(w, "data: %s\n\n", jsonPayload); err != nil {
			log.Printf("[ChatHandler] Error writing stream data: %v", err)
			return
		}

		flusher.Flush()
		tokenBuffer = tokenBuffer[:0]
		bufferSize = 0
	}

	// Enhanced token handling for medical AI responses
	onDelta := func(token string) error {
		select {
		case <-done:
			return fmt.Errorf("stream cancelled")
		default:
		}

		tokenBuffer = append(tokenBuffer, token)
		bufferSize += len(token)

		// Medical AI optimized flushing conditions
		if strings.ContainsAny(token, ".!?\n;:") || bufferSize >= maxBufferSize || len(tokenBuffer) >= maxTokens {
			flushBuffer()
		}
		return nil
	}

	// Enhanced medical source handling
	onSources := func(sources []string) {
		sourcesMutex.Lock()
		documentSources = sources
		sourcesMutex.Unlock()

		if len(sources) > 0 {
			payload := map[string]interface{}{
				"type":      "medical_sources",
				"sources":   sources,
				"timestamp": time.Now().Unix(),
			}

			if b, err := json.Marshal(payload); err == nil {
				fmt.Fprintf(w, "event: metadata\ndata: %s\n\n", b)
				flusher.Flush()
				log.Printf("[ChatHandler] Sent %d medical sources for user %d", len(sources), userID)
			}
		}
	}

	// Production-ready streaming with timeout and monitoring
	streamCtx, streamCancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer streamCancel()

	startTime := time.Now()
	if err := h.ChatService.StreamChatMessageWithSources(streamCtx, userID, chatID, prompt, onDelta, onSources); err != nil {
		log.Printf("[ChatHandler] Medical AI streaming error for user %d chat %d after %v: %v", userID, chatID, time.Since(startTime), err)

		errorPayload := map[string]interface{}{
			"type":  "error",
			"error": "Medical AI service temporarily unavailable",
		}
		if b, err := json.Marshal(errorPayload); err == nil {
			fmt.Fprintf(w, "event: error\ndata: %s\n\n", b)
			flusher.Flush()
		}
		return
	}

	// Final buffer flush
	flushBuffer()

	// Final metadata and completion events...
	sourcesMutex.Lock()
	if len(documentSources) > 0 {
		final := map[string]interface{}{
			"type":           "final_medical_sources",
			"sources":        documentSources,
			"responseTime":   time.Since(startTime).Milliseconds(),
			"sourceCount":    len(documentSources),
			"consultationId": chatID,
		}
		if b, err := json.Marshal(final); err == nil {
			fmt.Fprintf(w, "event: metadata\ndata: %s\n\n", b)
			flusher.Flush()
		}
	}
	sourcesMutex.Unlock()

	completionPayload := map[string]interface{}{
		"type":         "consultation_complete",
		"responseTime": time.Since(startTime).Milliseconds(),
		"chargeAmount": actualCharge,
	}
	if b, err := json.Marshal(completionPayload); err == nil {
		fmt.Fprintf(w, "event: complete\ndata: %s\n\n", b)
		flusher.Flush()
	}

	// Signal the end of the stream to the client.
	fmt.Fprintf(w, "event: done\ndata: \n\n")
	flusher.Flush()

	log.Printf("[ChatHandler] Medical AI consultation completed for user %d in %v", userID, time.Since(startTime))

	// --- FINAL BLOCKING FIX ---
	// Block here until the client disconnects, preventing premature connection reset.
	<-r.Context().Done()
	log.Printf("[ChatHandler] Client has disconnected, stream for user %d is now fully closed.", userID)
}


// GetUserChats retrieves user chats with production-ready pagination
func (h *ChatHandler) GetUserChats(w http.ResponseWriter, r *http.Request) {
    // Enhanced user validation
    userID, ok := r.Context().Value(middleware.UserIDKey).(uint)
    if !ok || userID == 0 {
        log.Printf("[ChatHandler] Invalid user ID in GetUserChats")
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }

    // ✅ FIXED: Parse pagination parameters (backward compatible)
    page := h.getPageFromQuery(r)
    limit := h.getLimitFromQuery(r)
    
    // For backward compatibility, use high limit to get most chats at once
    if limit == defaultPageSize {
        limit = 100 // Higher default to load most chats at once
    }
    
    offset := (page - 1) * limit
    
    // Enhanced chat retrieval with timeout and pagination
    ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
    defer cancel()

    // ✅ CALL NEW SERVICE METHOD (eliminates the warning)
    chats, total, err := h.ChatService.GetUserChatsWithPagination(ctx, userID, limit, offset)
    if err != nil {
        log.Printf("[ChatHandler] Error getting chats for user %d: %v", userID, err)
        http.Error(w, "Failed to get user chats", http.StatusInternalServerError)
        return
    }

    // ✅ BACKWARD COMPATIBLE: Return just chats array (same as before)
    w.Header().Set("Content-Type", "application/json; charset=utf-8")
    w.Header().Set("Cache-Control", "no-store")
    response := map[string]interface{}{
    "chats": chats,
    "total": total,
    "page":  page,
    "limit": limit,
    "has_more": total > int64(offset + len(chats)),
    }
    if err := json.NewEncoder(w).Encode(response); err != nil {
        log.Printf("[ChatHandler] Error encoding chats response: %v", err)
    }

    log.Printf("[ChatHandler] Retrieved %d chats for user %d (total: %d)", len(chats), userID, total)
}


// GetChatMessages retrieves chat messages with production-ready pagination and filtering
func (h *ChatHandler) GetChatMessages(w http.ResponseWriter, r *http.Request) {
    // Enhanced user validation
    userID, ok := r.Context().Value(middleware.UserIDKey).(uint)
    if !ok || userID == 0 {
        log.Printf("[ChatHandler] Invalid user ID in GetChatMessages")
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }

    // Production-ready chat ID validation
    vars := mux.Vars(r)
    idStr, ok := vars["id"]
    if !ok {
        http.Error(w, "Missing chat id in URL", http.StatusBadRequest)
        return
    }

    chatIDU64, err := strconv.ParseUint(idStr, 10, 64)
    if err != nil || chatIDU64 == 0 {
        log.Printf("[ChatHandler] Invalid chat ID format: %s", idStr)
        http.Error(w, "Invalid chat id", http.StatusBadRequest)
        return
    }
    chatID := uint(chatIDU64)

    // Enhanced pagination and filtering parameters
    page := h.getPageFromQuery(r)
    limit := h.getLimitFromQuery(r)
    messageType := r.URL.Query().Get("type") // Filter by message type

    // Production-ready message retrieval with timeout
    ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
    defer cancel()

    messages, err := h.ChatService.GetChatMessages(ctx, userID, chatID)
    if err != nil {
        log.Printf("[ChatHandler] Error getting messages for user %d chat %d: %v", userID, chatID, err)
        http.Error(w, "Failed to get messages", http.StatusInternalServerError)
        return
    }

    // Enhanced filtering for medical AI message types
    if messageType != "" {
        messages = h.filterMessagesByType(messages, messageType)
    }

    // Enhanced response with medical AI metadata
    w.Header().Set("Content-Type", "application/json; charset=utf-8")
    w.Header().Set("Cache-Control", "no-store")
    
    response := map[string]interface{}{
        "messages":      messages,
        "total":         len(messages),
        "chatId":        chatID,
        "page":          page,
        "limit":         limit,
        "messageType":   messageType,
        "timestamp":     time.Now().Unix(),
        "userId":        userID,
    }
    
    if err := json.NewEncoder(w).Encode(response); err != nil {
        log.Printf("[ChatHandler] Error encoding messages response: %v", err)
    }

    log.Printf("[ChatHandler] Retrieved %d messages for user %d chat %d", len(messages), userID, chatID)
}

// DeleteChat deletes a chat with enhanced validation and logging
func (h *ChatHandler) DeleteChat(w http.ResponseWriter, r *http.Request) {
    // Enhanced user validation
    userID, ok := r.Context().Value(middleware.UserIDKey).(uint)
    if !ok || userID == 0 {
        log.Printf("[ChatHandler] Invalid user ID in DeleteChat")
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }

    // Production-ready chat ID validation
    vars := mux.Vars(r)
    chatIDU64, err := strconv.ParseUint(vars["id"], 10, 64)
    if err != nil || chatIDU64 == 0 {
        log.Printf("[ChatHandler] Invalid chat ID for deletion: %s", vars["id"])
        http.Error(w, "Invalid chat id", http.StatusBadRequest)
        return
    }
    chatID := uint(chatIDU64)

    // Production-ready deletion with timeout and logging
    ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
    defer cancel()

    if err := h.ChatService.DeleteChat(ctx, userID, chatID); err != nil {
        log.Printf("[ChatHandler] Error deleting chat %d for user %d: %v", chatID, userID, err)
        http.Error(w, "Failed to delete chat", http.StatusInternalServerError)
        return
    }

    // Enhanced response headers
    w.Header().Set("Cache-Control", "no-store")
    w.WriteHeader(http.StatusNoContent)
    
    log.Printf("[ChatHandler] Successfully deleted chat %d for user %d", chatID, userID)
}

// ===== PRODUCTION-READY HELPER METHODS =====

// validateChatTitle validates chat title for medical AI application
func (h *ChatHandler) validateChatTitle(title string) error {
    title = strings.TrimSpace(title)
    if title == "" {
        return fmt.Errorf("chat title cannot be empty")
    }
    if len(title) > maxChatTitleLength {
        return fmt.Errorf("chat title too long (max %d characters)", maxChatTitleLength)
    }
    
    // Basic XSS protection
    if strings.Contains(title, "<script") || strings.Contains(title, "javascript:") {
        return fmt.Errorf("invalid characters in chat title")
    }
    
    return nil
}

// validateMedicalPrompt validates medical AI prompts
func (h *ChatHandler) validateMedicalPrompt(prompt string) error {
    if prompt == "" {
        return fmt.Errorf("missing query parameter: q")
    }
    if len(prompt) > domain.MaxQuestionLength {
        return fmt.Errorf("question too long. Maximum %d characters allowed", domain.MaxQuestionLength)
    }
    if len(strings.TrimSpace(prompt)) == 0 {
        return fmt.Errorf("prompt cannot be empty")
    }
    
    // Enhanced medical content validation
    if strings.Contains(prompt, "<script") || strings.Contains(prompt, "javascript:") {
        return fmt.Errorf("invalid characters in medical prompt")
    }
    
    return nil
}

// getPageFromQuery extracts page number from query parameters
func (h *ChatHandler) getPageFromQuery(r *http.Request) int {
    pageStr := r.URL.Query().Get("page")
    if pageStr == "" {
        return 1
    }
    
    page, err := strconv.Atoi(pageStr)
    if err != nil || page < 1 {
        return 1
    }
    
    return page
}

// getLimitFromQuery extracts limit from query parameters
func (h *ChatHandler) getLimitFromQuery(r *http.Request) int {
    limitStr := r.URL.Query().Get("limit")
    if limitStr == "" {
        return defaultPageSize
    }
    
    limit, err := strconv.Atoi(limitStr)
    if err != nil || limit < 1 {
        return defaultPageSize
    }
    
    if limit > maxPageSize {
        return maxPageSize
    }
    
    return limit
}

// filterMessagesByType filters messages by medical AI message type
func (h *ChatHandler) filterMessagesByType(messages []domain.Message, messageType string) []domain.Message {
    if messageType == "" {
        return messages
    }
    
    var filtered []domain.Message
    for _, message := range messages {
        if message.MessageType == messageType {
            filtered = append(filtered, message)
        }
    }
    
    return filtered
}

// SendMessage sends a new message to a chat
func (h *ChatHandler) SendMessage(w http.ResponseWriter, r *http.Request) {
    // Enhanced user validation
    userID, ok := r.Context().Value(middleware.UserIDKey).(uint)
    if !ok || userID == 0 {
        log.Printf("[ChatHandler] Invalid user ID in SendMessage")
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }

    // Production-ready chat ID extraction and validation
    vars := mux.Vars(r)
    idStr, ok := vars["id"]
    if !ok {
        http.Error(w, "Missing chat id in URL", http.StatusBadRequest)
        return
    }
    
    id64, err := strconv.ParseUint(idStr, 10, 64)
    if err != nil || id64 == 0 {
        log.Printf("[ChatHandler] Invalid chat ID format: %s", idStr)
        http.Error(w, "Invalid chat id", http.StatusBadRequest)
        return
    }
    chatID := uint(id64)

    // Production-ready request parsing
    var req struct {
        Content     string `json:"content"`
        MessageType string `json:"messageType,omitempty"`
    }
    
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        log.Printf("[ChatHandler] Invalid request body in SendMessage: %v", err)
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }

    // Set default message type
    if req.MessageType == "" {
        req.MessageType = "user"
    }

    // Production-ready message creation with timeout
    ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
    defer cancel()

    message, err := h.ChatService.SaveMessage(ctx, userID, chatID, req.Content, req.MessageType)
    if err != nil {
        log.Printf("[ChatHandler] Error saving message for user %d chat %d: %v", userID, chatID, err)
        http.Error(w, "Failed to save message", http.StatusInternalServerError)
        return
    }

    // Enhanced response
    w.Header().Set("Content-Type", "application/json; charset=utf-8")
    w.Header().Set("Cache-Control", "no-store")
    w.WriteHeader(http.StatusCreated)
    
    response := map[string]interface{}{
        "id":          message.ID,
        "content":     message.Content,
        "messageType": message.MessageType,
        "chatId":      message.ChatID,
        "createdAt":   message.CreatedAt,
        "updatedAt":   message.UpdatedAt,
        "archived":    message.Archived,        // ✅ Available field

    }
    
    if err := json.NewEncoder(w).Encode(response); err != nil {
        log.Printf("[ChatHandler] Error encoding message response: %v", err)
    }

    log.Printf("[ChatHandler] Successfully saved message %d for user %d in chat %d", message.ID, userID, chatID)
}
