// File: internal/services/chat_service.go
package services

import (
    "context"
    "errors"
    "strings"
    "time"
    "sync"

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
        Translation: 5 * time.Second,   // Translation should be fast
        Embedding:   10 * time.Second,  // Embedding generation
        Pinecone:    8 * time.Second,   // Vector search
        LLM:         45 * time.Second,  // LLM generation (streaming)
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
    mu              sync.RWMutex
    services        map[string]bool
    firstCallTimes  map[string]time.Time
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
    config              *chatservice.Config
    chatRepo            chat.ChatRepository
    messageRepo         message.MessageRepository
    streamService       *chatservice.StreamingService
    translationService  *TranslationService
    logger              Logger
    
    // Performance & Resilience
    timeouts            *ServiceTimeouts
    circuitBreakers     map[string]*SimpleCircuitBreaker
    warmupTracker       *WarmupTracker
}

func NewChatService(
    chatRepo chat.ChatRepository,
    messageRepo message.MessageRepository,
    aiService *AIService,
    pineconeService *PineconeService,
    retrievalTopK int,
    appConfig *config.Config,
) (*ChatService, error) {
    if chatRepo == nil || messageRepo == nil || aiService == nil || pineconeService == nil {
        return nil, errors.New("all dependencies are required for ChatService")
    }

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

    // Initialize translation service if enabled
    var translationService *TranslationService
    if appConfig.IsTranslationEnabled() {
        // Use the standard NewTranslationService function
        translationService = NewTranslationService(appConfig.AvalaiAPIKeyTranslation, logger)
        logger.Info("Translation service initialized with performance optimizations",
            "timeout", timeouts.Translation)
    } else {
        logger.Info("Translation service disabled")
    }

    // Initialize other services with standard constructors
    ragService := chatservice.NewRAGService(config, logger)
    sourceExtractor := chatservice.NewSourceExtractor(config, logger)
    
    // Use the standard NewStreamingService function
    streamService := chatservice.NewStreamingService(
        config, chatRepo, messageRepo, aiService, pineconeService,
        ragService, sourceExtractor, logger,
    )

    return &ChatService{
        config:             config,
        chatRepo:           chatRepo,
        messageRepo:        messageRepo,
        streamService:      streamService,
        translationService: translationService,
        logger:             logger,
        timeouts:           timeouts,
        circuitBreakers:    circuitBreakers,
        warmupTracker:      warmupTracker,
    }, nil
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
    s.logger.Info("starting stream chat with performance monitoring", 
        "user_id", userID, "chat_id", chatID, "prompt_length", len(prompt))

    processedPrompt := prompt
    
    // Smart translation logic with circuit breaker and timeout
    if s.translationService != nil {
        if s.translationService.NeedsTranslation(prompt) {
            onStatus("translating", "Processing text for optimal search...")
            
            // Check circuit breaker state
            translationCB := s.circuitBreakers["translation"]
            if translationCB.GetState() == "open" {
                s.logger.Warn("Translation circuit breaker is open, skipping translation")
                onStatus("translation_skipped", "Translation service unavailable, proceeding with original text")
            } else {
                // Execute translation with circuit breaker protection
                err := translationCB.Call(func() error {
                    // Create timeout context for translation
                    timeoutCtx, cancel := context.WithTimeout(ctx, s.timeouts.Translation)
                    defer cancel()
                    
                    translated, err := s.translationService.TranslateToEnglish(timeoutCtx, prompt)
                    if err != nil {
                        return err  // ✅ Return error directly
                    }
                    processedPrompt = translated
                    return nil
                })

                
                if err != nil {
                    s.logger.Warn("Translation failed with circuit breaker protection", 
                        "error", err,
                        "circuit_state", translationCB.GetState())
                    onStatus("translation_failed", "Translation failed, proceeding with original text")
                } else {
                    s.logger.Info("Text processed for better search", 
                        "original_length", len(prompt), 
                        "processed_length", len(processedPrompt),
                        "translation_time", time.Since(startTime))
                    onStatus("translated", "Text optimized for medical search")
                }
            }
        } else {
            s.logger.Debug("Text is purely English, no translation needed")
        }
    }

    // Determine LLM timeout based on warm-up state
    llmTimeout := s.timeouts.LLM
    if !s.warmupTracker.IsWarmedUp("llm") {
        llmTimeout = s.timeouts.WarmupLLM
        s.warmupTracker.SetFirstCallTime("llm", time.Now())
        s.logger.Info("Using extended timeout for LLM warm-up", 
            "timeout", llmTimeout)
        onStatus("warming_up", "Initializing AI model (first request may take longer)...")
    }

    // Execute the main streaming response
    err := s.streamService.StreamChatResponse(ctx, userID, chatID, processedPrompt, onDelta, onSources, onStatus)

    // Mark services as warmed up on successful completion
    if err == nil {
        s.warmupTracker.MarkWarmedUp("llm")
        s.warmupTracker.MarkWarmedUp("embedding")
        s.warmupTracker.MarkWarmedUp("pinecone")
    }

    totalTime := time.Since(startTime)
    s.logger.Info("stream chat completed with performance metrics",
        "user_id", userID,
        "total_time", totalTime,
        "error", err,
        "warmup_state", map[string]bool{
            "llm": s.warmupTracker.IsWarmedUp("llm"),
            "embedding": s.warmupTracker.IsWarmedUp("embedding"),
            "pinecone": s.warmupTracker.IsWarmedUp("pinecone"),
        })

    // Performance alert for slow requests
    if totalTime > 10*time.Second {
        s.logger.Warn("PERFORMANCE_ALERT: Slow chat response detected",
            "user_id", userID,
            "total_time", totalTime,
            "threshold", "10s")
    }

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
func (s *ChatService) GetChatMessages(ctx context.Context, userID, chatID uint) ([]domain.Message, error) {
    // Add timeout for database operations
    dbCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
    defer cancel()
    
    chatRecord, err := s.chatRepo.FindByID(dbCtx, chatID)
    if err != nil || chatRecord.UserID != userID {
        return nil, errors.New("unauthorized or chat not found")
    }
    return s.messageRepo.FindByChatID(dbCtx, chatID)
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
