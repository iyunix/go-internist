// File: internal/services/chat_service.go
package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/iyunix/go-internist/internal/domain"
	"github.com/iyunix/go-internist/internal/repository"
)

type ChatService struct {
	chatRepo        repository.ChatRepository
	messageRepo     repository.MessageRepository
	aiService       *AIService
	pineconeService *PineconeService
	retrievalTopK   int
}

func NewChatService(
	chatRepo repository.ChatRepository,
	messageRepo repository.MessageRepository,
	aiService *AIService,
	pineconeService *PineconeService,
	retrievalTopK int,
) *ChatService {
	if retrievalTopK <= 0 {
		retrievalTopK = 8 // Default value
	}
	return &ChatService{
		chatRepo:        chatRepo,
		messageRepo:     messageRepo,
		aiService:       aiService,
		pineconeService: pineconeService,
		retrievalTopK:   retrievalTopK,
	}
}

// CreateChat creates a new chat record in the database.
func (s *ChatService) CreateChat(ctx context.Context, userID uint, title string) (*domain.Chat, error) {
	if strings.TrimSpace(title) == "" {
		return nil, errors.New("chat title cannot be empty")
	}
	if len(title) > 100 {
		title = title[:100]
	}

	newChat := &domain.Chat{
		UserID: userID,
		Title:  title,
	}

	createdChat, err := s.chatRepo.Create(ctx, newChat)
	if err != nil {
		log.Printf("[ChatService] Failed to create chat for user %d: %v", userID, err)
		return nil, fmt.Errorf("could not create chat: %w", err)
	}
	return createdChat, nil
}

// StreamChatMessage handles the entire RAG and streaming process.
func (s *ChatService) StreamChatMessage(
	ctx context.Context,
	userID, chatID uint,
	prompt string,
	onDelta func(token string) error,
) error {
	// 1. Validate chat ownership
	chat, err := s.chatRepo.FindByID(ctx, chatID)
	if err != nil || chat.UserID != userID {
		return errors.New("chat not found or unauthorized")
	}

	// 2. Save the user's message
	userMessage := &domain.Message{ChatID: chatID, Role: "user", Content: prompt}
	if _, err := s.messageRepo.Create(ctx, userMessage); err != nil {
		return fmt.Errorf("failed to store user message: %w", err)
	}

	// 3. Perform RAG (Embedding + Pinecone Search)
	embedding, err := s.aiService.CreateEmbedding(ctx, prompt)
	if err != nil {
		return fmt.Errorf("failed to create embedding: %w", err)
	}
	matches, err := s.pineconeService.QuerySimilar(ctx, embedding, s.retrievalTopK)
	if err != nil {
		return fmt.Errorf("failed to query pinecone: %w", err)
	}

	var contextBuilder strings.Builder
	log.Println("--- [RAG] Retrieved Context from Pinecone ---")
	for i, match := range matches {
		if match != nil && match.Vector != nil && match.Vector.Metadata != nil {
			// CORRECTED METADATA FIELD NAME
			source := match.Vector.Metadata.Fields["source_file"].GetStringValue()
			section := match.Vector.Metadata.Fields["section_heading"].GetStringValue()
			takeaway := match.Vector.Metadata.Fields["key_takeaways"].GetStringValue()
			content := match.Vector.Metadata.Fields["text"].GetStringValue()

			log.Printf("[RAG Context] Chunk %d Source: %s", i+1, source)

			contextBuilder.WriteString(fmt.Sprintf("--- Context Chunk %d ---\n", i+1))
			contextBuilder.WriteString(fmt.Sprintf("Source: %s\n", source))
			contextBuilder.WriteString(fmt.Sprintf("Section: %s\n", section))
			contextBuilder.WriteString(fmt.Sprintf("Key Takeaway: %s\n", takeaway))
			contextBuilder.WriteString(fmt.Sprintf("Content:\n%s\n\n", content))
		}
	}
	log.Println("-------------------------------------------")

	finalPrompt := s.buildFinalPrompt(contextBuilder.String(), prompt)

	// 4. Stream the AI response
	var fullReply strings.Builder
	streamErr := s.aiService.StreamCompletion(ctx, "jabir-400b", finalPrompt, func(token string) error {
		fullReply.WriteString(token)
		return onDelta(token)
	})
	if streamErr != nil {
		log.Printf("[ChatService] Error during AI stream: %v", streamErr)
		return streamErr
	}

	// 5. Save the full AI response after the stream is complete
	go func() {
		if fullReply.Len() > 0 {
			assistantMessage := &domain.Message{ChatID: chatID, Role: "assistant", Content: fullReply.String()}
			if _, err := s.messageRepo.Create(context.Background(), assistantMessage); err != nil {
				log.Printf("Failed to save assistant message: %v", err)
			}
		}
	}()
	return nil
}

// buildFinalPrompt creates the final prompt with context for the LLM.
func (s *ChatService) buildFinalPrompt(contextText, question string) string {
	if strings.TrimSpace(contextText) == "" {
		contextText = "No relevant information was found in the database."
	}
	return fmt.Sprintf(`ROLE: You are an expert AI assistant named Internist.

TASK:
- Your primary task is to answer the user's QUESTION based *ONLY* on the structured information provided in the CONTEXT below.
- Synthesize a comprehensive answer from all the provided context chunks.
- **CRUCIAL**: After each sentence or key piece of information in your answer, you MUST provide a citation referencing the source file, like this: (Source: Cardiovascular/cardio_cad/some_article.md).
- If the CONTEXT does not contain the information needed to answer the QUESTION, you MUST state that the answer is not available in the provided documents. Do not use your own knowledge.

CONTEXT:
%s
---

QUESTION:
%s
`, contextText, question)
}

// --- Other existing functions ---
func (s *ChatService) GetUserChats(ctx context.Context, userID uint) ([]domain.Chat, error) {
	return s.chatRepo.FindByUserID(ctx, userID)
}
func (s *ChatService) GetChatMessages(ctx context.Context, userID, chatID uint) ([]domain.Message, error) {
	chat, err := s.chatRepo.FindByID(ctx, chatID)
	if err != nil || chat.UserID != userID {
		return nil, errors.New("chat not found or unauthorized")
	}
	return s.messageRepo.FindByChatID(ctx, chatID)
}
func (s *ChatService) DeleteChat(ctx context.Context, userID, chatID uint) error {
	chat, err := s.chatRepo.FindByID(ctx, chatID)
	if err != nil || chat.UserID != userID {
		return errors.New("chat not found or unauthorized")
	}
	return s.chatRepo.Delete(ctx, chatID, userID)
}
func (s *ChatService) AddChatMessage(ctx context.Context, userID, chatID uint, content string) (string, domain.Chat, error) {
	return "This is the non-streaming endpoint.", domain.Chat{}, nil
}