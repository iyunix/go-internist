// File: internal/services/ai_service.go
package services

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/sashabaranov/go-openai"
)

type AIService struct {
	embeddingClient *openai.Client
	llmClient       *openai.Client
	timeout         time.Duration // New: default API timeout duration
	maxRetries      int           // New: maximum retry attempts
}

// NewAIService creates clients for both embedding and chat completion.
// Adds production-ready settings: API timeout, retry count.
func NewAIService(embeddingKey, llmKey string) *AIService {
	embeddingConfig := openai.DefaultConfig(embeddingKey)
	embeddingConfig.BaseURL = "https://api.avalai.ir/v1"

	llmConfig := openai.DefaultConfig(llmKey)
	llmConfig.BaseURL = "https://api.avalai.ir/v1"

	return &AIService{
		embeddingClient: openai.NewClientWithConfig(embeddingConfig),
		llmClient:       openai.NewClientWithConfig(llmConfig),
		// --- THE ONLY CHANGE IS HERE ---
		timeout:    60 * time.Second, // Increased timeout to 60 seconds for slow AI responses
		maxRetries: 3,                // reasonable retry attempts
	}
}

// retryWithTimeout wraps an API call with context timeout and retry logic.
func (s *AIService) retryWithTimeout(call func(ctx context.Context) error) error {
	var lastErr error
	for attempt := 1; attempt <= s.maxRetries; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
		defer cancel()
		err := call(ctx)
		if err == nil {
			return nil
		}
		lastErr = err
		log.Printf("[AIService] Retry %d/%d failed: %v", attempt, s.maxRetries, err)
		time.Sleep(time.Duration(attempt) * time.Second)
	}
	return lastErr
}

// CreateEmbedding creates a vector embedding for a given text, with retries and timeout.
func (s *AIService) CreateEmbedding(ctx context.Context, text string) ([]float32, error) {
	var embedding []float32
	err := s.retryWithTimeout(func(ctx context.Context) error {
		req := openai.EmbeddingRequest{
			Input: []string{text},
			Model: "text-embedding-3-large",
		}
		resp, err := s.embeddingClient.CreateEmbeddings(ctx, req)
		if err != nil {
			return err
		}
		if len(resp.Data) == 0 || len(resp.Data[0].Embedding) == 0 {
			return errors.New("embedding API returned empty response")
		}
		embedding = resp.Data[0].Embedding
		return nil
	})
	return embedding, err
}

// GetCompletion generates a chat response with retries and timeout.
func (s *AIService) GetCompletion(ctx context.Context, prompt string) (string, error) {
	var reply string
	err := s.retryWithTimeout(func(ctx context.Context) error {
		req := openai.ChatCompletionRequest{
			Model: "gpt-5-nano",
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt,
				},
			},
		}
		resp, err := s.llmClient.CreateChatCompletion(ctx, req)
		if err != nil {
			return err
		}
		if len(resp.Choices) == 0 || resp.Choices[0].Message.Content == "" {
			return errors.New("language model returned empty reply")
		}
		reply = resp.Choices[0].Message.Content
		return nil
	})
	return reply, err
}
