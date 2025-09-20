// G:\go_internist\internal\services\chat_service.go
package services

import (
    "context"
    "errors"
    "fmt"
    "strings"
    "time"
    "sync"
    "log"
    "github.com/iyunix/go-internist/internal/config"
    "github.com/iyunix/go-internist/internal/domain"
    "github.com/iyunix/go-internist/internal/repository/chat"
    "github.com/iyunix/go-internist/internal/repository/message"
    chatservice "github.com/iyunix/go-internist/internal/services/chat"
)

// ServiceTimeouts defines timeouts for each external service
type ServiceTimeouts struct {
    Translation time.Duration
    Embedding   time.Duration
    Pinecone    time.Duration
    LLM         time.Duration
    WarmupLLM   time.Duration // Longer timeout for first request (cold start)
}

// DefaultTimeouts returns production-ready timeouts
func DefaultTimeouts() *ServiceTimeouts {
    return &ServiceTimeouts{
        Translation: 30 * time.Second,   // Translation should be fast
        Embedding:   30 * time.Second,  // Embedding generation
        Pinecone:    30 * time.Second,   // Vector search
        LLM:         60 * time.Second,  // LLM generation (streaming)
        WarmupLLM:   90 * time.Second,  // First request can be slow (cold start)
    }
}

// SimpleCircuitBreaker implements basic circuit breaker pattern
type SimpleCircuitBreaker struct {
    mu           sync.RWMutex
    failures     int
    lastFailTime time.Time
    state        string // "closed", "open", "half-open"
    maxFailures  int
    timeout      time.Duration
    name         string
}

func NewSimpleCircuitBreaker(name string, maxFailures int, timeout time.Duration) *SimpleCircuitBreaker {
    return &SimpleCircuitBreaker{
        name:        name,
        maxFailures: maxFailures,
        timeout:     timeout,
        state:       "closed",
    }
}

func (cb *SimpleCircuitBreaker) Call(fn func() error) error {
    cb.mu.Lock()
    defer cb.mu.Unlock()

    // Check if we should transition from open to half-open
    if cb.state == "open" && time.Since(cb.lastFailTime) > cb.timeout {
        cb.state = "half-open"
    }

    // Reject calls if circuit is open
    if cb.state == "open" {
        return errors.New("circuit breaker is open for " + cb.name)
    }

    // Execute the function
    err := fn()
    
    if err != nil {
        cb.failures++
        cb.lastFailTime = time.Now()
        if cb.failures >= cb.maxFailures {
            cb.state = "open"
        }
        return err
    }

    // Success - reset failures
    if cb.state == "half-open" || cb.failures > 0 {
        cb.failures = 0
        cb.state = "closed"
    }
    return nil
}

func (cb *SimpleCircuitBreaker) GetState() string {
    cb.mu.RLock()
    defer cb.mu.RUnlock()
    return cb.state
}

// WarmupTracker keeps track of API warm-up state
type WarmupTracker struct {
    mu             sync.RWMutex
    services       map[string]bool
    firstCallTimes map[string]time.Time
}

func NewWarmupTracker() *WarmupTracker {
    return &WarmupTracker{
        services:       make(map[string]bool),
        firstCallTimes: make(map[string]time.Time),
    }
}

func (w *WarmupTracker) IsWarmedUp(serviceName string) bool {
    w.mu.RLock()
    defer w.mu.RUnlock()
    return w.services[serviceName]
}

func (w *WarmupTracker) MarkWarmedUp(serviceName string) {
    w.mu.Lock()
    defer w.mu.Unlock()
    w.services[serviceName] = true
}

func (w *WarmupTracker) SetFirstCallTime(serviceName string, t time.Time) {
    w.mu.Lock()
    defer w.mu.Unlock()
    if _, exists := w.firstCallTimes[serviceName]; !exists {
        w.firstCallTimes[serviceName] = t
    }
}

type ChatService struct {
    config             *chatservice.Config
    chatRepo           chat.ChatRepository
    messageRepo        message.MessageRepository
    streamService      *chatservice.StreamingService
    translationService *TranslationService
    logger             Logger
    
    // Performance & Resilience
    timeouts           *ServiceTimeouts
    circuitBreakers    map[string]*SimpleCircuitBreaker
    warmupTracker      *WarmupTracker
}

