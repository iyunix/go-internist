// File: internal/services/chat_service.go
// FIXED: This version correctly saves the assistant's message to the database after streaming.
package services

import (
	"context"
	"strings"
	"github.com/iyunix/go-internist/internal/domain"
	"github.com/iyunix/go-internist/internal/repository/chat"
	"github.com/iyunix/go-internist/internal/repository/message"
	chatservice "github.com/iyunix/go-internist/internal/services/chat"
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

	config := chatservice.DefaultConfig()
	if retrievalTopK > 0 {
		config.RetrievalTopK = retrievalTopK
	}

	if err := config.Validate(); err != nil {
		return nil, chatservice.NewValidationError("config", err.Error())
	}

	logger := NewLogger("chat_service") // Use the application logger

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

// --- THIS IS THE NEW CORE LOGIC ---
// streamAndSaveAssistantResponse is a new private method that handles the entire process.
func (s *ChatService) streamAndSaveAssistantResponse(
    ctx context.Context,
    userID, chatID uint,
    prompt string,
    onDelta func(string) error,
    onSources func([]string),
) error {
    // Create a buffer to collect the full AI response as it streams.
    var responseBuilder strings.Builder

    // Wrap the original onDelta function. This new function will do two things:
    // 1. Send the token to the user's browser for real-time display.
    // 2. Add the token to our buffer to build the complete message.
    wrappedOnDelta := func(token string) error {
        responseBuilder.WriteString(token)
        if onDelta != nil {
            return onDelta(token)
        }
        return nil
    }

    // Call the streaming service with our wrapped function.
    err := s.streamService.StreamChatResponse(ctx, userID, chatID, prompt, wrappedOnDelta, onSources)

    // After the stream finishes, save the complete message to the database.
    if err == nil {
        fullResponse := responseBuilder.String()
        if strings.TrimSpace(fullResponse) != "" {
            assistantMessage := &domain.Message{
                ChatID:      chatID,
                Content:     fullResponse,
                MessageType: domain.MessageTypeAssistant, // This is the critical fix!
            }
            
            // We use context.Background() here because the original request context
            // might have been cancelled by the client disconnecting. We still want to save the message.
            _, saveErr := s.messageRepo.Create(context.Background(), assistantMessage)
            if saveErr != nil {
                s.logger.Error("CRITICAL: Failed to save assistant's response", "error", saveErr, "chat_id", chatID)
            }
        }
    }

    return err
}

// StreamChatMessage now calls our new private method.
func (s *ChatService) StreamChatMessage(
	ctx context.Context,
	userID, chatID uint,
	prompt string,
	onDelta func(string) error,
) error {
	return s.streamAndSaveAssistantResponse(ctx, userID, chatID, prompt, onDelta, nil)
}

// StreamChatMessageWithSources also calls our new private method.
func (s *ChatService) StreamChatMessageWithSources(
	ctx context.Context,
	userID, chatID uint,
	prompt string,
	onDelta func(string) error,
	onSources func([]string),
) error {
	return s.streamAndSaveAssistantResponse(ctx, userID, chatID, prompt, onDelta, onSources)
}

// --- The rest of the file remains the same ---

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

func (s *ChatService) AddChatMessage(ctx context.Context, userID, chatID uint, content string) (string, domain.Chat, error) {
	return "This is the non-streaming endpoint.", domain.Chat{}, nil
}

func (s *ChatService) ExtractSourceTitles(matches []*pinecone.ScoredVector) []string {
	return s.sourceExtractor.ExtractSources(matches)
}

func (s *ChatService) GetUserChatsWithPagination(ctx context.Context, userID uint, limit, offset int) ([]domain.Chat, int64, error) {
	return s.chatRepo.FindByUserIDWithPagination(ctx, userID, limit, offset)
}

func (s *ChatService) GetUserChats(ctx context.Context, userID uint) ([]domain.Chat, error) {
	chats, _, err := s.chatRepo.FindByUserIDWithPagination(ctx, userID, 100, 0)
	return chats, err
}

func (s *ChatService) SaveMessage(ctx context.Context, userID, chatID uint, content, messageType string) (*domain.Message, error) {
	chatRecord, err := s.chatRepo.FindByID(ctx, chatID)
	if err != nil || chatRecord.UserID != userID {
		return nil, chatservice.NewUnauthorizedError(userID, chatID)
	}
	message := &domain.Message{
		ChatID:      chatID,
		Content:     content,
		MessageType: messageType,
		Archived:    false,
	}
	savedMessage, err := s.messageRepo.Create(ctx, message)
	if err != nil {
		return nil, chatservice.NewRAGError("save_message", "could not save message", err)
	}
	return savedMessage, nil
}
