// File: internal/services/chat/streaming.go
package chat

import (
    "context"
    "strings"
    "time"

    "github.com/iyunix/go-internist/internal/domain"
    "github.com/iyunix/go-internist/internal/repository/chat"
    "github.com/iyunix/go-internist/internal/repository/message"
    "github.com/qdrant/go-client/qdrant"
)

// NEW: Define clear, local timeouts for each network operation.
const (
    embeddingAPITimeout = 30 * time.Second
    pineconeAPITimeout  = 30 * time.Second // Keep name for compatibility
    llmStreamTimeout    = 60 * time.Second // Streaming can take longer
    dbSaveTimeout       = 5 * time.Second  // Timeout for background saves
)

// StreamingService orchestrates the RAG pipeline for a chat.
type StreamingService struct {
    config          *Config
    chatRepo        chat.ChatRepository
    messageRepo     message.MessageRepository
    aiService       AIProvider
    pineconeService PineconeProvider // Keep name for compatibility
    ragService      *RAGService
    sourceExtractor *SourceExtractor
    logger          Logger
}

// AIProvider defines the interface for AI model interactions (embedding and completion).
type AIProvider interface {
    CreateEmbedding(ctx context.Context, text string) ([]float32, error)
    StreamCompletion(ctx context.Context, model, prompt string, onDelta func(string) error) error
}

// PineconeProvider defines the interface for vector database queries (now using Qdrant types).
type PineconeProvider interface {
    QuerySimilar(ctx context.Context, embedding []float32, topK int) ([]*qdrant.ScoredPoint, error)
}

// NewStreamingService creates a new instance of the StreamingService.
func NewStreamingService(
    config *Config,
    chatRepo chat.ChatRepository,
    messageRepo message.MessageRepository,
    aiService AIProvider,
    pineconeService PineconeProvider,
    ragService *RAGService,
    sourceExtractor *SourceExtractor,
    logger Logger,
) *StreamingService {
    return &StreamingService{
        config:          config,
        chatRepo:        chatRepo,
        messageRepo:     messageRepo,
        aiService:       aiService,
        pineconeService: pineconeService,
        ragService:      ragService,
        sourceExtractor: sourceExtractor,
        logger:          logger,
    }
}

// StreamChatResponse orchestrates the full RAG pipeline with timeouts for each external call.
// StreamChatResponse orchestrates the full RAG pipeline with separate embedding and LLM contexts.
func (s *StreamingService) StreamChatResponse(
    ctx context.Context,
    userID, chatID uint,
    embeddingText string, // ONLY previous user questions + current
    llmText string,       // previous user questions + last assistant + current
    onDelta func(string) error,
    onSources func([]string),
    onStatus func(status, message string),
) error {
    s.logger.Info("starting stream chat", "user_id", userID, "chat_id", chatID)
    onStatus("understanding", "Understanding question...")

    // Validate chat ownership
    chat, err := s.chatRepo.FindByID(ctx, chatID)
    if err != nil || chat.UserID != userID {
        return NewUnauthorizedError(userID, chatID)
    }

    // --- 1️⃣ Harden Embedding Call with a Timeout ---
    embeddingCtx, embeddingCancel := context.WithTimeout(ctx, embeddingAPITimeout)
    defer embeddingCancel()
    embedding, err := s.aiService.CreateEmbedding(embeddingCtx, embeddingText) // ✅ use embeddingText
    if err != nil {
        s.logger.Error("embedding call failed", "error", err)
        return NewRAGError("embedding", "failed to create embedding", err)
    }

    onStatus("searching", "Searching UpToDate knowledge base...")

    // --- 2️⃣ Harden Qdrant Call with a Timeout ---
    pineconeCtx, pineconeCancel := context.WithTimeout(ctx, pineconeAPITimeout)
    defer pineconeCancel()
    matches, err := s.pineconeService.QuerySimilar(pineconeCtx, embedding, s.config.RetrievalTopK)
    if err != nil {
        s.logger.Error("qdrant call failed", "error", err)
        return NewRAGError("qdrant_query", "failed to query Qdrant", err)
    }

    // Send sources if configured
    if s.config.EnableSources && onSources != nil {
        sources := s.sourceExtractor.ExtractSources(matches)
        if len(sources) > 0 {
            onSources(sources)
        }
    }

    // Build final LLM prompt using llmText
    contextJSON, entries := s.ragService.BuildContext(matches)
    finalPrompt := s.ragService.BuildPrompt(contextJSON, llmText, entries) // ✅ use llmText
    onStatus("thinking", "AI is generating a response...")

    // --- 3️⃣ Harden LLM Stream Call with a Timeout ---
    var fullReply strings.Builder
    llmCtx, llmCancel := context.WithTimeout(ctx, llmStreamTimeout)
    defer llmCancel()
    streamErr := s.aiService.StreamCompletion(llmCtx, s.config.StreamModel, finalPrompt, func(token string) error {
        fullReply.WriteString(token)
        return onDelta(token)
    })

    if streamErr != nil {
        s.logger.Error("stream completion failed", "error", streamErr)
        return NewRAGError("streaming", "AI streaming failed", streamErr)
    }

    // Save assistant message in background
    go s.saveAssistantMessage(chatID, fullReply.String())

    s.logger.Info("stream chat completed", "response_length", fullReply.Len())
    return nil
}

// saveUserMessage saves the user's message to the database.
func (s *StreamingService) saveUserMessage(ctx context.Context, chatID uint, content string) error {
    userMessage := &domain.Message{
        ChatID:      chatID,
        MessageType: domain.MessageTypeUser,
        Content:     content,
    }
    _, err := s.messageRepo.Create(ctx, userMessage)
    if err != nil {
        return err
    }
    _ = s.chatRepo.TouchUpdatedAt(ctx, chatID)
    return nil
}

// saveAssistantMessage saves the AI's response to the database in the background.
func (s *StreamingService) saveAssistantMessage(chatID uint, content string) {
    if len(content) > 0 {
        // FIXED: Create a new context with a timeout for this background task.
        ctx, cancel := context.WithTimeout(context.Background(), dbSaveTimeout)
        defer cancel()

        aiMessage := &domain.Message{
            ChatID:      chatID,
            MessageType: domain.MessageTypeAssistant,
            Content:     content,
        }
        if _, err := s.messageRepo.Create(ctx, aiMessage); err != nil {
            s.logger.Error("failed to save assistant message", "error", err)
        }
        // FIXED: Pass the correct argument (chatID) to the function.
        _ = s.chatRepo.TouchUpdatedAt(ctx, chatID)
    }
}
