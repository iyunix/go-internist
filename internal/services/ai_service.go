// G:\go_internist\internal\services\ai_service.go
package services

import (
    "context"
    "github.com/iyunix/go-internist/internal/services/ai"
)

type AIService struct {
    provider ai.AIProvider
    logger   Logger
}

func NewAIService(provider ai.AIProvider, logger Logger) *AIService {
    return &AIService{
        provider: provider,
        logger:   logger,
    }
}

func (s *AIService) CreateEmbedding(ctx context.Context, text string) ([]float32, error) {
    s.logger.Info("creating embedding", "text_length", len(text))
    
    embedding, err := s.provider.CreateEmbedding(ctx, text)
    if err != nil {
        s.logger.Error("embedding creation failed", "error", err)
        return nil, err
    }
    
    s.logger.Info("embedding created successfully", "dimension", len(embedding))
    return embedding, nil
}

func (s *AIService) GetCompletion(ctx context.Context, model, prompt string) (string, error) {
    s.logger.Info("getting completion", "model", model, "prompt_length", len(prompt))
    
    completion, err := s.provider.GetCompletion(ctx, model, prompt)
    if err != nil {
        s.logger.Error("completion failed", "error", err, "model", model)
        return "", err
    }
    
    s.logger.Info("completion successful", "response_length", len(completion))
    return completion, nil
}

func (s *AIService) StreamCompletion(ctx context.Context, model, prompt string, onDelta func(string) error) error {
    s.logger.Info("starting stream completion", "model", model)
    
    err := s.provider.StreamCompletion(ctx, model, prompt, onDelta)
    if err != nil {
        s.logger.Error("stream completion failed", "error", err)
        return err
    }
    
    s.logger.Info("stream completion finished")
    return nil
}
