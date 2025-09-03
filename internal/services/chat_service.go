// File: internal/services/chat_service.go
package services

import (
	"context"
	"errors"
	"strings"

	"github.com/iyunix/go-internist/internal/domain"
	"github.com/iyunix/go-internist/internal/repository"
)

// ChatService now includes the AIService.
type ChatService struct {
	chatRepo    repository.ChatRepository
	messageRepo repository.MessageRepository
	aiService   *AIService // <-- New dependency
}

// NewChatService now requires the AIService.
func NewChatService(chatRepo repository.ChatRepository, messageRepo repository.MessageRepository, aiService *AIService) *ChatService {
	return &ChatService{
		chatRepo:    chatRepo,
		messageRepo: messageRepo,
		aiService:   aiService,
	}
}

// GetResponse is a new, high-level function for the entire chat turn.
func (s *ChatService) GetResponse(ctx context.Context, userID uint, chatID uint, userMessage string) (string, *domain.Chat, error) {
	var currentChat *domain.Chat
	var err error

	// If chatID is 0, it's the first message of a new chat.
	if chatID == 0 {
		currentChat, err = s.CreateNewChat(ctx, userID, userMessage)
		if err != nil {
			return "", nil, err
		}
	} else {
		// Verify the user owns the chat they are posting to
		currentChat, err = s.chatRepo.FindByID(ctx, chatID)
		if err != nil || currentChat.UserID != userID {
			return "", nil, errors.New("unauthorized")
		}
	}

	// Save the user's message to the database.
	if err := s.SaveMessage(ctx, currentChat.ID, "user", userMessage); err != nil {
		return "", currentChat, err
	}

	// --- Get the REAL AI response ---
	assistantReply, err := s.aiService.GetCompletion(ctx, userMessage)
	if err != nil {
		return "", currentChat, err
	}

	// Save the assistant's message to the database.
	if err := s.SaveMessage(ctx, currentChat.ID, "assistant", assistantReply); err != nil {
		return "", currentChat, err
	}

	return assistantReply, currentChat, nil
}


// CreateNewChat creates a new chat for a user.
func (s *ChatService) CreateNewChat(ctx context.Context, userID uint, firstMessage string) (*domain.Chat, error) {
	words := strings.Split(firstMessage, " ")
	title := firstMessage
	if len(words) > 5 {
		title = strings.Join(words[:5], " ") + "..."
	}
	chat := &domain.Chat{UserID: userID, Title: title}
	return s.chatRepo.Create(ctx, chat)
}

// SaveMessage saves a new message to a chat.
func (s *ChatService) SaveMessage(ctx context.Context, chatID uint, role, content string) error {
	message := &domain.Message{ChatID: chatID, Role: role, Content: content}
	return s.messageRepo.Create(ctx, message)
}

// GetChatMessages retrieves all messages for a given chat.
func (s *ChatService) GetChatMessages(ctx context.Context, chatID uint, userID uint) ([]domain.Message, error) {
	chat, err := s.chatRepo.FindByID(ctx, chatID)
	if err != nil { return nil, err }
	if chat.UserID != userID { return nil, errors.New("unauthorized") }
	return s.messageRepo.FindByChatID(ctx, chatID)
}

// GetUserChats retrieves all chat histories for a user.
func (s *ChatService) GetUserChats(ctx context.Context, userID uint) ([]domain.Chat, error) {
    return s.chatRepo.FindByUserID(ctx, userID)
}

// Add this function to chat_service.go
func (s *ChatService) DeleteChat(ctx context.Context, chatID uint, userID uint) error {
	// In a real app, you might also delete all messages associated with the chat.
	// For now, we'll just delete the chat itself.
	return s.chatRepo.Delete(ctx, chatID, userID)
}