// File: internal/repository/interface.go
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
	Update(ctx context.Context, user *domain.User) error
	FindByUsernameOrPhone(ctx context.Context, username, phone string) (*domain.User, error)
	FindByPhoneAndStatus(ctx context.Context, phone string, status domain.UserStatus) (*domain.User, error)
    FindByPhone(ctx context.Context, phone string) (*domain.User, error)  // ADD THIS
	ResetFailedAttempts(ctx context.Context, id uint) error
	Delete(ctx context.Context, userID uint) error  // ADD THIS
	// Character balance methods
	GetCharacterBalance(ctx context.Context, userID uint) (int, error)
	UpdateCharacterBalance(ctx context.Context, userID uint, newBalance int) error

	// <-- ADD THIS METHOD
	// FindAll retrieves all users, which is necessary for the admin panel.
	FindAll(ctx context.Context) ([]domain.User, error)
}

// ChatRepository handles chat data operations.
type ChatRepository interface {
	Create(ctx context.Context, chat *domain.Chat) (*domain.Chat, error)
	FindByID(ctx context.Context, id uint) (*domain.Chat, error)
	FindByUserID(ctx context.Context, userID uint) ([]domain.Chat, error)
	Delete(ctx context.Context, chatID uint, userID uint) error
	TouchUpdatedAt(ctx context.Context, chatID uint) error
}

// MessageRepository handles message data operations.
type MessageRepository interface {
	Create(ctx context.Context, message *domain.Message) (*domain.Message, error)
	FindByChatID(ctx context.Context, chatID uint) ([]domain.Message, error)
}