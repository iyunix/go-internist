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

// In internal/services/chat/streaming.go

// Add a new onStatus parameter to the function signature
func (s *StreamingService) StreamChatResponse(
	ctx context.Context,
	userID, chatID uint,
	prompt string,
	onDelta func(string) error,
	onSources func([]string),
	onStatus func(status, message string), // <-- ADD THIS NEW PARAMETER
) error {
	s.logger.Info("starting stream chat", "user_id", userID, "chat_id", chatID)
	
	// Send the first status update
	onStatus("understanding", "Understanding question...")

	// Validate chat ownership (no change here)
	chat, err := s.chatRepo.FindByID(ctx, chatID)
	if err != nil || chat.UserID != userID {
		return NewUnauthorizedError(userID, chatID)
	}

	// Get embedding for RAG (no change here)
	embedding, err := s.aiService.CreateEmbedding(ctx, prompt)
	if err != nil {
		return NewRAGError("embedding", "failed to create embedding", err)
	}

	// Send the next status update
	onStatus("searching", "Searching UpToDate knowledge base...")

	// Query similar documents (no change here)
	matches, err := s.pineconeService.QuerySimilar(ctx, embedding, s.config.RetrievalTopK)
	if err != nil {
		return NewRAGError("pinecone_query", "failed to query Pinecone", err)
	}

	// ... (Extracting sources logic is unchanged) ...
	if s.config.EnableSources && onSources != nil {
		sources := s.sourceExtractor.ExtractSources(matches)
		if len(sources) > 0 { onSources(sources) }
	}

	// Build RAG context and prompt (no change here)
	contextJSON := s.ragService.BuildContext(matches)
	finalPrompt := s.ragService.BuildPrompt(contextJSON, prompt)
	
	// Send the final status update before streaming
	onStatus("thinking", "AI is generating a response...")

	// Stream AI response (no change here)
	var fullReply strings.Builder
	streamErr := s.aiService.StreamCompletion(ctx, s.config.StreamModel, finalPrompt, func(token string) error {
		fullReply.WriteString(token)
		return onDelta(token)
	})
	
	if streamErr != nil {
		s.logger.Error("stream completion failed", "error", streamErr)
		return NewRAGError("streaming", "AI streaming failed", streamErr)
	}

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
