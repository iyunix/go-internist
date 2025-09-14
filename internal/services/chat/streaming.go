// File: internal/services/chat/streaming.go
package chat

import (
	"context"
	"strings"

	"github.com/iyunix/go-internist/internal/domain"
	"github.com/iyunix/go-internist/internal/repository/chat"
	"github.com/iyunix/go-internist/internal/repository/message"
	"github.com/pinecone-io/go-pinecone/v4/pinecone"
)

type StreamingService struct {
	config          *Config
	chatRepo        chat.ChatRepository
	messageRepo     message.MessageRepository
	aiService       AIProvider
	pineconeService PineconeProvider
	ragService      *RAGService
	sourceExtractor *SourceExtractor
	logger          Logger
}

type AIProvider interface {
	CreateEmbedding(ctx context.Context, text string) ([]float32, error)
	StreamCompletion(ctx context.Context, model, prompt string, onDelta func(string) error) error
}

type PineconeProvider interface {
	QuerySimilar(ctx context.Context, embedding []float32, topK int) ([]*pinecone.ScoredVector, error)
}

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

// StreamChatResponse handles the complete streaming chat flow
func (s *StreamingService) StreamChatResponse(
	ctx context.Context,
	userID, chatID uint,
	prompt string,
	onDelta func(string) error,
	onSources func([]string),
) error {
	s.logger.Info("starting stream chat", "user_id", userID, "chat_id", chatID)

	// Validate chat ownership
	chat, err := s.chatRepo.FindByID(ctx, chatID)
	if err != nil || chat.UserID != userID {
		return NewUnauthorizedError(userID, chatID)
	}

	// =================================================================
	// CRITICAL FIX: The block that saved the user message here is removed
	// to prevent duplication, as the frontend now handles it.
	// =================================================================

	// Get embedding for RAG
	embedding, err := s.aiService.CreateEmbedding(ctx, prompt)
	if err != nil {
		return NewRAGError("embedding", "failed to create embedding", err)
	}

	// Query similar documents
	matches, err := s.pineconeService.QuerySimilar(ctx, embedding, s.config.RetrievalTopK)
	if err != nil {
		return NewRAGError("pinecone_query", "failed to query Pinecone", err)
	}

	// Extract and send sources
	if s.config.EnableSources && onSources != nil {
		sources := s.sourceExtractor.ExtractSources(matches)
		if len(sources) > 0 {
			onSources(sources)
		}
	}

	// Build RAG context and prompt
	contextJSON := s.ragService.BuildContext(matches)
	finalPrompt := s.ragService.BuildPrompt(contextJSON, prompt)

	// Stream AI response
	var fullReply strings.Builder
	streamErr := s.aiService.StreamCompletion(ctx, s.config.StreamModel, finalPrompt, func(token string) error {
		fullReply.WriteString(token)
		return onDelta(token)
	})

	if streamErr != nil {
		s.logger.Error("stream completion failed", "error", streamErr)
		return NewRAGError("streaming", "AI streaming failed", streamErr)
	}

	// Save assistant response asynchronously
	go s.saveAssistantMessage(chatID, fullReply.String())

	s.logger.Info("stream chat completed", "response_length", fullReply.Len())
	return nil
}

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

func (s *StreamingService) saveAssistantMessage(chatID uint, content string) {
	if len(content) > 0 {
		aiMessage := &domain.Message{
			ChatID:      chatID,
			MessageType: domain.MessageTypeAssistant,
			Content:     content,
		}
		if _, err := s.messageRepo.Create(context.Background(), aiMessage); err != nil {
			s.logger.Error("failed to save assistant message", "error", err)
		}
		_ = s.chatRepo.TouchUpdatedAt(context.Background(), chatID)
	}
}