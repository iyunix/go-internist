// G:\go_internist\internal\services\pinecone\config.go
package pinecone

import (
    "fmt"
    "time"
)

type Config struct {
    // Authentication
    APIKey    string
    
    // Connection
    IndexHost string
    Namespace string
    
    // Performance
    Timeout    time.Duration
    MaxRetries int
    RetryDelay time.Duration
    
    // Vector Operations
    BatchSize     int
    IncludeValues bool
    TopKLimit     int
}

func (c *Config) Validate() error {
    if c.APIKey == "" {
        return fmt.Errorf("PINECONE_API_KEY is required")
    }
    if c.IndexHost == "" {
        return fmt.Errorf("PINECONE_INDEX_HOST is required")
    }
    if c.Namespace == "" {
        return fmt.Errorf("PINECONE_NAMESPACE is required")
    }
    if c.Timeout <= 0 {
        return fmt.Errorf("timeout must be positive")
    }
    if c.MaxRetries < 1 {
        return fmt.Errorf("max_retries must be at least 1")
    }
    if c.BatchSize <= 0 {
        return fmt.Errorf("batch_size must be positive")
    }
    return nil
}

func DefaultConfig() *Config {
    return &Config{
        Timeout:       20 * time.Second,
        MaxRetries:    3,
        RetryDelay:    time.Second,
        BatchSize:     100,
        IncludeValues: false,
        TopKLimit:     50,
    }
}
