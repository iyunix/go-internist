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

// AIService provides methods for LLM chat completions and embeddings.
type AIService struct {
	embeddingClient    *openai.Client
	llmClient          *openai.Client
	embeddingModelName string
	timeout            time.Duration
	maxRetries         int
}

// ========== MODIFIED FUNCTION START ==========

// NewAIService initializes AIService with separate clients for embeddings and LLM.
func NewAIService(embeddingKey, llmKey, embeddingBaseURL, llmBaseURL, embeddingModelName string) (*AIService, error) {
	// --- ADDED VALIDATION ---
	if llmKey == "" {
		return nil, errors.New("LLM API key is required but was not provided")
	}
	if embeddingKey == "" {
		return nil, errors.New("embedding API key is required but was not provided")
	}
	// --- END ADDED VALIDATION ---

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
	}, nil // Return the service and a nil error on success
}

// ========== MODIFIED FUNCTION END ==========

// systemJSONGuard enforces strict JSON output from the model.
func systemJSONGuard() string {
	return "You must output STRICT JSON only that begins with '{' and ends with '}', no code fences, no extra text, and complete each key-value pair before moving on. No trailing commas. If you cannot comply, return an empty object {}."
}

// GetCompletion returns a non-streamed chat completion from the LLM.
func (s *AIService) GetCompletion(ctx context.Context, model, prompt string) (string, error) {
	// --- ADDED NIL CHECK ---
	if s == nil || s.llmClient == nil {
		return "", errors.New("AIService or its llmClient is not initialized")
	}
	// --- END NIL CHECK ---
	var reply string
	err := s.retryWithTimeout(func(ctx context.Context) error {
		req := openai.ChatCompletionRequest{
			Model: model,
			Messages: []openai.ChatCompletionMessage{
				{Role: openai.ChatMessageRoleSystem, Content: systemJSONGuard()},
				{Role: openai.ChatMessageRoleUser, Content: prompt},
			},
			Temperature: 0.1,
			TopP:        0.9,
			ResponseFormat: &openai.ChatCompletionResponseFormat{
				Type: openai.ChatCompletionResponseFormatTypeJSONObject,
			},
		}

		resp, err := s.llmClient.CreateChatCompletion(ctx, req)
		if err != nil {
			return fmt.Errorf("CreateChatCompletion error: %w", err)
		}

		if len(resp.Choices) == 0 || resp.Choices[0].Message.Content == "" {
			return errors.New("language model returned empty reply")
		}

		reply = resp.Choices[0].Message.Content
		return nil
	})

	return reply, err
}

// CreateEmbedding generates an embedding vector for the given text.
func (s *AIService) CreateEmbedding(ctx context.Context, text string) ([]float32, error) {
	// --- ADDED NIL CHECK ---
	if s == nil || s.embeddingClient == nil {
		return nil, errors.New("AIService or its embeddingClient is not initialized")
	}
	// --- END NIL CHECK ---
	var embedding []float32
	err := s.retryWithTimeout(func(ctx context.Context) error {
		req := openai.EmbeddingRequest{
			Input: []string{text},
			Model: openai.EmbeddingModel(s.embeddingModelName),
		}

		resp, err := s.embeddingClient.CreateEmbeddings(ctx, req)
		if err != nil {
			return fmt.Errorf("CreateEmbeddings error: %w", err)
		}

		if len(resp.Data) == 0 || len(resp.Data[0].Embedding) == 0 {
			return errors.New("embedding API returned empty response")
		}

		embedding = resp.Data[0].Embedding
		return nil
	})

	return embedding, err
}

// StreamCompletion streams the LLM's response in chunks, calling onDelta for each chunk.
func (s *AIService) StreamCompletion(ctx context.Context, model, prompt string, onDelta func(string) error) error {
	// --- ADDED NIL CHECK ---
	if s == nil || s.llmClient == nil {
		return errors.New("AIService or its llmClient is not initialized")
	}
	// --- END NIL CHECK ---
	req := openai.ChatCompletionRequest{
		Model:  model,
		Stream: true,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleUser, Content: prompt},
		},
		Temperature: 0.1,
		TopP:        0.9,
	}

	stream, err := s.llmClient.CreateChatCompletionStream(ctx, req)
	if err != nil {
		var apiErr *openai.APIError
		if errors.As(err, &apiErr) {
			log.Printf("[AIService] OpenAI API Error on stream creation: status=%d type=%s message=%s", apiErr.HTTPStatusCode, apiErr.Type, apiErr.Message)
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
			var apiErr *openai.APIError
			if errors.As(err, &apiErr) {
				log.Printf("[AIService] OpenAI API Error during stream receive: status=%d type=%s message=%s", apiErr.HTTPStatusCode, apiErr.Type, apiErr.Message)
			}
			return fmt.Errorf("stream receive error: %w", err)
		}

		if len(resp.Choices) > 0 {
			if delta := resp.Choices[0].Delta.Content; delta != "" && onDelta != nil {
				if cbErr := onDelta(delta); cbErr != nil {
					return cbErr
				}
			}
		}
	}
}

// retryWithTimeout retries a function multiple times with a timeout for each attempt.
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