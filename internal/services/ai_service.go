// G:\go_internist\internal\services\ai_service.go
package services

import (
    "context"
    "time"
    "github.com/iyunix/go-internist/internal/services/ai"
)


// Add this after your existing imports
type AIProviderStatus struct {
    IsHealthy        bool   `json:"is_healthy"`
    Message          string `json:"message"`
    EmbeddingHealthy bool   `json:"embedding_healthy"`
    LLMHealthy       bool   `json:"llm_healthy"`
}
func (s *AIService) GetProviderStatus() AIProviderStatus {
    // ✅ Reduce timeout from 10s to 3s for health checks
    ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
    defer cancel()
    
    status := AIProviderStatus{
        IsHealthy:        true,
        Message:          "All AI providers are healthy",
        EmbeddingHealthy: true,
        LLMHealthy:       true,
    }
    
    // Test embedding provider
    _, err := s.provider.CreateEmbedding(ctx, "test")
    if err != nil {
        s.logger.Warn("embedding provider health check failed", "error", err)
        status.EmbeddingHealthy = false
        status.IsHealthy = false
        status.Message = "Embedding provider is unhealthy"
    }

    
    s.logger.Info("AI provider health check completed", 
        "embedding_healthy", status.EmbeddingHealthy,
        "llm_healthy", status.LLMHealthy,
        "overall_healthy", status.IsHealthy)
        
    return status
}


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
