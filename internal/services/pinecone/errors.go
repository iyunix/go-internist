// G:\go_internist\internal\services\pinecone\errors.go
package pinecone

import "fmt"

type ErrorType string

const (
    ErrTypeConfig     ErrorType = "CONFIG"
    ErrTypeAuth       ErrorType = "AUTH"
    ErrTypeConnection ErrorType = "CONNECTION"
    ErrTypeVector     ErrorType = "VECTOR"
    ErrTypeQuery      ErrorType = "QUERY"
    ErrTypeRetry      ErrorType = "RETRY"
    ErrTypeQuota      ErrorType = "QUOTA"
    ErrTypeValidation ErrorType = "VALIDATION"
)

type PineconeError struct {
    Type      ErrorType
    Operation string
    Message   string
    Index     string
    Namespace string
    VectorID  string
    Cause     error
}

func (e *PineconeError) Error() string {
    if e.Cause != nil {
        return fmt.Sprintf("Pinecone %s error in %s: %s (caused by: %v)", 
            e.Type, e.Operation, e.Message, e.Cause)
    }
    return fmt.Sprintf("Pinecone %s error in %s: %s", e.Type, e.Operation, e.Message)
}

func NewConfigError(msg string) *PineconeError {
    return &PineconeError{Type: ErrTypeConfig, Operation: "config", Message: msg}
}

func NewConnectionError(operation, msg string, cause error) *PineconeError {
    return &PineconeError{Type: ErrTypeConnection, Operation: operation, Message: msg, Cause: cause}
}

func NewVectorError(operation, vectorID, msg string, cause error) *PineconeError {
    return &PineconeError{Type: ErrTypeVector, Operation: operation, VectorID: vectorID, Message: msg, Cause: cause}
}

func NewQueryError(operation, msg string, cause error) *PineconeError {
    return &PineconeError{Type: ErrTypeQuery, Operation: operation, Message: msg, Cause: cause}
}
