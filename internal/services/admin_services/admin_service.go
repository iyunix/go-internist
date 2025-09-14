// File: internal/services/admin_services/admin_service.go
package admin_services

import (
	"context"
	"errors"
	"fmt"

	"github.com/iyunix/go-internist/internal/domain"
	"github.com/iyunix/go-internist/internal/repository/user"
)

type Logger interface {
	Info(msg string, keysAndValues ...interface{})
	Error(msg string, keysAndValues ...interface{})
	Debug(msg string, keysAndValues ...interface{})
	Warn(msg string, keysAndValues ...interface{})
}

type AdminService struct {
	userRepo user.UserRepository
	logger   Logger
}

func NewAdminService(userRepo user.UserRepository, logger Logger) *AdminService {
	return &AdminService{
		userRepo: userRepo,
		logger:   logger,
	}
}

// GetAllUsers retrieves a paginated and searchable list of all users.
// It now returns the list of users, the total count for pagination, and an error.
func (s *AdminService) GetAllUsers(ctx context.Context, page, limit int, search string) ([]domain.User, int64, error) {
	s.logger.Info("retrieving users for admin dashboard", "page", page, "limit", limit, "search", search)

	// A page or limit of 0 can mean "fetch all" for exports.
	if page == 0 || limit == 0 {
		users, err := s.userRepo.FindAll(ctx) // Use the existing FindAll for full exports
		if err != nil {
			s.logger.Error("failed to retrieve all users for export", "error", err)
			return nil, 0, err
		}
		return users, int64(len(users)), nil
	}

	// This now calls a new repository method that supports pagination and search.
	users, total, err := s.userRepo.FindAllWithPaginationAndSearch(ctx, page, limit, search)
	if err != nil {
		s.logger.Error("failed to retrieve paginated users", "error", err)
		return nil, 0, fmt.Errorf("failed to get users: %w", err)
	}

	s.logger.Info("paginated users retrieved successfully", "user_count", len(users), "total_users", total)
	return users, total, nil
}

// ChangeUserPlan updates a user's subscription plan. (No changes needed)
func (s *AdminService) ChangeUserPlan(ctx context.Context, userID uint, newPlan domain.SubscriptionPlan) error {
	if userID == 0 {
		s.logger.Warn("attempt to change plan with invalid user ID", "user_id", userID)
		return errors.New("user ID must be provided")
	}
	s.logger.Info("changing user subscription plan", "user_id", userID, "new_plan", newPlan)
	if _, ok := domain.PlanCredits[newPlan]; !ok {
		s.logger.Warn("invalid subscription plan requested", "user_id", userID, "invalid_plan", newPlan)
		return fmt.Errorf("invalid subscription plan: %s", newPlan)
	}
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		s.logger.Error("failed to find user for plan change", "error", err, "user_id", userID)
		return fmt.Errorf("failed to find user with ID %d: %w", userID, err)
	}
	oldPlan := user.SubscriptionPlan
	user.SubscriptionPlan = newPlan
	if err := s.userRepo.Update(ctx, user); err != nil {
		s.logger.Error("failed to update user plan", "error", err, "user_id", userID, "new_plan", newPlan)
		return fmt.Errorf("failed to update user plan: %w", err)
	}
	s.logger.Info("user subscription plan changed successfully", "user_id", userID, "old_plan", oldPlan, "new_plan", newPlan)
	return nil
}

// RenewSubscription resets a user's balance to their plan's full amount. (No changes needed)
func (s *AdminService) RenewSubscription(ctx context.Context, userID uint) error {
	if userID == 0 {
		s.logger.Warn("attempt to renew subscription with invalid user ID", "user_id", userID)
		return errors.New("user ID must be provided")
	}
	s.logger.Info("renewing user subscription", "user_id", userID)
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		s.logger.Error("failed to find user for subscription renewal", "error", err, "user_id", userID)
		return fmt.Errorf("failed to find user with ID %d: %w", userID, err)
	}
	creditsForPlan, ok := domain.PlanCredits[user.SubscriptionPlan]
	if !ok {
		s.logger.Error("user has unknown subscription plan", "user_id", userID, "unknown_plan", user.SubscriptionPlan)
		return fmt.Errorf("user has an unknown subscription plan: %s", user.SubscriptionPlan)
	}
	oldBalance := user.CharacterBalance
	user.CharacterBalance = creditsForPlan
	user.TotalCharacterBalance = creditsForPlan
	if err := s.userRepo.Update(ctx, user); err != nil {
		s.logger.Error("failed to save subscription renewal", "error", err, "user_id", userID)
		return fmt.Errorf("failed to renew subscription: %w", err)
	}
	s.logger.Info("subscription renewed successfully", "user_id", userID, "plan", user.SubscriptionPlan, "old_balance", oldBalance, "new_balance", creditsForPlan)
	return nil
}

// TopUpBalance adds credits to a user's current balance. (No changes needed)
func (s *AdminService) TopUpBalance(ctx context.Context, userID uint, amountToAdd int) error {
	if userID == 0 {
		s.logger.Warn("attempt to top up balance with invalid user ID", "user_id", userID)
		return errors.New("user ID must be provided")
	}
	if amountToAdd <= 0 {
		s.logger.Warn("attempt to top up balance with invalid amount", "user_id", userID, "amount", amountToAdd)
		return errors.New("amount to add must be a positive number")
	}
	if amountToAdd > 10000 {
		s.logger.Warn("attempt to top up balance with excessive amount", "user_id", userID, "amount", amountToAdd, "max_allowed", 10000)
		return errors.New("amount to add exceeds maximum allowed (10000)")
	}
	s.logger.Info("topping up user balance", "user_id", userID, "amount", amountToAdd)
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		s.logger.Error("failed to find user for balance top-up", "error", err, "user_id", userID)
		return fmt.Errorf("failed to find user with ID %d: %w", userID, err)
	}
	oldBalance := user.CharacterBalance
	user.AddCharacters(amountToAdd)
	if err := s.userRepo.Update(ctx, user); err != nil {
		s.logger.Error("failed to save balance top-up", "error", err, "user_id", userID, "amount", amountToAdd)
		return fmt.Errorf("failed to top-up balance: %w", err)
	}
	s.logger.Info("balance topped up successfully", "user_id", userID, "amount_added", amountToAdd, "old_balance", oldBalance, "new_balance", user.CharacterBalance)
	return nil
}