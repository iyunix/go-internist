// File: internal/services/user_services/verification_service.go
package user_services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/iyunix/go-internist/internal/domain"
	"github.com/iyunix/go-internist/internal/repository"
)


// VerificationService holds dependencies for user verification logic.
type VerificationService struct {
	userRepo         repository.UserRepository
	adminPhoneNumber string
}

// NewVerificationService creates a new VerificationService.
func NewVerificationService(userRepo repository.UserRepository, adminPhoneNumber string) *VerificationService {
	return &VerificationService{
		userRepo:         userRepo,
		adminPhoneNumber: adminPhoneNumber,
	}
}

// InitiateVerification handles the first step of registration.
func (s *VerificationService) InitiateVerification(ctx context.Context, user *domain.User, code string, ttl time.Duration) error {
	if err := s.checkUserExists(ctx, user.Username, user.PhoneNumber); err != nil {
		return err
	}

	if err := s.hashUserPassword(user); err != nil {
		return err
	}

	// This function now correctly sets up the user with a default plan.
	s.setupPendingUser(user, code, ttl)

	_, err := s.userRepo.Create(ctx, user)
	if err != nil {
		return fmt.Errorf("failed to create pending user: %w", err)
	}

	return nil
}

// FinalizeVerification completes the registration process.
func (s *VerificationService) FinalizeVerification(ctx context.Context, phone, code string) (*domain.User, error) {
	user, err := s.findPendingUser(ctx, phone)
	if err != nil {
		return nil, err
	}

	if err := s.validateVerificationCode(user, code); err != nil {
		return nil, err
	}

	if err := s.activateUser(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}
// ResendVerificationCode generates and saves a new code for a pending user.
func (s *VerificationService) ResendVerificationCode(ctx context.Context, phone, newCode string, ttl time.Duration) error {
	user, err := s.findPendingUser(ctx, phone)
	if err != nil {
		return err
	}

	user.VerificationCode = newCode
	user.VerificationExpiresAt = time.Now().Add(ttl)

	if err := s.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("failed to update verification code: %w", err)
	}
	return nil
}

// --- Private helper methods ---

func (s *VerificationService) checkUserExists(ctx context.Context, username, phone string) error {
	existingUser, err := s.userRepo.FindByUsernameOrPhone(ctx, username, phone)
	if err != nil && !errors.Is(err, repository.ErrUserNotFound) {
		return fmt.Errorf("database error while checking for existing user: %w", err)
	}
	if existingUser != nil {
		return errors.New("a user with that username or phone number already exists")
	}
	return nil
}

func (s *VerificationService) hashUserPassword(user *domain.User) error {
	hashedPwd, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Password hashing error: %v", err)
		return errors.New("internal error occurred during registration")
	}
	user.Password = string(hashedPwd)
	return nil
}

// --- THIS FUNCTION HAS BEEN UPDATED ---
func (s *VerificationService) setupPendingUser(user *domain.User, code string, ttl time.Duration) {
	user.Status = domain.UserStatusPending
	user.VerificationCode = code
	user.VerificationExpiresAt = time.Now().Add(ttl)

	// NEW LOGIC: Set balance based on the default subscription plan.
	defaultPlan := domain.PlanBasic
	startingCredits, ok := domain.PlanCredits[defaultPlan]
	if !ok {
		// This is a fallback in case the 'basic' plan is ever removed from the map.
		// It's good practice but should ideally never happen.
		log.Printf("Warning: Default plan '%s' not found in PlanCredits map. Defaulting to 2500 credits.", defaultPlan)
		startingCredits = 2500
	}

	user.SubscriptionPlan = defaultPlan
	user.CharacterBalance = startingCredits
	user.TotalCharacterBalance = startingCredits // Also set the total.
}


func (s *VerificationService) findPendingUser(ctx context.Context, phone string) (*domain.User, error) {
	user, err := s.userRepo.FindByPhoneAndStatus(ctx, phone, domain.UserStatusPending)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, errors.New("no pending registration found for this phone number")
		}
		return nil, fmt.Errorf("database error: %w", err)
	}
	return user, nil
}

func (s *VerificationService) validateVerificationCode(user *domain.User, code string) error {
	if time.Now().After(user.VerificationExpiresAt) {
		return errors.New("verification code has expired")
	}

	if user.VerificationCode != code {
		return errors.New("invalid verification code")
	}

	return nil
}


// --- THIS FUNCTION HAS BEEN MODIFIED ---
func (s *VerificationService) activateUser(ctx context.Context, user *domain.User) error {
	user.Status = domain.UserStatusActive
	user.VerificationCode = "" // Clear the code

	// 4. UPDATE the check to use the stored admin phone number from the config.
	if s.adminPhoneNumber != "" && user.PhoneNumber == s.adminPhoneNumber {
		user.IsAdmin = true
		log.Printf("User with phone number %s has been automatically promoted to ADMIN.", user.PhoneNumber)
	}

	if err := s.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("failed to activate user: %w", err)
	}
	return nil
}