// G:\go_internist\internal\services\chat\config.go
package chat

import (
    "fmt"
    "time"
)

type Config struct {
    // RAG Configuration
    RetrievalTopK    int           // Number of similar documents to retrieve
    ContextMaxTokens int           // Maximum tokens for context
    
    // Model Configuration
    ChatModel        string        // AI model for chat completions
    StreamModel      string        // AI model for streaming
    
    // Performance Configuration
    Timeout          time.Duration // Chat request timeout
    MaxRetries       int          // Maximum retry attempts
    
    // Medical AI Parameters
    Temperature      float32      // Model creativity (should be low for medical)
    MaxTokens        int          // Maximum response tokens
    
    // Citation Configuration
    EnableSources    bool         // Whether to extract source citations
    MaxSources       int          // Maximum number of sources to extract
}

func (c *Config) Validate() error {
    if c.RetrievalTopK <= 0 {
        return fmt.Errorf("retrieval_top_k must be positive")
    }
    if c.RetrievalTopK > 20 {
        return fmt.Errorf("retrieval_top_k cannot exceed 20")
    }
    if c.ChatModel == "" {
        return fmt.Errorf("chat_model is required")
    }
    if c.StreamModel == "" {
        return fmt.Errorf("stream_model is required")
    }
    if c.Timeout <= 0 {
        return fmt.Errorf("timeout must be positive")
    }
    if c.MaxRetries < 1 {
        return fmt.Errorf("max_retries must be at least 1")
    }
    return nil
}

func DefaultConfig() *Config {
    return &Config{
        RetrievalTopK:    8,
        ContextMaxTokens: 4000,
        ChatModel:        "jabir-400b",
        StreamModel:      "jabir-400b",
        Timeout:          120 * time.Second,
        MaxRetries:       3,
        Temperature:      0.1, // Low for medical accuracy
        MaxTokens:        2000,
        EnableSources:    true,
        MaxSources:       10,
    }
}
