// File: internal/services/user_services/auth_service.go
package user_services

import (
    "context"
    "errors"
    "log"

    "golang.org/x/crypto/bcrypt"

    "github.com/iyunix/go-internist/internal/auth"
    "github.com/iyunix/go-internist/internal/domain"
    "github.com/iyunix/go-internist/internal/repository"
)

type AuthService struct {
    userRepo       repository.UserRepository
    jwtSecret      string
    lockoutService *LockoutService
}

func NewAuthService(userRepo repository.UserRepository, jwtSecret string, lockoutService *LockoutService) *AuthService {
    return &AuthService{
        userRepo:       userRepo,
        jwtSecret:      jwtSecret,
        lockoutService: lockoutService,
    }
}

// Login authenticates a user and returns a JWT token.
func (s *AuthService) Login(ctx context.Context, username, password string) (string, error) {
    // 1. Find user by username
    user, err := s.userRepo.FindByUsername(ctx, username)
    if err != nil {
        // Note: We don't record a failed attempt here because the user might not exist.
        // The error is generic to prevent username enumeration attacks.
        return "", errors.New("invalid username or password")
    }

    // 2. Check for account lockout
    if s.lockoutService.IsUserLockedOut(user) {
        return "", errors.New("account temporarily locked due to multiple failed login attempts")
    }
    
    // 3. Ensure only active users can log in
    if user.Status != domain.UserStatusActive {
        return "", errors.New("account is not active")
    }

    // 4. Verify password
    if err := s.verifyPassword(user.Password, password); err != nil {
        // Password does not match, record the failed attempt
        s.lockoutService.RecordFailedLogin(ctx, user)
        return "", errors.New("invalid username or password")
    }

    // 5. On success, reset failed login attempts
    if err := s.lockoutService.ResetFailedAttempts(ctx, user.ID); err != nil {
        log.Printf("Could not reset failed login attempts for user %d: %v", user.ID, err)
        // We can still let the login proceed, but we log the error.
    }

    // 6. Generate JWT token
    token, err := s.generateToken(user.ID, username)
    if err != nil {
        return "", err
    }

    return token, nil
}

// Private helper methods
func (s *AuthService) verifyPassword(hashedPassword, plainPassword string) error {
    return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(plainPassword))
}

func (s *AuthService) generateToken(userID uint, username string) (string, error) {
    token, err := auth.GenerateJWT(userID, []byte(s.jwtSecret))
    if err != nil {
        log.Printf("JWT generation error for user %s: %v", username, err)
        return "", errors.New("authentication error")
    }
    return token, nil
}
