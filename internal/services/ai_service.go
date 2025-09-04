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
	timeout         time.Duration
	maxRetries      int
}

// NewAIService is now more flexible. It accepts custom base URLs for both services.
// If a baseURL is an empty string, it will use the OpenAI default.
func NewAIService(embeddingKey, llmKey, embeddingBaseURL, llmBaseURL string) *AIService {
	embeddingConfig := openai.DefaultConfig(embeddingKey)
	if embeddingBaseURL != "" {
		embeddingConfig.BaseURL = embeddingBaseURL
	}

	llmConfig := openai.DefaultConfig(llmKey)
	if llmBaseURL != "" {
		llmConfig.BaseURL = llmBaseURL
	}

	return &AIService{
		embeddingClient: openai.NewClientWithConfig(embeddingConfig),
		llmClient:       openai.NewClientWithConfig(llmConfig),
		timeout:         60 * time.Second,
		maxRetries:      3,
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
// THIS FUNCTION IS NOW UPDATED
func (s *AIService) GetCompletion(ctx context.Context, model, prompt string) (string, error) {
	var reply string
	err := s.retryWithTimeout(func(ctx context.Context) error {
		req := openai.ChatCompletionRequest{
			Model: model, // Model is now a parameter
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