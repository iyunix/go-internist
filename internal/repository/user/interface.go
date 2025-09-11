package user

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
    FindByPhone(ctx context.Context, phone string) (*domain.User, error)
    ResetFailedAttempts(ctx context.Context, id uint) error
    Delete(ctx context.Context, userID uint) error
    GetCharacterBalance(ctx context.Context, userID uint) (int, error)
    UpdateCharacterBalance(ctx context.Context, userID uint, newBalance int) error
    FindAll(ctx context.Context) ([]domain.User, error)
}
