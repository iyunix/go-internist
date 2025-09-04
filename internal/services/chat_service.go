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
    // New dependency for RAG
    pineconeService *PineconeService
}

// Updated constructor: accepts pineconeService
func NewChatService(
    chatRepo repository.ChatRepository,
    messageRepo repository.MessageRepository,
    aiService *AIService,
    pineconeService *PineconeService,
) *ChatService {
    return &ChatService{
        chatRepo:        chatRepo,
        messageRepo:     messageRepo,
        aiService:       aiService,
        pineconeService: pineconeService,
    }
}

func (s *ChatService) GetUserChats(ctx context.Context, userID uint) ([]domain.Chat, error) {
    return s.chatRepo.FindByUserID(ctx, userID)
}

func (s *ChatService) AddChatMessage(ctx context.Context, userID, chatID uint, content string) (string, domain.Chat, error) {
    if content == "" {
        return "", domain.Chat{}, errors.New("message content cannot be empty")
    }

    var chat domain.Chat
    var err error

    // Part 1: find or create chat
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

    // Part 2: store user message
    userMessage := &domain.Message{
        ChatID:  chatID,
        Role:    "user",
        Content: content,
    }
    if _, err := s.messageRepo.Create(ctx, userMessage); err != nil {
        log.Printf("[ChatService] User message create error: %v", err)
        return "", domain.Chat{}, errors.New("failed to store user message")
    }

    // Part 3: RAG workflow

    // 1) Create embedding for the user's message
    embedding, err := s.aiService.CreateEmbedding(ctx, content)
    if err != nil {
        return "", domain.Chat{}, fmt.Errorf("failed to create embedding: %w", err)
    }

    // 2) Query Pinecone for similar vectors (ensure IncludeMetadata is enabled in the Pinecone service)
    matches, err := s.pineconeService.QuerySimilar(ctx, embedding, 3)
    if err != nil {
        return "", domain.Chat{}, fmt.Errorf("failed to query pinecone: %w", err)
    }

    // 3) Build textual context from metadata (expects a "text" field)
    var contextBuilder strings.Builder
    for _, m := range matches {
        if m != nil && m.Vector != nil && m.Vector.Metadata != nil {
            if v, ok := m.Vector.Metadata.Fields["text"]; ok {
                txt := v.GetStringValue()
                if txt != "" {
                    contextBuilder.WriteString(txt)
                    contextBuilder.WriteString("\n\n")
                }
            }
        }
    }


    // 4) Compose final prompt using retrieved context
    finalPrompt := s.buildFinalPrompt(contextBuilder.String(), content)

    // 5) Get completion from the LLM
    aiReply, err := s.aiService.GetCompletion(ctx, "jabir-400b", finalPrompt)
    if err != nil {
        log.Printf("[ChatService] AI service error: %v", err)
        return "", domain.Chat{}, errors.New("failed to get AI completion")
    }

    // Part 4: store assistant message
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

// Builds a grounded prompt for the model
func (s *ChatService) buildFinalPrompt(contextText, question string) string {
    return fmt.Sprintf(`Based on the following context, provide a direct and concise answer to the user's question. Do not mention the context in your response.

Context:
---
%s
---

Question: %s
`, contextText, question)
}

// Existing placeholder (kept for compatibility if referenced elsewhere)
func (s *ChatService) buildPromptWithContext(prompt string, context interface{}) string {
    return prompt
}

func (s *ChatService) GetChatMessages(ctx context.Context, userID, chatID uint) ([]domain.Message, error) {
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
    log.Printf("[ChatService] Found %d messages for chat %d", len(messages), chatID)
    return messages, nil
}

func (s *ChatService) DeleteChat(ctx context.Context, userID, chatID uint) error {
    chat, err := s.chatRepo.FindByID(ctx, chatID)
    if err != nil || chat.UserID != userID {
        log.Printf("[ChatService] DeleteChat invalid access user %d chat %d", userID, chatID)
        return errors.New("chat not found or unauthorized")
    }
    if err := s.chatRepo.Delete(ctx, chatID, userID); err != nil {
        log.Printf("[ChatService] DeleteChat DB error: %v", err)
        return errors.New("failed to delete chat")
    }
    return nil
}
