package services

import (
    "context"
    "strings"

    "github.com/iyunix/go-internist/internal/domain"
    chatservice "github.com/iyunix/go-internist/internal/services/chat"
    "github.com/iyunix/go-internist/internal/repository/chat"
    "github.com/iyunix/go-internist/internal/repository/message"
    "github.com/pinecone-io/go-pinecone/v4/pinecone"
)

type ChatService struct {
    config          *chatservice.Config
    chatRepo        chat.ChatRepository
    messageRepo     message.MessageRepository
    streamService   *chatservice.StreamingService
    ragService      *chatservice.RAGService
    sourceExtractor *chatservice.SourceExtractor
    logger          Logger
}

func NewChatService(
    chatRepo chat.ChatRepository,
    messageRepo message.MessageRepository,
    aiService *AIService,
    pineconeService *PineconeService,
    retrievalTopK int,
) (*ChatService, error) {
    // Validate dependencies
    if chatRepo == nil {
        return nil, chatservice.NewValidationError("constructor", "chat repository is required")
    }
    if messageRepo == nil {
        return nil, chatservice.NewValidationError("constructor", "message repository is required")
    }
    if aiService == nil {
        return nil, chatservice.NewValidationError("constructor", "AI service is required")
    }
    if pineconeService == nil {
        return nil, chatservice.NewValidationError("constructor", "Pinecone service is required")
    }

    // Create configuration
    config := chatservice.DefaultConfig()
    if retrievalTopK > 0 {
        config.RetrievalTopK = retrievalTopK
    }

    // Validate configuration
    if err := config.Validate(); err != nil {
        return nil, chatservice.NewValidationError("config", err.Error())
    }

    // Create logger
    logger := &NoOpLogger{}

    // Create modular components
    ragService := chatservice.NewRAGService(config, logger)
    sourceExtractor := chatservice.NewSourceExtractor(config, logger)
    streamService := chatservice.NewStreamingService(
        config, chatRepo, messageRepo, aiService, pineconeService,
        ragService, sourceExtractor, logger,
    )

    return &ChatService{
        config:          config,
        chatRepo:        chatRepo,
        messageRepo:     messageRepo,
        streamService:   streamService,
        ragService:      ragService,
        sourceExtractor: sourceExtractor,
        logger:          logger,
    }, nil
}

// Basic chat operations
func (s *ChatService) CreateChat(ctx context.Context, userID uint, title string) (*domain.Chat, error) {
    if strings.TrimSpace(title) == "" {
        return nil, chatservice.NewValidationError("create_chat", "chat title cannot be empty")
    }
    if len(title) > 100 {
        title = title[:100]
    }

    newChat := &domain.Chat{UserID: userID, Title: title}
    createdChat, err := s.chatRepo.Create(ctx, newChat)
    if err != nil {
        return nil, chatservice.NewRAGError("create_chat", "could not create chat", err)
    }
    return createdChat, nil
}

func (s *ChatService) GetUserChats(ctx context.Context, userID uint) ([]domain.Chat, error) {
    return s.chatRepo.FindByUserID(ctx, userID)
}

func (s *ChatService) GetChatMessages(ctx context.Context, userID, chatID uint) ([]domain.Message, error) {
    chatRecord, err := s.chatRepo.FindByID(ctx, chatID)
    if err != nil || chatRecord.UserID != userID {
        return nil, chatservice.NewUnauthorizedError(userID, chatID)
    }
    return s.messageRepo.FindByChatID(ctx, chatID)
}

func (s *ChatService) DeleteChat(ctx context.Context, userID, chatID uint) error {
    chatRecord, err := s.chatRepo.FindByID(ctx, chatID)
    if err != nil || chatRecord.UserID != userID {
        return chatservice.NewUnauthorizedError(userID, chatID)
    }
    return s.chatRepo.Delete(ctx, chatID, userID)
}

// Streaming functionality
func (s *ChatService) StreamChatMessage(
    ctx context.Context,
    userID, chatID uint,
    prompt string,
    onDelta func(string) error,
) error {
    return s.streamService.StreamChatResponse(ctx, userID, chatID, prompt, onDelta, nil)
}

func (s *ChatService) StreamChatMessageWithSources(
    ctx context.Context,
    userID, chatID uint,
    prompt string,
    onDelta func(string) error,
    onSources func([]string),
) error {
    return s.streamService.StreamChatResponse(ctx, userID, chatID, prompt, onDelta, onSources)
}

// Legacy compatibility
func (s *ChatService) AddChatMessage(ctx context.Context, userID, chatID uint, content string) (string, domain.Chat, error) {
    return "This is the non-streaming endpoint.", domain.Chat{}, nil
}

func (s *ChatService) ExtractSourceTitles(matches []*pinecone.ScoredVector) []string {
    return s.sourceExtractor.ExtractSources(matches)
}
