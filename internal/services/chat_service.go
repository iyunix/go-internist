// File: internal/services/chat_service.go
package services

import (
    "context"
    "errors"
    "strings"

    "github.com/iyunix/go-internist/internal/config"
    "github.com/iyunix/go-internist/internal/domain"
    "github.com/iyunix/go-internist/internal/repository/chat"
    "github.com/iyunix/go-internist/internal/repository/message"
    chatservice "github.com/iyunix/go-internist/internal/services/chat"
)

type ChatService struct {
    config              *chatservice.Config
    chatRepo            chat.ChatRepository
    messageRepo         message.MessageRepository
    streamService       *chatservice.StreamingService
    translationService  *TranslationService // NEW: Translation service
    logger              Logger
}

func NewChatService(
    chatRepo chat.ChatRepository,
    messageRepo message.MessageRepository,
    aiService *AIService,
    pineconeService *PineconeService,
    retrievalTopK int,
    appConfig *config.Config, // NEW: Add app config parameter
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

    // NEW: Initialize translation service if enabled
    var translationService *TranslationService
    if appConfig.IsTranslationEnabled() {
        translationService = NewTranslationService(appConfig.AvalaiAPIKeyTranslation, logger)
        logger.Info("Translation service initialized for Persian-to-English translation")
    } else {
        logger.Info("Translation service disabled")
    }

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
        translationService: translationService,
        logger:             logger,
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
    processedPrompt := prompt
    
    // Smart translation logic
    if s.translationService != nil {
        if s.translationService.NeedsTranslation(prompt) {
            onStatus("translating", "Processing text for optimal search...")
            
            translated, err := s.translationService.TranslateToEnglish(ctx, prompt)
            if err != nil {
                s.logger.Warn("Translation failed, using original text", "error", err)
                onStatus("translation_failed", "Translation failed, proceeding with original text")
            } else {
                processedPrompt = translated
                s.logger.Info("Text processed for better search", 
                    "original", prompt, 
                    "processed", processedPrompt)
                onStatus("translated", "Text optimized for medical search")
            }
        } else {
            s.logger.Debug("Text is purely English, no translation needed", "text", prompt)
        }
    }

    // Use processed prompt for embedding search and response generation
    return s.streamService.StreamChatResponse(ctx, userID, chatID, processedPrompt, onDelta, onSources, onStatus)
}

func (s *ChatService) CreateChat(ctx context.Context, userID uint, title string) (*domain.Chat, error) {
    if strings.TrimSpace(title) == "" {
        return nil, errors.New("chat title cannot be empty")
    }
    if len(title) > 100 {
        title = title[:100]
    }
    newChat := &domain.Chat{UserID: userID, Title: title}
    return s.chatRepo.Create(ctx, newChat)
}

func (s *ChatService) GetChatMessages(ctx context.Context, userID, chatID uint) ([]domain.Message, error) {
    chatRecord, err := s.chatRepo.FindByID(ctx, chatID)
    if err != nil || chatRecord.UserID != userID {
        return nil, errors.New("unauthorized or chat not found")
    }
    return s.messageRepo.FindByChatID(ctx, chatID)
}

func (s *ChatService) DeleteChat(ctx context.Context, userID, chatID uint) error {
    chatRecord, err := s.chatRepo.FindByID(ctx, chatID)
    if err != nil || chatRecord.UserID != userID {
        return errors.New("unauthorized or chat not found")
    }
    if err := s.messageRepo.DeleteByChatID(ctx, chatID); err != nil {
        s.logger.Error("failed to delete messages for chat", "error", err, "chat_id", chatID)
        return err
    }
    return s.chatRepo.Delete(ctx, chatID, userID)
}

func (s *ChatService) GetUserChats(ctx context.Context, userID uint) ([]domain.Chat, error) {
    chats, _, err := s.GetUserChatsWithPagination(ctx, userID, 100, 0)
    return chats, err
}

func (s *ChatService) GetUserChatsWithPagination(ctx context.Context, userID uint, limit, offset int) ([]domain.Chat, int64, error) {
    return s.chatRepo.FindByUserIDWithPagination(ctx, userID, limit, offset)
}

func (s *ChatService) SaveMessage(ctx context.Context, userID, chatID uint, content, messageType string) (*domain.Message, error) {
    chatRecord, err := s.chatRepo.FindByID(ctx, chatID)
    if err != nil || chatRecord.UserID != userID {
        return nil, errors.New("unauthorized or chat not found")
    }
    message := &domain.Message{
        ChatID:      chatID,
        Content:     content,
        MessageType: messageType,
    }
    return s.messageRepo.Create(ctx, message)
}
