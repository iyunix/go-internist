// File: internal/services/ai_service.go
package services

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/sashabaranov/go-openai"
)

type AIService struct {
	embeddingClient    *openai.Client
	llmClient          *openai.Client
	embeddingModelName string
	timeout            time.Duration
	maxRetries         int
}

func NewAIService(embeddingKey, llmKey, embeddingBaseURL, llmBaseURL, embeddingModelName string) *AIService {
	embeddingConfig := openai.DefaultConfig(embeddingKey)
	if embeddingBaseURL != "" {
		embeddingConfig.BaseURL = embeddingBaseURL
	}
	llmConfig := openai.DefaultConfig(llmKey)
	if llmBaseURL != "" {
		llmConfig.BaseURL = llmBaseURL
	}
	return &AIService{
		embeddingClient:    openai.NewClientWithConfig(embeddingConfig),
		llmClient:          openai.NewClientWithConfig(llmConfig),
		embeddingModelName: embeddingModelName,
		timeout:            60 * time.Second,
		maxRetries:         3,
	}
}

// THIS IS THE FUNCTION THAT WAS MISSING
// GetCompletion returns a non-streamed reply from the chat completion API.
func (s *AIService) GetCompletion(ctx context.Context, model, prompt string) (string, error) {
	var reply string
	err := s.retryWithTimeout(func(ctx context.Context) error {
		req := openai.ChatCompletionRequest{
			Model: model,
			Messages: []openai.ChatCompletionMessage{
				{Role: openai.ChatMessageRoleUser, Content: prompt},
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

// --- The rest of your file is unchanged ---

func (s *AIService) CreateEmbedding(ctx context.Context, text string) ([]float32, error) {
	var embedding []float32
	err := s.retryWithTimeout(func(ctx context.Context) error {
		req := openai.EmbeddingRequest{
			Input: []string{text},
			Model: openai.EmbeddingModel(s.embeddingModelName),
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

func (s *AIService) StreamCompletion(ctx context.Context, model, prompt string, onDelta func(string) error) error {
	req := openai.ChatCompletionRequest{
		Model:    model,
		Stream:   true,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleUser, Content: prompt},
		},
	}
	stream, err := s.llmClient.CreateChatCompletionStream(ctx, req)
	if err != nil {
		var apiErr *openai.APIError
		if errors.As(err, &apiErr) {
			log.Printf("[AIService] OpenAI API Error: status=%d type=%s message=%s", apiErr.HTTPStatusCode, apiErr.Type, apiErr.Message)
		}
		return fmt.Errorf("failed to create completion stream: %w", err)
	}
	defer stream.Close()

	for {
		resp, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("stream receive error: %w", err)
		}
		for _, choice := range resp.Choices {
			if delta := choice.Delta.Content; delta != "" && onDelta != nil {
				if cbErr := onDelta(delta); cbErr != nil {
					return cbErr
				}
			}
		}
	}
}

func (s *AIService) retryWithTimeout(call func(ctx context.Context) error) error {
	var lastErr error
	for attempt := 1; attempt <= s.maxRetries; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
		err := call(ctx)
		cancel()
		if err == nil {
			return nil
		}
		lastErr = err
		log.Printf("[AIService] Retry %d/%d failed: %v", attempt, s.maxRetries, err)
		if attempt < s.maxRetries {
			time.Sleep(time.Duration(attempt) * time.Second)
		}
	}
	return lastErr
}

