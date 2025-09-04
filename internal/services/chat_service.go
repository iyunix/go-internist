// File: internal/services/chat_service.go

package services

import (
	"context"
	"errors"
	"log"
	"strings"

	"github.com/iyunix/go-internist/internal/domain"
	"github.com/iyunix/go-internist/internal/repository"
)

type ChatService struct {
	chatRepo    repository.ChatRepository
	messageRepo repository.MessageRepository
	aiService   *AIService
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

// AddChatMessage now returns (aiReply string, chat domain.Chat, err error)
func (s *ChatService) AddChatMessage(ctx context.Context, userID, chatID uint, content string) (string, domain.Chat, error) {
	if content == "" {
		return "", domain.Chat{}, errors.New("message content cannot be empty")
	}

	var chat domain.Chat
	var err error

	if chatID == 0 {
		title := strings.TrimSpace(content)
		if len(title) > 50 {
			title = title[:50] + "..."
		}
		newChat := domain.Chat{
			UserID: userID,
			Title:  title,
		}
		createdChat, err := s.chatRepo.Create(ctx, &newChat)
		if err != nil {
			log.Printf("[ChatService] Failed to create new chat for user %d: %v", userID, err)
			return "", domain.Chat{}, errors.New("failed to create new chat")
		}
		chat = *createdChat
		chatID = chat.ID
	} else {
		existingChat, err := s.chatRepo.FindByID(ctx, chatID)
		if err != nil || existingChat.UserID != userID {
			return "", domain.Chat{}, errors.New("chat not found or unauthorized")
		}
		chat = *existingChat
	}

	userMessage := &domain.Message{
		ChatID:  chatID,
		Role:    "user",
		Content: content,
	}
	if _, err := s.messageRepo.Create(ctx, userMessage); err != nil {
		log.Printf("[ChatService] User message create error: %v", err)
		return "", domain.Chat{}, errors.New("failed to store user message")
	}

	// --- UNCOMMENT AND USE THE AI SERVICE ---
	aiReply, err := s.aiService.GetCompletion(ctx, content)
	if err != nil {
		log.Printf("[ChatService] AI service error: %v", err)
		return "", domain.Chat{}, errors.New("failed to get AI completion")
	}

	assistantMessage := &domain.Message{
		ChatID:  chatID,
		Role:    "assistant",
		Content: aiReply,
	}
	if _, err := s.messageRepo.Create(ctx, assistantMessage); err != nil {
		log.Printf("[ChatService] Assistant message create error: %v", err)
		return "", domain.Chat{}, errors.New("failed to store assistant message")
	}

	return aiReply, chat, nil
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
	// Add logging to see how many messages are found
	log.Printf("[ChatService] Found %d messages for chat %d", len(messages), chatID)
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

