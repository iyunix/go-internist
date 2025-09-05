// File: internal/repository/interface.go
package repository

import (
	"context"

	"github.com/iyunix/go-internist/internal/domain"
)

// UserRepository handles user data operations.
// CHANGED: This interface is now updated to support the full verification and lockout flow.
type UserRepository interface {
	Create(ctx context.Context, user *domain.User) (*domain.User, error)
	FindByID(ctx context.Context, id uint) (*domain.User, error)
	FindByUsername(ctx context.Context, username string) (*domain.User, error)
	
	// NEW: Update saves changes to a user model.
	Update(ctx context.Context, user *domain.User) error
	
	// NEW: FindByUsernameOrPhone checks for existing users during registration.
	FindByUsernameOrPhone(ctx context.Context, username, phone string) (*domain.User, error)
	
	// NEW: FindByPhoneAndStatus finds a user to verify or resend a code to.
	FindByPhoneAndStatus(ctx context.Context, phone string, status domain.UserStatus) (*domain.User, error)
	
	// NEW: ResetFailedAttempts clears lockout fields after a successful login.
	ResetFailedAttempts(ctx context.Context, id uint) error
}

// REMOVED: The VerificationCodeRepository is no longer needed
// as this logic is now handled directly within the User model and UserRepository.

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