func NewChatService(
    chatRepo chat.ChatRepository,
    messageRepo message.MessageRepository,
    aiService *AIService,
    pineconeService *PineconeService,
    retrievalTopK int, // This value (15) is injected by Wire
    appConfig *config.Config,
    translationService *TranslationService, // <--- Only injected!
) (*ChatService, error) {
    if chatRepo == nil || messageRepo == nil || aiService == nil || pineconeService == nil {
        return nil, errors.New("all dependencies are required for ChatService")
    }

    log.Printf("[DEBUG] NewChatService created with RetrievalTopK: %d", retrievalTopK)
    config := chatservice.DefaultConfig()
    if retrievalTopK > 0 {
        config.RetrievalTopK = retrievalTopK
    }
    if err := config.Validate(); err != nil {
        return nil, err
    }

    logger := NewLogger("chat_service")

    // Initialize performance & resilience components
    timeouts := DefaultTimeouts()
    warmupTracker := NewWarmupTracker()
    
    // Create circuit breakers for each service
    circuitBreakers := map[string]*SimpleCircuitBreaker{
        "translation": NewSimpleCircuitBreaker("translation", 3, 30*time.Second),
        "embedding":   NewSimpleCircuitBreaker("embedding", 3, 30*time.Second),
        "pinecone":    NewSimpleCircuitBreaker("pinecone", 3, 30*time.Second),
        "llm":         NewSimpleCircuitBreaker("llm", 3, 30*time.Second),
    }

    // Initialize other services with standard constructors
    ragService := chatservice.NewRAGService(config, logger)
    sourceExtractor := chatservice.NewSourceExtractor(config, logger)
    streamService := chatservice.NewStreamingService(
        config, chatRepo, messageRepo, aiService, pineconeService,
        ragService, sourceExtractor, logger,
    )

    return &ChatService{
        config:             config,
        chatRepo:           chatRepo,
        messageRepo:        messageRepo,
        streamService:      streamService,
        translationService: translationService, // <--- Only set once!
        logger:             logger,
        timeouts:           timeouts,
        circuitBreakers:    circuitBreakers,
        warmupTracker:      warmupTracker,
    }, nil
}

// ENHANCED: LLM context with current query prioritization and attention guidance
func (s *ChatService) buildEnhancedLLMContext(pairs []domain.Message, currentQuery string) string {
    var contextParts []string
    
    // 1. PRIORITY: Current question with clear emphasis
    currentEmphasis := fmt.Sprintf("🎯 CURRENT MEDICAL QUESTION: %s", currentQuery)
    contextParts = append(contextParts, currentEmphasis)
    
    // 2. Add conversation context if exists (compressed)
    if len(pairs) > 0 {
        contextParts = append(contextParts, "\n💬 CONVERSATION CONTEXT:")
        
        // Compress previous exchanges to key medical facts only
        for i, m := range pairs {
            if m.MessageType == "internal_context" {
                // Compress user messages to key medical terms
                compressed := s.compressMedicalQuery(m.Content)
                contextParts = append(contextParts, fmt.Sprintf("Previous Q%d: %s", (i/2)+1, compressed))
            } else if m.MessageType == "assistant" {
                // Compress AI responses to key medical facts (max 100 chars)
                compressed := s.compressAssistantResponse(m.Content)
                contextParts = append(contextParts, fmt.Sprintf("Previous A%d: %s", (i/2)+1, compressed))
            }
        }
    }
    
    // 3. INSTRUCTION: Clear task definition
    instruction := fmt.Sprintf("\n⚡ FOCUS INSTRUCTION: Answer the CURRENT question above. Previous context is for reference only. Topic: %s", 
        s.detectMedicalIntent(currentQuery))
    contextParts = append(contextParts, instruction)
    
    return strings.Join(contextParts, "\n")
}

// Helper: Compress medical queries to key terms
func (s *ChatService) compressMedicalQuery(query string) string {
    if len(query) <= 80 {
        return query
    }
    // Extract key medical terms (simplified approach)
    return query[:80] + "..."
}

// Helper: Compress assistant responses to key medical facts
func (s *ChatService) compressAssistantResponse(response string) string {
    if len(response) <= 100 {
        return response
    }
    
    // Look for key medical terms and extract first sentence
    sentences := strings.Split(response, ".")
    if len(sentences) > 0 && len(sentences[0]) > 0 {
        firstSentence := strings.TrimSpace(sentences[0])
        if len(firstSentence) <= 100 {
            return firstSentence + "."
        }
        return firstSentence[:97] + "..."
    }
    
    return response[:97] + "..."
}

// Helper: Detect medical intent for instruction clarity
func (s *ChatService) detectMedicalIntent(query string) string {
    queryLower := strings.ToLower(query)
    
    if strings.Contains(queryLower, "treatment") || strings.Contains(queryLower, "medication") || 
       strings.Contains(queryLower, "drug") || strings.Contains(queryLower, "therapy") {
        return "TREATMENT/MEDICATIONS"
    }
    if strings.Contains(queryLower, "complication") || strings.Contains(queryLower, "dangerous") || 
       strings.Contains(queryLower, "risk") {
        return "COMPLICATIONS/RISKS"  
    }
    if strings.Contains(queryLower, "diagnosis") || strings.Contains(queryLower, "symptom") || 
       strings.Contains(queryLower, "sign") {
        return "DIAGNOSIS/SYMPTOMS"
    }
    
    return "GENERAL MEDICAL QUERY"
}

