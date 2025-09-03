// File: internal/services/ai_service.go
package services

import (
	"context"

	"github.com/sashabaranov/go-openai"
)

type AIService struct {
	embeddingClient *openai.Client
	llmClient       *openai.Client
}

// NewAIService creates clients for both embedding and chat completion.
func NewAIService(embeddingKey, llmKey string) *AIService {
	// Configure the client for creating embeddings
	embeddingConfig := openai.DefaultConfig(embeddingKey)
	embeddingConfig.BaseURL = "https://api.avalai.ir/v1"
	
	// Configure the client for generating answers
	llmConfig := openai.DefaultConfig(llmKey)
	llmConfig.BaseURL = "https://api.avalai.ir/v1"

	return &AIService{
		embeddingClient: openai.NewClientWithConfig(embeddingConfig),
		llmClient:       openai.NewClientWithConfig(llmConfig),
	}
}

// CreateEmbedding creates a vector embedding for a given text.
func (s *AIService) CreateEmbedding(ctx context.Context, text string) ([]float32, error) {
	req := openai.EmbeddingRequest{
		Input: []string{text},
		Model: "text-embedding-3-large", // Using a standard, well-known model
	}
	resp, err := s.embeddingClient.CreateEmbeddings(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp.Data[0].Embedding, nil
}

// GetCompletion generates a chat response based on a full prompt with context.
func (s *AIService) GetCompletion(ctx context.Context, prompt string) (string, error) {
	req := openai.ChatCompletionRequest{
		Model: "gpt-5-nano", // Your specified answer model
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: prompt,
			},
		},
	}
	resp, err := s.llmClient.CreateChatCompletion(ctx, req)
	if err != nil {
		return "", err
	}
	return resp.Choices[0].Message.Content, nil
}