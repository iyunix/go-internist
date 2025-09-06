// File: internal/services/user_services/lockout_service.go
package user_services

import (
    "context"
    "log"
    "time"

    "github.com/iyunix/go-internist/internal/domain"
    "github.com/iyunix/go-internist/internal/repository"
)

// Lockout settings
const (
    MaxFailedAttempts = 5
    LockoutDuration   = 15 * time.Minute
)

type LockoutService struct {
    userRepo repository.UserRepository
}

func NewLockoutService(userRepo repository.UserRepository) *LockoutService {
    return &LockoutService{
        userRepo: userRepo,
    }
}

// IsUserLockedOut checks if a user is currently locked out
func (s *LockoutService) IsUserLockedOut(user *domain.User) bool {
    return user.LockedUntil.After(time.Now())
}

// RecordFailedLogin records a failed login attempt and potentially locks the account
func (s *LockoutService) RecordFailedLogin(ctx context.Context, user *domain.User) {
    user.FailedLoginAttempts++
    
    if user.FailedLoginAttempts >= MaxFailedAttempts {
        user.LockedUntil = time.Now().Add(LockoutDuration)
        log.Printf("User %s locked out until %v due to failed logins", user.Username, user.LockedUntil)
    }

    if err := s.userRepo.Update(ctx, user); err != nil {
        log.Printf("Failed to record failed login for user %s: %v", user.Username, err)
    }
}

// ResetFailedAttempts resets the failed login attempts for a user
func (s *LockoutService) ResetFailedAttempts(ctx context.Context, userID uint) error {
    return s.userRepo.ResetFailedAttempts(ctx, userID)
}

// GetLockoutInfo returns lockout information for a user
func (s *LockoutService) GetLockoutInfo(user *domain.User) (bool, time.Time, int) {
    isLocked := s.IsUserLockedOut(user)
    return isLocked, user.LockedUntil, user.FailedLoginAttempts
}
