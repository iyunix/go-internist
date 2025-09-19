// G:\go_internist\internal\services\pinecone\retry.go
package pinecone

import (
    "context"
    "time"
)

type RetryService struct {
    config *Config
    logger Logger
}

func NewRetryService(config *Config, logger Logger) *RetryService {
    return &RetryService{
        config: config,
        logger: logger,
    }
}

func (r *RetryService) RetryWithTimeout(call func(ctx context.Context) error) error {
    ctx, cancel := context.WithTimeout(context.Background(), r.config.Timeout)
    defer cancel()
    
    var lastErr error
    
    for attempt := 0; attempt <= r.config.MaxRetries; attempt++ {
        if attempt > 0 {
            r.logger.Debug("retrying operation", "attempt", attempt, "max_retries", r.config.MaxRetries)
            select {
            case <-ctx.Done():
                return NewTimeoutError("operation timed out during retry", ctx.Err())
            case <-time.After(r.config.RetryDelay):
                // Continue with retry
            }
        }
        
        err := call(ctx)
        if err == nil {
            if attempt > 0 {
                r.logger.Info("operation succeeded after retry", "attempts", attempt+1)
            }
            return nil
        }
        
        lastErr = err
        
        // Don't retry on context cancellation or certain errors
        if ctx.Err() != nil {
            return NewTimeoutError("operation timed out", ctx.Err())
        }
        
        if attempt < r.config.MaxRetries {
            r.logger.Warn("operation failed, retrying", "attempt", attempt+1, "error", err)
        }
    }
    
    r.logger.Error("operation failed after all retries", "attempts", r.config.MaxRetries+1, "error", lastErr)
    return NewRetryError("operation failed after all retries", lastErr)
}
