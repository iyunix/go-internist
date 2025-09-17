// G:\go_internist\internal\services\pinecone\retry.go
package pinecone

import (
    "context"
    "time"
)

// RetryService holds configuration and logger for retry operations
type RetryService struct {
    config *Config
    logger Logger
}

// NewRetryService creates a new RetryService with given config and logger
func NewRetryService(config *Config, logger Logger) *RetryService {
    return &RetryService{
        config: config,
        logger: logger,
    }
}

// RetryWithTimeout retries a given call up to MaxRetries, each with Timeout,
// propagating the parent context (cancellation & deadline)
func (r *RetryService) RetryWithTimeout(
    parentCtx context.Context,
    call func(ctx context.Context) error,
) error {
    var lastErr error

    for attempt := 1; attempt <= r.config.MaxRetries; attempt++ {
        // Use parentCtx so deadlines/cancellations propagate
        ctx, cancel := context.WithTimeout(parentCtx, r.config.Timeout)

        r.logger.Debug("attempting Pinecone operation",
            "attempt", attempt,
            "max_attempts", r.config.MaxRetries,
            "timeout", r.config.Timeout.String())

        err := call(ctx)
        cancel()

        if err == nil {
            if attempt > 1 {
                r.logger.Info("Pinecone operation succeeded after retry",
                    "attempt", attempt,
                    "total_attempts", r.config.MaxRetries)
            }
            return nil
        }

        lastErr = err
        r.logger.Warn("Pinecone operation failed",
            "attempt", attempt,
            "max_attempts", r.config.MaxRetries,
            "error", err)

        if attempt < r.config.MaxRetries {
            backoffDuration := time.Duration(attempt) * r.config.RetryDelay
            r.logger.Debug("backing off before retry",
                "backoff_duration", backoffDuration.String())
            time.Sleep(backoffDuration)
        }
    }

    r.logger.Error("Pinecone operation failed after all retries",
        "total_attempts", r.config.MaxRetries,
        "final_error", lastErr)

    return NewRetryError("retry_exhausted",
        "operation failed after all retries", lastErr)
}

// NewRetryError constructs a PineconeError for retry exhaustion
func NewRetryError(operation, msg string, cause error) *PineconeError {
    return &PineconeError{
        Type:      ErrTypeRetry,
        Operation: operation,
        Message:   msg,
        Cause:     cause,
    }
}
