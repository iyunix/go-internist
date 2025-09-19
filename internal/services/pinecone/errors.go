// G:\go_internist\internal\services\pinecone\errors.go
package pinecone

import (
    "fmt"
)

// QdrantError represents a Qdrant-specific error (keeping compatible with existing error handling)
type QdrantError struct {
    Type    string
    Message string
    Err     error
}

func (e *QdrantError) Error() string {
    if e.Err != nil {
        return fmt.Sprintf("qdrant %s error: %s: %v", e.Type, e.Message, e.Err)
    }
    return fmt.Sprintf("qdrant %s error: %s", e.Type, e.Message)
}

func (e *QdrantError) Unwrap() error {
    return e.Err
}

// Error constructors (keeping same names for compatibility)
func NewConnectionError(errorType, message string, err error) *QdrantError {
    return &QdrantError{
        Type:    errorType,
        Message: message,
        Err:     err,
    }
}

func NewOperationError(message string, err error) *QdrantError {
    return &QdrantError{
        Type:    "operation",
        Message: message,
        Err:     err,
    }
}

func NewConfigError(message string) *QdrantError {
    return &QdrantError{
        Type:    "config",
        Message: message,
        Err:     nil,
    }
}

func NewTimeoutError(message string, err error) *QdrantError {
    return &QdrantError{
        Type:    "timeout",
        Message: message,
        Err:     err,
    }
}

func NewRetryError(message string, err error) *QdrantError {
    return &QdrantError{
        Type:    "retry",
        Message: message,
        Err:     err,
    }
}
