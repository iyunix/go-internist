// File: internal/services/ai/openai_provider.go
package ai

import (
    "context"
    "io"
    openai "github.com/sashabaranov/go-openai"
)

type OpenAIProvider struct {
    config          *Config
    embeddingClient *openai.Client
    llmClient       *openai.Client
}

func NewOpenAIProvider(config *Config) *OpenAIProvider {
    // ✅ FIXED: Use exact same approach as diagnostic test for LLM client
    llmConfig := openai.DefaultConfig(config.LLMKey)
    llmConfig.BaseURL = config.LLMBaseURL
    llmClient := openai.NewClientWithConfig(llmConfig)

    // Embedding client (separate configuration)
    embeddingConfig := openai.DefaultConfig(config.EmbeddingKey)
    if config.EmbeddingBaseURL != "" {
        embeddingConfig.BaseURL = config.EmbeddingBaseURL
    }
    embeddingClient := openai.NewClientWithConfig(embeddingConfig)
    
    return &OpenAIProvider{
        config:          config,
        embeddingClient: embeddingClient,
        llmClient:       llmClient,
    }
}

func (p *OpenAIProvider) CreateEmbedding(ctx context.Context, text string) ([]float32, error) {
    req := openai.EmbeddingRequest{
        Input: []string{text},
        Model: openai.EmbeddingModel(p.config.EmbeddingModel),
    }
    
    resp, err := p.embeddingClient.CreateEmbeddings(ctx, req)
    if err != nil {
        return nil, NewProviderError("embedding", "failed to create embedding", err)
    }
    
    if len(resp.Data) == 0 || len(resp.Data[0].Embedding) == 0 {
        return nil, &AIError{
            Type:      ErrTypeProvider,
            Operation: "embedding",
            Message:   "empty embedding response",
        }
    }
    
    return resp.Data[0].Embedding, nil
}

// ✅ FIXED: Use exact same approach as diagnostic test for chat completion
func (p *OpenAIProvider) GetCompletion(ctx context.Context, model, prompt string) (string, error) {
    resp, err := p.llmClient.CreateChatCompletion(
        ctx,
        openai.ChatCompletionRequest{
            Model: model,
            Messages: []openai.ChatCompletionMessage{
                {
                    Role:    openai.ChatMessageRoleUser,
                    Content: prompt,
                },
            },
            Temperature: p.config.Temperature,
            TopP:        p.config.TopP,
        },
    )
    
    if err != nil {
        return "", NewProviderError("completion", "failed to create completion", err)
    }
    
    if len(resp.Choices) == 0 || resp.Choices[0].Message.Content == "" {
        return "", &AIError{
            Type:      ErrTypeProvider,
            Operation: "completion",
            Message:   "empty completion response",
        }
    }
    
    return resp.Choices[0].Message.Content, nil
}

// ✅ FIXED: Simple streaming based on diagnostic test approach
func (p *OpenAIProvider) StreamCompletion(ctx context.Context, model, prompt string, onDelta func(string) error) error {
    stream, err := p.llmClient.CreateChatCompletionStream(ctx, openai.ChatCompletionRequest{
        Model: model,
        Messages: []openai.ChatCompletionMessage{
            {
                Role:    openai.ChatMessageRoleUser,
                Content: prompt,
            },
        },
        Temperature: p.config.Temperature,
        TopP:        p.config.TopP,
    })

    if err != nil {
        return NewProviderError("streaming", "failed to create stream", err)
    }
    defer stream.Close()

    for {
        response, err := stream.Recv()
        if err != nil {
            if err == io.EOF {
                return nil
            }
            return NewProviderError("streaming", "stream receive error", err)
        }

        if len(response.Choices) > 0 {
            delta := response.Choices[0].Delta.Content
            if delta != "" && onDelta != nil {
                if cbErr := onDelta(delta); cbErr != nil {
                    return cbErr
                }
            }
        }
    }
}

func (p *OpenAIProvider) HealthCheck(ctx context.Context) error {
    return nil
}

func (p *OpenAIProvider) GetStatus(ctx context.Context) ProviderStatus {
    return ProviderStatus{
        IsHealthy:        true,
        EmbeddingHealthy: true,
        LLMHealthy:       true,
        Message:          "OpenAI provider healthy",
    }
}
