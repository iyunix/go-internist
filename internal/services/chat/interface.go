// G:\go_internist\internal\services\chat\interface.go
package chat

import (
    "context"
    "github.com/iyunix/go-internist/internal/domain"
    "github.com/pinecone-io/go-pinecone/v4/pinecone"
)

// RAGProvider handles retrieval-augmented generation
type RAGProvider interface {
    BuildContext(matches []*pinecone.ScoredVector) string
    BuildPrompt(context, question string) string
    ExtractSources(matches []*pinecone.ScoredVector) []string
}

// StreamProvider handles chat streaming
type StreamProvider interface {
    StreamChatResponse(ctx context.Context, chatID, userID uint, prompt string, 
        onDelta func(string) error, onSources func([]string)) error
}

// ChatProvider handles basic chat operations
type ChatProvider interface {
    CreateChat(ctx context.Context, userID uint, title string) (*domain.Chat, error)
    GetUserChats(ctx context.Context, userID uint) ([]domain.Chat, error)
    GetChatMessages(ctx context.Context, userID, chatID uint) ([]domain.Message, error)
    DeleteChat(ctx context.Context, userID, chatID uint) error
}

// Service combines all chat capabilities
type Service interface {
    ChatProvider
    StreamProvider
    RAGProvider
    HealthCheck(ctx context.Context) error
}

// ServiceStatus represents chat service health
type ServiceStatus struct {
    IsHealthy       bool
    RAGHealthy      bool
    StreamHealthy   bool
    DatabaseHealthy bool
    Message         string
}
