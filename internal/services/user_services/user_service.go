// File: internal/services/user_services/user_service.go
package user_services

import (
	"context"
	"time"

	"github.com/iyunix/go-internist/internal/domain"
	"github.com/iyunix/go-internist/internal/repository"
)

// UserService is the main service that composes other user-related services
type UserService struct {
	*VerificationService
	*AuthService
	*BalanceService
	*LockoutService
}

// NewUserService creates a new composite UserService
// <-- 1. UPDATED the function to accept adminPhoneNumber
func NewUserService(userRepo repository.UserRepository, jwtSecret string, adminPhoneNumber string) *UserService {
	lockoutService := NewLockoutService(userRepo)
	authService := NewAuthService(userRepo, jwtSecret, lockoutService)
	// <-- 2. PASS the variable down to the VerificationService constructor
	verificationService := NewVerificationService(userRepo, adminPhoneNumber)
	balanceService := NewBalanceService(userRepo)

	return &UserService{
		VerificationService: verificationService,
		AuthService:         authService,
		BalanceService:      balanceService,
		LockoutService:      lockoutService,
	}
}

// UserServiceInterface defines the complete interface for user operations
type UserServiceInterface interface {
	// Verification methods
	InitiateVerification(ctx context.Context, user *domain.User, code string, ttl time.Duration) error
	FinalizeVerification(ctx context.Context, phone, code string) (*domain.User, error)
	ResendVerificationCode(ctx context.Context, phone, newCode string, ttl time.Duration) error

	// Authentication methods
	Login(ctx context.Context, username, password string) (string, error)

	// Balance methods
	GetCharacterBalance(ctx context.Context, userID uint) (int, error)
	CanUserAskQuestion(ctx context.Context, userID uint, questionLength int) (bool, int, error)
	DeductCharactersForQuestion(ctx context.Context, userID uint, questionLength int) (int, error)
	CalculateChargePreview(questionLength int) int
	AddCharacters(ctx context.Context, userID uint, amount int) error
}