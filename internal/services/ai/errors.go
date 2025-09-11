// G:\go_internist\internal\services\ai\errors.go
package ai

import "fmt"

type ErrorType string

const (
    ErrTypeConfig       ErrorType = "CONFIG"
    ErrTypeNetwork      ErrorType = "NETWORK"  
    ErrTypeProvider     ErrorType = "PROVIDER"
    ErrTypeRateLimit    ErrorType = "RATE_LIMIT"
    ErrTypeQuota        ErrorType = "QUOTA"
    ErrTypeModel        ErrorType = "MODEL"
    ErrTypeValidation   ErrorType = "VALIDATION"
)

type AIError struct {
    Type       ErrorType
    Code       int
    Message    string
    Model      string
    Operation  string
    Cause      error
}

func (e *AIError) Error() string {
    if e.Cause != nil {
        return fmt.Sprintf("AI %s error in %s: %s (caused by: %v)", 
            e.Type, e.Operation, e.Message, e.Cause)
    }
    return fmt.Sprintf("AI %s error in %s: %s", e.Type, e.Operation, e.Message)
}

func NewConfigError(msg string) *AIError {
    return &AIError{Type: ErrTypeConfig, Message: msg, Operation: "config"}
}

func NewProviderError(operation, msg string, cause error) *AIError {
    return &AIError{Type: ErrTypeProvider, Operation: operation, Message: msg, Cause: cause}
}
