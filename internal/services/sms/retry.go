// G:\go_internist\internal\services\sms\retry.go
package sms

import (
    "context"
    "time"
)

// RetryConfig defines simple retry behavior
type RetryConfig struct {
    MaxAttempts int
    Delay       time.Duration
}

// DefaultRetryConfig provides sensible defaults
func DefaultRetryConfig() *RetryConfig {
    return &RetryConfig{
        MaxAttempts: 3,
        Delay:       500 * time.Millisecond,
    }
}

// RetryWithBackoff executes a function with simple retry logic
func RetryWithBackoff(ctx context.Context, config *RetryConfig, fn func(ctx context.Context) error) error {
    var lastErr error
    
    for attempt := 0; attempt < config.MaxAttempts; attempt++ {
        err := fn(ctx)
        if err == nil {
            return nil
        }
        
        lastErr = err
        
        // Don't retry non-retryable errors
        if smsErr, ok := err.(*SMSError); ok {
            if smsErr.Type == ErrTypeConfig || smsErr.Type == ErrTypeValidation {
                return err
            }
        }
        
        // Don't wait after last attempt
        if attempt < config.MaxAttempts-1 {
            select {
            case <-ctx.Done():
                return ctx.Err()
            case <-time.After(config.Delay):
            }
        }
    }
    
    return lastErr
}
