// G:\go_internist\internal\services\ai\interface.go
package ai

import "context"

// ProviderStatus represents AI provider health
type ProviderStatus struct {
    IsHealthy         bool
    EmbeddingHealthy  bool
    LLMHealthy        bool
    Message           string
}

// EmbeddingProvider handles text embeddings
type EmbeddingProvider interface {
    CreateEmbedding(ctx context.Context, text string) ([]float32, error)
    HealthCheck(ctx context.Context) error
}

// CompletionProvider handles chat completions  
type CompletionProvider interface {
    GetCompletion(ctx context.Context, model, prompt string) (string, error)
    StreamCompletion(ctx context.Context, model, prompt string, onDelta func(string) error) error
    HealthCheck(ctx context.Context) error
}


// AIProvider combines embedding and completion capabilities
type AIProvider interface {
    EmbeddingProvider
    CompletionProvider
    GetStatus(ctx context.Context) ProviderStatus
}

// Service defines high-level AI service interface
type Service interface {
    CreateEmbedding(ctx context.Context, text string) ([]float32, error)
    GetCompletion(ctx context.Context, model, prompt string) (string, error)
    StreamCompletion(ctx context.Context, model, prompt string, onDelta func(string) error) error
    GetProviderStatus() ProviderStatus
}
