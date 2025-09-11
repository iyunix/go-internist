// G:\go_internist\internal\services\chat\errors.go
package chat

import "fmt"

type ErrorType string

const (
    ErrTypeConfig      ErrorType = "CONFIG"
    ErrTypeValidation  ErrorType = "VALIDATION"
    ErrTypeRAG         ErrorType = "RAG"
    ErrTypeStreaming   ErrorType = "STREAMING"
    ErrTypeContext     ErrorType = "CONTEXT"
    ErrTypeEmbedding   ErrorType = "EMBEDDING"
    ErrTypePinecone    ErrorType = "PINECONE"
    ErrTypeUnauthorized ErrorType = "UNAUTHORIZED"
    ErrTypeNotFound    ErrorType = "NOT_FOUND"
)

type ChatError struct {
    Type      ErrorType
    Operation string
    Message   string
    ChatID    uint
    UserID    uint
    Cause     error
}

func (e *ChatError) Error() string {
    if e.Cause != nil {
        return fmt.Sprintf("Chat %s error in %s: %s (caused by: %v)", 
            e.Type, e.Operation, e.Message, e.Cause)
    }
    return fmt.Sprintf("Chat %s error in %s: %s", e.Type, e.Operation, e.Message)
}

func NewValidationError(operation, msg string) *ChatError {
    return &ChatError{Type: ErrTypeValidation, Operation: operation, Message: msg}
}

func NewRAGError(operation, msg string, cause error) *ChatError {
    return &ChatError{Type: ErrTypeRAG, Operation: operation, Message: msg, Cause: cause}
}

func NewUnauthorizedError(userID, chatID uint) *ChatError {
    return &ChatError{
        Type: ErrTypeUnauthorized, 
        Operation: "authorization", 
        Message: "chat not found or unauthorized",
        UserID: userID,
        ChatID: chatID,
    }
}
