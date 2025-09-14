// File: internal/services/chat_service.go
package services

import (
	"context"
	"errors"
	"strings"

	"github.com/iyunix/go-internist/internal/domain"
	"github.com/iyunix/go-internist/internal/repository/chat"
	"github.com/iyunix/go-internist/internal/repository/message"
	chatservice "github.com/iyunix/go-internist/internal/services/chat"
)

type ChatService struct {
	config        *chatservice.Config
	chatRepo      chat.ChatRepository
	messageRepo   message.MessageRepository
	streamService *chatservice.StreamingService
	logger        Logger
}

func NewChatService(
	chatRepo chat.ChatRepository,
	messageRepo message.MessageRepository,
	aiService *AIService,
	pineconeService *PineconeService,
	retrievalTopK int,
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
	ragService := chatservice.NewRAGService(config, logger)
	sourceExtractor := chatservice.NewSourceExtractor(config, logger)
	streamService := chatservice.NewStreamingService(
		config, chatRepo, messageRepo, aiService, pineconeService,
		ragService, sourceExtractor, logger,
	)
	return &ChatService{
		config:        config,
		chatRepo:      chatRepo,
		messageRepo:   messageRepo,
		streamService: streamService,
		logger:        logger,
	}, nil
}

// CHANGE 1: Add the new onStatus parameter to the function signature
func (s *ChatService) StreamChatMessageWithSources(
	ctx context.Context,
	userID, chatID uint,
	prompt string,
	onDelta func(string) error,
	onSources func([]string),
	onStatus func(status string, message string), // <-- ADD THIS
) error {
	// CHANGE 2: Pass the new onStatus parameter down to the streaming service
	return s.streamService.StreamChatResponse(ctx, userID, chatID, prompt, onDelta, onSources, onStatus)
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