func (s *ChatService) StreamChatMessageWithSources(
    ctx context.Context,
    userID, chatID uint,
    prompt string,
    onDelta func(string) error,
    onSources func([]string),
    onStatus func(status string, message string),
) error {
    startTime := time.Now()
    s.logger.Info("starting stream chat",
        "user_id", userID, "chat_id", chatID, "prompt_length", len(prompt))

    originalPrompt := prompt
    
    // ----- STEP 1: Get conversation context BEFORE processing -----
    const memoryPairLimit = 3
    
    onStatus("retrieving_context", "Analyzing conversation history...")
    
    pairs, err := s.messageRepo.FindRecentUserAssistantPairs(ctx, chatID, memoryPairLimit, "internal_context")
    if err != nil {
        s.logger.Warn("Failed to retrieve conversation history", "error", err)
        pairs = []domain.Message{} // Continue with empty context
    }

    var embeddingQuery, llmQuery string

    // ----- STEP 2: ENHANCED processing for ALL queries (Persian + English) -----
    if s.translationService != nil {
        onStatus("processing", "Processing your question...")
        
        translationCB := s.circuitBreakers["translation"]
        if translationCB.GetState() == "open" {
            s.logger.Warn("Translation circuit breaker is open; using fallback")
            embeddingQuery = prompt
            llmQuery = prompt
        } else {
            err := translationCB.Call(func() error {
                timeoutCtx, cancel := context.WithTimeout(ctx, s.timeouts.Translation)
                defer cancel()
                
                // Use enhanced processing for ALL queries
                embQ, llmQ, err := s.translationService.TranslateWithMedicalContext(
                    timeoutCtx, 
                    prompt, 
                    pairs, // Pass conversation history
                )
                if err != nil {
                    return err
                }
                embeddingQuery = embQ
                llmQuery = llmQ
                return nil
            })
            
            if err != nil {
                s.logger.Warn("Enhanced processing failed, using original query", "error", err)
                // Fallback: use original query
                embeddingQuery = prompt
                llmQuery = prompt
            }
        }
        
        onStatus("processed", "Processing your question...")
    } else {
        // No translation service available
        embeddingQuery = prompt
        llmQuery = prompt
    }

    // ----- STEP 3: Save messages -----
    if originalPrompt != llmQuery {
        // Save original user message for display
        _, err := s.SaveMessage(ctx, userID, chatID, originalPrompt, "user")
        if err != nil {
            s.logger.Error("failed to save original user message", "error", err)
            return err
        }

        // Save processed version for internal context
        _, err = s.SaveMessage(ctx, userID, chatID, llmQuery, "internal_context")
        if err != nil {
            s.logger.Error("failed to save internal context message", "error", err)
        }
    } else {
        // No processing - save for both display and internal use
        _, err := s.SaveMessage(ctx, userID, chatID, originalPrompt, "user")
        if err != nil {
            s.logger.Error("failed to save user message", "error", err)
            return err
        }
        _, err = s.SaveMessage(ctx, userID, chatID, originalPrompt, "internal_context")
        if err != nil {
            s.logger.Error("failed to save internal context message", "error", err)
        }
    }

    // ----- STEP 4: Build SEPARATE embedding and LLM windows -----
    // Refresh pairs to include the new message
    pairs, err = s.messageRepo.FindRecentUserAssistantPairs(ctx, chatID, memoryPairLimit, "internal_context")
    if err != nil {
        pairs = []domain.Message{}
    }

    // Use enhanced LLM context instead of simple concatenation
    llmText := s.buildEnhancedLLMContext(pairs, llmQuery)

    // For EMBEDDING: Use the FOCUSED query only (KEY ENHANCEMENT)
    embeddingText := embeddingQuery

    s.logger.Info("Prepared queries for RAG",
        "embedding_query", embeddingText,
        "embedding_length", len(embeddingText),
        "llm_context_length", len(llmText),
        "pairs_count", len(pairs))

    // ----- STEP 5: Stream with SEPARATE embedding and LLM text -----
    onStatus("searching", "Finding relevant medical information...")
    
    err = s.streamService.StreamChatResponse(
        ctx,
        userID,
        chatID,
        embeddingText, // FOCUSED query for vector search
        llmText,       // Enhanced context for LLM prompt
        onDelta,
        onSources,
        onStatus,
    )

    if err == nil {
        s.warmupTracker.MarkWarmedUp("llm")
        s.warmupTracker.MarkWarmedUp("embedding")
        s.warmupTracker.MarkWarmedUp("pinecone")
    }

    s.logger.Info("stream chat completed",
        "user_id", userID,
        "total_time", time.Since(startTime),
        "error", err)

    return err
}

