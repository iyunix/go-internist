// File: internal/services/ai/config.go
package ai

import (
    "fmt"
    "time"
)

type Config struct {
    // Embedding Configuration
    EmbeddingKey     string
    EmbeddingBaseURL string
    EmbeddingModel   string
    
    // LLM Configuration  
    LLMKey           string
    LLMBaseURL       string
    
    // Performance Configuration - ✅ FIXED: Longer timeouts
    Timeout          time.Duration
    MaxRetries       int
    RetryDelay       time.Duration
    
    // Model Parameters
    Temperature      float32
    TopP             float32
}

func (c *Config) Validate() error {
    if c.EmbeddingKey == "" {
        return fmt.Errorf("AI_EMBEDDING_KEY is required")
    }
    if c.LLMKey == "" {
        return fmt.Errorf("AI_LLM_KEY is required")
    }
    if c.EmbeddingModel == "" {
        return fmt.Errorf("AI_EMBEDDING_MODEL is required")
    }
    if c.Timeout <= 0 {
        return fmt.Errorf("timeout must be positive")
    }
    if c.MaxRetries < 1 {
        return fmt.Errorf("max retries must be at least 1")
    }
    return nil
}

// ✅ FIXED: Increased timeout values for production
func DefaultConfig() *Config {
    return &Config{
        Timeout:     5 * time.Minute,  // Increased from 60s
        MaxRetries:  3,
        RetryDelay:  2 * time.Second,  // Increased from 1s
        Temperature: 0.1,
        TopP:        0.9,
    }
}
