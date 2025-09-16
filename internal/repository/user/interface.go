// File: internal/repository/user/interface.go
package user

import (
    "context"
    "github.com/iyunix/go-internist/internal/domain"
)

// UserRepository handles user data operations.
type UserRepository interface {
    // ===== EXISTING METHODS (unchanged) =====
    Create(ctx context.Context, user *domain.User) (*domain.User, error)
    FindByID(ctx context.Context, id uint) (*domain.User, error)
    FindByUsername(ctx context.Context, username string) (*domain.User, error)
    Update(ctx context.Context, user *domain.User) error
    FindByUsernameOrPhone(ctx context.Context, username, phone string) (*domain.User, error)
    FindByPhoneAndStatus(ctx context.Context, phone string, status domain.UserStatus) (*domain.User, error)
    FindByPhone(ctx context.Context, phone string) (*domain.User, error)
    FindByPhoneNumber(ctx context.Context, phoneNumber string) (*domain.User, error) // ADD THIS LINE
    ResetFailedAttempts(ctx context.Context, id uint) error
    Delete(ctx context.Context, userID uint) error
    GetCharacterBalance(ctx context.Context, userID uint) (int, error)
    UpdateCharacterBalance(ctx context.Context, userID uint, newBalance int) error
    FindAll(ctx context.Context) ([]domain.User, error)

    // ===== NEW PRODUCTION-READY METHODS =====
    FindAllWithPagination(ctx context.Context, limit, offset int) ([]domain.User, int64, error)
    CreateInBatch(ctx context.Context, users []*domain.User, batchSize int) error
    ExistsByUsername(ctx context.Context, username string) (bool, error)
    ExistsByPhone(ctx context.Context, phone string) (bool, error)
    CountUsers(ctx context.Context) (int64, error)
    CountActiveUsers(ctx context.Context) (int64, error)
    IncrementFailedAttempts(ctx context.Context, userID uint) error
    UpdateMultipleBalances(ctx context.Context, updates []domain.BalanceUpdate) error
    FindAllWithPaginationAndSearch(ctx context.Context, page, limit int, search string) ([]domain.User, int64, error)
}
