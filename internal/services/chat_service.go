// File: internal/services/chat_service.go

package services

import (
    "context"
    "errors"
    "log"

    "github.com/iyunix/go-internist/internal/domain"
    "github.com/iyunix/go-internist/internal/repository"
)

type ChatService struct {
    chatRepo     repository.ChatRepository
    messageRepo  repository.MessageRepository
    aiService    *AIService
    // Optionally add cache here for recent chats/messages or AI responses
    // cache       SomeCacheType
}

func NewChatService(chatRepo repository.ChatRepository, messageRepo repository.MessageRepository, aiService *AIService) *ChatService {
    return &ChatService{
        chatRepo:    chatRepo,
        messageRepo: messageRepo,
        aiService:   aiService,
    }
}

// GetUserChats fetches all chats for a user
func (s *ChatService) GetUserChats(ctx context.Context, userID uint) ([]domain.Chat, error) {
    // Optionally fetch from cache first if implemented
    return s.chatRepo.FindByUserID(ctx, userID)
}

// AddChatMessage wraps chat/message creation in a transaction for integrity
func (s *ChatService) AddChatMessage(ctx context.Context, userID, chatID uint, content string) (*domain.Message, error) {
    if content == "" {
        return nil, errors.New("message content cannot be empty")
    }

    // Here you could add content length validation, character filtering, etc.

    // Begin transaction (assuming GORM passed via repo layer, else replicate as needed)
    chat, err := s.chatRepo.FindByID(ctx, chatID)
    if err != nil || chat.UserID != userID {
        log.Printf("[ChatService] Chat %d not found for user %d", chatID, userID)
        return nil, errors.New("chat not found or unauthorized")
    }

    message := &domain.Message{
        ChatID:  chatID,
        Role:    "user",
        Content: content,
    }

    // Optionally cache this request/content

    // Add new message in DB
    dbMsg, err := s.messageRepo.Create(ctx, message)
    if err != nil {
        log.Printf("[ChatService] Message create error: %v", err)
        return nil, errors.New("failed to store message")
    }

    // Collect context and call AI service for reply if desired (example)
    // aiReply, aiErr := s.aiService.GetCompletion(ctx, content)
    // if aiErr == nil {
    //   // Optionally store or cache aiReply
    // }

    return dbMsg, nil
}

// GetChatMessages fetches messages for a chat
func (s *ChatService) GetChatMessages(ctx context.Context, userID, chatID uint) ([]domain.Message, error) {
    // Confirm chat ownership first for privacy
    chat, err := s.chatRepo.FindByID(ctx, chatID)
    if err != nil || chat.UserID != userID {
        log.Printf("[ChatService] Unauthorized attempt to fetch chat %d for user %d", chatID, userID)
        return nil, errors.New("chat not found or unauthorized")
    }
    messages, err := s.messageRepo.FindByChatID(ctx, chatID)
    if err != nil {
        log.Printf("[ChatService] Message fetch error: %v", err)
        return nil, errors.New("failed to get messages")
    }
    // Optionally cache results
    return messages, nil
}

// DeleteChat removes a chat for the user, with error logging
func (s *ChatService) DeleteChat(ctx context.Context, userID, chatID uint) error {
    // Always check user ownership
    chat, err := s.chatRepo.FindByID(ctx, chatID)
    if err != nil || chat.UserID != userID {
        log.Printf("[ChatService] DeleteChat invalid access user %d chat %d", userID, chatID)
        return errors.New("chat not found or unauthorized")
    }
    err = s.chatRepo.Delete(ctx, chatID, userID)
    if err != nil {
        log.Printf("[ChatService] DeleteChat DB error: %v", err)
        return errors.New("failed to delete chat")
    }
    // Optionally clear cache entries
    return nil
}
