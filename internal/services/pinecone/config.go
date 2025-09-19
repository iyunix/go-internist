// G:\go_internist\internal\services\pinecone\config.go
package pinecone

import (
    "errors"
    "time"
)

type Config struct {
    // Connection settings (using existing field names for compatibility)
    APIKey    string        // Qdrant API Key
    IndexHost string        // Qdrant URL 
    Namespace string        // Qdrant Collection Name
    
    // Operation settings
    Timeout        time.Duration
    MaxRetries     int
    RetryDelay     time.Duration
    
    // Performance settings
    BatchSize      int
    PoolSize       int
}

func DefaultConfig() *Config {
    return &Config{
        Timeout:        30 * time.Second,
        MaxRetries:     3,
        RetryDelay:     2 * time.Second,
        BatchSize:      100,
        PoolSize:       10,
    }
}

func (c *Config) Validate() error {
    if c.IndexHost == "" {
        return errors.New("qdrant URL is required")
    }
    
    if c.APIKey == "" {
        return errors.New("qdrant API key is required")
    }
    
    if c.Namespace == "" {
        return errors.New("qdrant collection name is required")
    }
    
    if c.Timeout <= 0 {
        return errors.New("timeout must be positive")
    }
    
    if c.MaxRetries < 0 {
        return errors.New("max retries cannot be negative")
    }
    
    return nil
}

func (c *Config) GetTimeoutSeconds() uint64 {
    return uint64(c.Timeout.Seconds())
}
