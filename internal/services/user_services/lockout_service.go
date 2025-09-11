package user_services

import (
    "context"
    "fmt"
    "time"

    "github.com/iyunix/go-internist/internal/repository/user"
)

const (
    MaxFailedAttempts = 5
    LockoutDuration   = 15 * time.Minute
)

// LockoutService handles account security and brute force protection
type LockoutService struct {
    userRepo user.UserRepository
    logger   Logger
}

// NewLockoutService creates a new lockout service
func NewLockoutService(userRepo user.UserRepository, logger Logger) *LockoutService {
    return &LockoutService{
        userRepo: userRepo,
        logger:   logger,
    }
}

// RecordFailedAttempt records a failed login attempt and may lock the account
func (s *LockoutService) RecordFailedAttempt(ctx context.Context, phone string, sourceIP string) error {
    if phone == "" {
        s.logger.Warn("failed attempt recorded with empty phone number")
        return fmt.Errorf("phone number is required")
    }

    s.logger.Warn("recording failed login attempt", 
        "phone", phone[:min(4, len(phone))]+"****",
        "source_ip", sourceIP,
        "max_attempts", MaxFailedAttempts)

    user, err := s.userRepo.FindByPhone(ctx, phone)
    if err != nil {
        s.logger.Error("failed to find user for failed attempt recording", 
            "error", err,
            "phone", phone[:min(4, len(phone))]+"****",
            "source_ip", sourceIP)
        return fmt.Errorf("failed to find user: %w", err)
    }

    // Increment failed attempts
    user.FailedLoginAttempts++
    now := time.Now()
    user.LastFailedLoginAt = &now

    s.logger.Warn("failed login attempt recorded", 
        "user_id", user.ID,
        "phone", phone[:min(4, len(phone))]+"****",
        "attempts", user.FailedLoginAttempts,
        "max_attempts", MaxFailedAttempts,
        "source_ip", sourceIP)

    // Check if account should be locked
    if user.FailedLoginAttempts >= MaxFailedAttempts {
        lockUntil := time.Now().Add(LockoutDuration)
        user.LockedUntil = &lockUntil

        s.logger.Error("account locked due to excessive failed attempts", 
            "user_id", user.ID,
            "phone", phone[:min(4, len(phone))]+"****",
            "attempts", user.FailedLoginAttempts,
            "locked_until", lockUntil.Format(time.RFC3339),
            "lockout_duration", LockoutDuration.String(),
            "source_ip", sourceIP)
    }

    if err := s.userRepo.Update(ctx, user); err != nil {
        s.logger.Error("failed to update failed login attempts", 
            "error", err,
            "user_id", user.ID,
            "phone", phone[:min(4, len(phone))]+"****")
        return fmt.Errorf("failed to update user: %w", err)
    }

    return nil
}

// ClearFailedAttempts clears failed login attempts after successful login
func (s *LockoutService) ClearFailedAttempts(ctx context.Context, userID uint) error {
    if userID == 0 {
        s.logger.Warn("clear failed attempts attempted with invalid user ID", "user_id", userID)
        return fmt.Errorf("user ID is required")
    }

    s.logger.Info("clearing failed login attempts", "user_id", userID)

    user, err := s.userRepo.FindByID(ctx, userID)
    if err != nil {
        s.logger.Error("failed to find user for clearing attempts", 
            "error", err,
            "user_id", userID)
        return fmt.Errorf("failed to find user: %w", err)
    }

    oldAttempts := user.FailedLoginAttempts
    wasLocked := user.LockedUntil != nil && time.Now().Before(*user.LockedUntil)

    // Clear failed attempts and lockout
    user.FailedLoginAttempts = 0
    user.LastFailedLoginAt = nil
    user.LockedUntil = nil

    if err := s.userRepo.Update(ctx, user); err != nil {
        s.logger.Error("failed to clear failed attempts", 
            "error", err,
            "user_id", userID)
        return fmt.Errorf("failed to update user: %w", err)
    }

    s.logger.Info("failed login attempts cleared", 
        "user_id", userID,
        "phone", user.PhoneNumber[:min(4, len(user.PhoneNumber))]+"****",
        "previous_attempts", oldAttempts,
        "was_locked", wasLocked)

    return nil
}

// IsAccountLocked checks if an account is currently locked
func (s *LockoutService) IsAccountLocked(ctx context.Context, phone string) (bool, *LockoutStatus, error) {
    if phone == "" {
        s.logger.Warn("lockout check attempted with empty phone number")
        return false, nil, fmt.Errorf("phone number is required")
    }

    user, err := s.userRepo.FindByPhone(ctx, phone)
    if err != nil {
        s.logger.Error("failed to find user for lockout check", 
            "error", err,
            "phone", phone[:min(4, len(phone))]+"****")
        return false, nil, fmt.Errorf("failed to find user: %w", err)
    }

    now := time.Now()
    isLocked := user.LockedUntil != nil && now.Before(*user.LockedUntil)

    status := &LockoutStatus{
        UserID:         user.ID,
        Phone:          phone[:min(4, len(phone))] + "****",
        IsLocked:       isLocked,
        FailedAttempts: user.FailedLoginAttempts,
        MaxAttempts:    MaxFailedAttempts,
    }

    if user.LockedUntil != nil {
        status.LockedUntil = user.LockedUntil
        if isLocked {
            status.TimeRemaining = user.LockedUntil.Sub(now)
        }
    }

    if user.LastFailedLoginAt != nil {
        status.LastFailedAt = user.LastFailedLoginAt
    }

    return isLocked, status, nil
}

// LockoutStatus represents the current lockout state of an account
type LockoutStatus struct {
    UserID         uint           `json:"user_id"`
    Phone          string         `json:"phone"`
    IsLocked       bool           `json:"is_locked"`
    FailedAttempts int            `json:"failed_attempts"`
    MaxAttempts    int            `json:"max_attempts"`
    LockedUntil    *time.Time     `json:"locked_until,omitempty"`
    LastFailedAt   *time.Time     `json:"last_failed_at,omitempty"`
    TimeRemaining  time.Duration  `json:"time_remaining,omitempty"`
}

// REMOVED: min function (now in types.go)
// REMOVED: unused domain import
