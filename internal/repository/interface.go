// File: internal/repository/interfaces.go
package repository

import (
	"context"

	"github.com/iyunix/go-internist/internal/domain"
)

// UserRepository handles user data operations.
type UserRepository interface {
	Create(ctx context.Context, user *domain.User) (*domain.User, error)
	FindByID(ctx context.Context, id uint) (*domain.User, error)
	FindByUsername(ctx context.Context, username string) (*domain.User, error)
}

// ChatRepository handles chat data operations.
type ChatRepository interface {
	Create(ctx context.Context, chat *domain.Chat) (*domain.Chat, error)
	FindByID(ctx context.Context, id uint) (*domain.Chat, error)
	FindByUserID(ctx context.Context, userID uint) ([]domain.Chat, error)
	Delete(ctx context.Context, chatID uint, userID uint) error

	// TouchUpdatedAt bumps the chat's updated_at to the current time to reflect activity.
	TouchUpdatedAt(ctx context.Context, chatID uint) error
}

// MessageRepository handles message data operations.
type MessageRepository interface {
	Create(ctx context.Context, message *domain.Message) (*domain.Message, error)
	FindByChatID(ctx context.Context, chatID uint) ([]domain.Message, error)
}
