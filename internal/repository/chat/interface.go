package chat

import (
    "context"

    "github.com/iyunix/go-internist/internal/domain"
)

// ChatRepository handles chat data operations.
type ChatRepository interface {
    Create(ctx context.Context, chat *domain.Chat) (*domain.Chat, error)
    FindByID(ctx context.Context, id uint) (*domain.Chat, error)
    FindByUserID(ctx context.Context, userID uint) ([]domain.Chat, error)
    Delete(ctx context.Context, chatID uint, userID uint) error
    TouchUpdatedAt(ctx context.Context, chatID uint) error
}