// CreateChat creates a new chat with validation and timeout
func (s *ChatService) CreateChat(ctx context.Context, userID uint, title string) (*domain.Chat, error) {
    if strings.TrimSpace(title) == "" {
        return nil, errors.New("chat title cannot be empty")
    }
    if len(title) > 100 {
        title = title[:100]
    }
    
    // Add timeout for database operations
    dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()
    
    newChat := &domain.Chat{UserID: userID, Title: title}
    return s.chatRepo.Create(dbCtx, newChat)
}

// GetChatMessages retrieves messages with timeout protection
func (s *ChatService) GetChatMessagesWithPagination(ctx context.Context, userID, chatID uint, limit, offset int) ([]domain.Message, int64, error) {
    dbCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
    defer cancel()

    chatRecord, err := s.chatRepo.FindByID(dbCtx, chatID)
    if err != nil || chatRecord.UserID != userID {
        return nil, 0, errors.New("unauthorized or chat not found")
    }
    // Always fetch paginated
    return s.messageRepo.FindByChatIDWithPagination(dbCtx, chatID, limit, offset)
}

// DeleteChat deletes a chat with proper cleanup and timeout
func (s *ChatService) DeleteChat(ctx context.Context, userID, chatID uint) error {
    // Add timeout for database operations
    dbCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
    defer cancel()
    
    chatRecord, err := s.chatRepo.FindByID(dbCtx, chatID)
    if err != nil || chatRecord.UserID != userID {
        return errors.New("unauthorized or chat not found")
    }
    
    // Delete messages first
    if err := s.messageRepo.DeleteByChatID(dbCtx, chatID); err != nil {
        s.logger.Error("failed to delete messages for chat", 
            "error", err, "chat_id", chatID, "user_id", userID)
        return err
    }
    
    // Then delete the chat
    return s.chatRepo.Delete(dbCtx, chatID, userID)
}

// GetUserChats retrieves user chats with default pagination
func (s *ChatService) GetUserChats(ctx context.Context, userID uint) ([]domain.Chat, error) {
    chats, _, err := s.GetUserChatsWithPagination(ctx, userID, 100, 0)
    return chats, err
}

// GetUserChatsWithPagination retrieves user chats with pagination and timeout
func (s *ChatService) GetUserChatsWithPagination(ctx context.Context, userID uint, limit, offset int) ([]domain.Chat, int64, error) {
    // Add timeout for database operations
    dbCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
    defer cancel()
    
    return s.chatRepo.FindByUserIDWithPagination(dbCtx, userID, limit, offset)
}

// SaveMessage saves a message with validation and timeout
func (s *ChatService) SaveMessage(ctx context.Context, userID, chatID uint, content, messageType string) (*domain.Message, error) {
    // Add timeout for database operations
    dbCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
    defer cancel()
    
    chatRecord, err := s.chatRepo.FindByID(dbCtx, chatID)
    if err != nil || chatRecord.UserID != userID {
        return nil, errors.New("unauthorized or chat not found")
    }
    
    message := &domain.Message{
        ChatID:      chatID,
        Content:     content,
        MessageType: messageType,
    }
    return s.messageRepo.Create(dbCtx, message)
}

// GetPerformanceMetrics returns current performance and health metrics
func (s *ChatService) GetPerformanceMetrics() map[string]interface{} {
    metrics := make(map[string]interface{})
    
    // Circuit breaker states
    cbStates := make(map[string]string)
    for name, cb := range s.circuitBreakers {
        cbStates[name] = cb.GetState()
    }
    metrics["circuit_breaker_states"] = cbStates
    
    // Warmup states
    warmupStates := make(map[string]bool)
    for _, service := range []string{"llm", "embedding", "pinecone", "translation"} {
        warmupStates[service] = s.warmupTracker.IsWarmedUp(service)
    }
    metrics["warmup_states"] = warmupStates
    
    // Configured timeouts
    metrics["timeouts"] = map[string]string{
        "translation": s.timeouts.Translation.String(),
        "embedding":   s.timeouts.Embedding.String(),
        "pinecone":    s.timeouts.Pinecone.String(),
        "llm":         s.timeouts.LLM.String(),
        "warmup_llm":  s.timeouts.WarmupLLM.String(),
    }
    
    return metrics
}
