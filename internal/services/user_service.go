// File: internal/services/user_service.go
package services

import (
	"context"
	"errors"
	"fmt" // NEW: For more detailed errors
	"log"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/iyunix/go-internist/internal/auth"
	"github.com/iyunix/go-internist/internal/domain"
	"github.com/iyunix/go-internist/internal/repository"
)

// Lockout settings
const (
	maxFailedAttempts = 5
	lockoutDuration   = 15 * time.Minute
)

// UserService is responsible for all user-related business logic.
// CHANGED: This struct is now stateless. The sync.Mutex and failedLogins map have been removed.
type UserService struct {
	userRepo  repository.UserRepository
	jwtSecret string
}

// NewUserService creates a new, stateless UserService.
func NewUserService(userRepo repository.UserRepository, jwtSecret string) *UserService {
	return &UserService{
		userRepo:  userRepo,
		jwtSecret: jwtSecret,
	}
}

// NEW: InitiateVerification handles the first step of registration.
// It validates the user, hashes their password, and stores them in a pending state with a verification code.
func (s *UserService) InitiateVerification(ctx context.Context, user *domain.User, code string, ttl time.Duration) error {
	// 1. Check if user already exists (by username or phone number)
	existingUser, err := s.userRepo.FindByUsernameOrPhone(ctx, user.Username, user.PhoneNumber)
	if err != nil && !errors.Is(err, repository.ErrUserNotFound) {
		return fmt.Errorf("database error while checking for existing user: %w", err)
	}
	if existingUser != nil {
		return errors.New("a user with that username or phone number already exists")
	}

	// 2. Hash the password
	hashedPwd, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Password hashing error: %v", err)
		return errors.New("internal error occurred during registration")
	}
	user.Password = string(hashedPwd)

	// 3. Set user status to pending and add verification details
	user.Status = domain.UserStatusPending
	user.VerificationCode = code
	user.VerificationExpiresAt = time.Now().Add(ttl)

	// 4. Create the pending user record in the database
	_, err = s.userRepo.Create(ctx, user)
	if err != nil {
		return fmt.Errorf("failed to create pending user: %w", err)
	}

	return nil
}

// NEW: FinalizeVerification completes the registration process.
// It finds a pending user, validates their code, and activates their account.
func (s *UserService) FinalizeVerification(ctx context.Context, phone, code string) (*domain.User, error) {
	// 1. Find the user by phone number, but only if they are pending
	user, err := s.userRepo.FindByPhoneAndStatus(ctx, phone, domain.UserStatusPending)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, errors.New("no pending registration found for this phone number")
		}
		return nil, fmt.Errorf("database error: %w", err)
	}

	// 2. Check if the code has expired
	if time.Now().After(user.VerificationExpiresAt) {
		return nil, errors.New("verification code has expired")
	}

	// 3. Check if the code matches
	if user.VerificationCode != code {
		return nil, errors.New("invalid verification code")
	}

	// 4. Activate the user
	user.Status = domain.UserStatusActive
	user.VerificationCode = "" // Clear the code

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to activate user: %w", err)
	}

	return user, nil
}

// NEW: ResendVerificationCode generates and saves a new code for a pending user.
func (s *UserService) ResendVerificationCode(ctx context.Context, phone, newCode string, ttl time.Duration) error {
	user, err := s.userRepo.FindByPhoneAndStatus(ctx, phone, domain.UserStatusPending)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return errors.New("no pending registration found for this phone number")
		}
		return fmt.Errorf("database error: %w", err)
	}
    
    // Optional: Add rate-limiting logic here to prevent abuse.
    // For example: check if the last code was sent less than a minute ago.

	user.VerificationCode = newCode
	user.VerificationExpiresAt = time.Now().Add(ttl)

	if err := s.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("failed to update verification code: %w", err)
	}
	return nil
}


// Login authenticates a user and returns a JWT token.
// CHANGED: Lockout logic is now based on database fields.
func (s *UserService) Login(ctx context.Context, username, password string) (string, error) {
	user, err := s.userRepo.FindByUsername(ctx, username)
	if err != nil {
		// Note: We don't record a failed attempt here because the user might not exist.
		// The error is generic to prevent username enumeration attacks.
		return "", errors.New("invalid username or password")
	}

	// Check for account lockout from the database
	if user.LockedUntil.After(time.Now()) {
		return "", errors.New("account temporarily locked due to multiple failed login attempts")
	}
    
    // Ensure only active users can log in
    if user.Status != domain.UserStatusActive {
        return "", errors.New("account is not active")
    }

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		// Password does not match, record the failed attempt
		s.recordFailedLogin(ctx, user)
		return "", errors.New("invalid username or password")
	}

	// On success, reset failed login attempts
	if err := s.userRepo.ResetFailedAttempts(ctx, user.ID); err != nil {
		log.Printf("Could not reset failed login attempts for user %d: %v", user.ID, err)
		// We can still let the login proceed, but we log the error.
	}

	token, err := auth.GenerateJWT(user.ID, []byte(s.jwtSecret))
	if err != nil {
		log.Printf("JWT generation error for user %s: %v", username, err)
		return "", errors.New("authentication error")
	}

	return token, nil
}

// CHANGED: This helper now interacts with the repository to update the user's state in the database.
func (s *UserService) recordFailedLogin(ctx context.Context, user *domain.User) {
	user.FailedLoginAttempts++
	if user.FailedLoginAttempts >= maxFailedAttempts {
		user.LockedUntil = time.Now().Add(lockoutDuration)
		log.Printf("User %s locked out until %v due to failed logins", user.Username, user.LockedUntil)
	}

	if err := s.userRepo.Update(ctx, user); err != nil {
		log.Printf("Failed to record failed login for user %s: %v", user.Username, err)
	}
}