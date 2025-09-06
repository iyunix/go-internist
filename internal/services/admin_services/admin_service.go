// File: internal/services/admin_services/admin_service.go
package admin_services

import (
	"context"
	"errors"
	"fmt"

	"github.com/iyunix/go-internist/internal/domain"
	"github.com/iyunix/go-internist/internal/repository"
)

// AdminService provides functionalities for administrative tasks.
type AdminService struct {
	userRepo repository.UserRepository
}

// NewAdminService creates a new instance of AdminService.
func NewAdminService(userRepo repository.UserRepository) *AdminService {
	return &AdminService{
		userRepo: userRepo,
	}
}

// GetAllUsers retrieves a list of all users in the system.
func (s *AdminService) GetAllUsers(ctx context.Context) ([]domain.User, error) {
	users, err := s.userRepo.FindAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get all users: %w", err)
	}
	return users, nil
}

// ChangeUserPlan updates a user's subscription plan.
func (s *AdminService) ChangeUserPlan(ctx context.Context, userID uint, newPlan domain.SubscriptionPlan) error {
	// 1. Validate that the new plan is a real, defined plan.
	if _, ok := domain.PlanCredits[newPlan]; !ok {
		return fmt.Errorf("invalid subscription plan: %s", newPlan)
	}

	// 2. Find the user.
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to find user with ID %d: %w", userID, err)
	}

	// 3. Set the new plan and save.
	user.SubscriptionPlan = newPlan
	if err := s.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("failed to update user plan: %w", err)
	}

	return nil
}

// RenewSubscription resets a user's balance to their plan's full amount.
func (s *AdminService) RenewSubscription(ctx context.Context, userID uint) error {
	// 1. Find the user.
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to find user with ID %d: %w", userID, err)
	}

	// 2. Look up the credit amount for the user's current plan.
	creditsForPlan, ok := domain.PlanCredits[user.SubscriptionPlan]
	if !ok {
		return fmt.Errorf("user has an unknown subscription plan: %s", user.SubscriptionPlan)
	}

	// 3. Reset both the current and total balance to the plan's amount.
	user.CharacterBalance = creditsForPlan
	user.TotalCharacterBalance = creditsForPlan

	// 4. Save the changes.
	if err := s.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("failed to renew subscription: %w", err)
	}

	return nil
}

// TopUpBalance adds credits to a user's current balance without changing their total.
// This is useful for giving a user a small bonus.
func (s *AdminService) TopUpBalance(ctx context.Context, userID uint, amountToAdd int) error {
	if amountToAdd <= 0 {
		return errors.New("amount to add must be a positive number")
	}

	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to find user with ID %d: %w", userID, err)
	}

	user.AddCharacters(amountToAdd) // This only affects the current balance.

	if err := s.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("failed to top-up balance: %w", err)
	}

	return nil
}