package message

import (
    "context"

    "github.com/iyunix/go-internist/internal/domain"
)

// MessageRepository handles message data operations.
type MessageRepository interface {
    Create(ctx context.Context, message *domain.Message) (*domain.Message, error)
    FindByChatID(ctx context.Context, chatID uint) ([]domain.Message, error)
}
