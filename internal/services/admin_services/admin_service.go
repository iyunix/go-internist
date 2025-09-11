package admin_services

import (
    "context"
    "errors"
    "fmt"

    "github.com/iyunix/go-internist/internal/domain"
    "github.com/iyunix/go-internist/internal/repository/user"
)

// Logger interface for admin operations
type Logger interface {
    Info(msg string, keysAndValues ...interface{})
    Error(msg string, keysAndValues ...interface{})
    Debug(msg string, keysAndValues ...interface{})
    Warn(msg string, keysAndValues ...interface{})
}

// AdminService provides functionalities for administrative tasks.
type AdminService struct {
    userRepo user.UserRepository
    logger   Logger
}

// NewAdminService creates a new instance of AdminService.
func NewAdminService(userRepo user.UserRepository, logger Logger) *AdminService {
    return &AdminService{
        userRepo: userRepo,
        logger:   logger,
    }
}

// GetAllUsers retrieves a list of all users in the system.
func (s *AdminService) GetAllUsers(ctx context.Context) ([]domain.User, error) {
    s.logger.Info("retrieving all users for admin dashboard")
    
    users, err := s.userRepo.FindAll(ctx)
    if err != nil {
        s.logger.Error("failed to retrieve all users", "error", err)
        return nil, fmt.Errorf("failed to get all users: %w", err)
    }
    
    s.logger.Info("users retrieved successfully", "user_count", len(users))
    return users, nil
}

// ChangeUserPlan updates a user's subscription plan.
func (s *AdminService) ChangeUserPlan(ctx context.Context, userID uint, newPlan domain.SubscriptionPlan) error {
    if userID == 0 {
        s.logger.Warn("attempt to change plan with invalid user ID", "user_id", userID)
        return errors.New("user ID must be provided")
    }
    
    s.logger.Info("changing user subscription plan", 
        "user_id", userID,
        "new_plan", newPlan)
    
    // Validate that the new plan is a real, defined plan.
    if _, ok := domain.PlanCredits[newPlan]; !ok {
        s.logger.Warn("invalid subscription plan requested", 
            "user_id", userID,
            "invalid_plan", newPlan)
        return fmt.Errorf("invalid subscription plan: %s", newPlan)
    }

    // Find the user.
    user, err := s.userRepo.FindByID(ctx, userID)
    if err != nil {
        s.logger.Error("failed to find user for plan change", 
            "error", err,
            "user_id", userID)
        return fmt.Errorf("failed to find user with ID %d: %w", userID, err)
    }

    oldPlan := user.SubscriptionPlan
    
    // Set the new plan and save.
    user.SubscriptionPlan = newPlan
    if err := s.userRepo.Update(ctx, user); err != nil {
        s.logger.Error("failed to update user plan", 
            "error", err,
            "user_id", userID,
            "new_plan", newPlan)
        return fmt.Errorf("failed to update user plan: %w", err)
    }

    s.logger.Info("user subscription plan changed successfully",
        "user_id", userID,
        "old_plan", oldPlan,
        "new_plan", newPlan)
    return nil
}

// RenewSubscription resets a user's balance to their plan's full amount.
func (s *AdminService) RenewSubscription(ctx context.Context, userID uint) error {
    if userID == 0 {
        s.logger.Warn("attempt to renew subscription with invalid user ID", "user_id", userID)
        return errors.New("user ID must be provided")
    }
    
    s.logger.Info("renewing user subscription", "user_id", userID)
    
    // Find the user.
    user, err := s.userRepo.FindByID(ctx, userID)
    if err != nil {
        s.logger.Error("failed to find user for subscription renewal", 
            "error", err,
            "user_id", userID)
        return fmt.Errorf("failed to find user with ID %d: %w", userID, err)
    }

    // Look up the credit amount for the user's current plan.
    creditsForPlan, ok := domain.PlanCredits[user.SubscriptionPlan]
    if !ok {
        s.logger.Error("user has unknown subscription plan", 
            "user_id", userID,
            "unknown_plan", user.SubscriptionPlan)
        return fmt.Errorf("user has an unknown subscription plan: %s", user.SubscriptionPlan)
    }

    oldBalance := user.CharacterBalance
    
    // Reset both the current and total balance to the plan's amount.
    user.CharacterBalance = creditsForPlan
    user.TotalCharacterBalance = creditsForPlan

    // Save the changes.
    if err := s.userRepo.Update(ctx, user); err != nil {
        s.logger.Error("failed to save subscription renewal", 
            "error", err,
            "user_id", userID)
        return fmt.Errorf("failed to renew subscription: %w", err)
    }

    s.logger.Info("subscription renewed successfully",
        "user_id", userID,
        "plan", user.SubscriptionPlan,
        "old_balance", oldBalance,
        "new_balance", creditsForPlan)
    return nil
}

// TopUpBalance adds credits to a user's current balance without changing their total.
// This is useful for giving a user a small bonus.
func (s *AdminService) TopUpBalance(ctx context.Context, userID uint, amountToAdd int) error {
    if userID == 0 {
        s.logger.Warn("attempt to top up balance with invalid user ID", "user_id", userID)
        return errors.New("user ID must be provided")
    }
    if amountToAdd <= 0 {
        s.logger.Warn("attempt to top up balance with invalid amount", 
            "user_id", userID,
            "amount", amountToAdd)
        return errors.New("amount to add must be a positive number")
    }
    if amountToAdd > 10000 { // Reasonable admin limit
        s.logger.Warn("attempt to top up balance with excessive amount", 
            "user_id", userID,
            "amount", amountToAdd,
            "max_allowed", 10000)
        return errors.New("amount to add exceeds maximum allowed (10000)")
    }

    s.logger.Info("topping up user balance", 
        "user_id", userID,
        "amount", amountToAdd)

    user, err := s.userRepo.FindByID(ctx, userID)
    if err != nil {
        s.logger.Error("failed to find user for balance top-up", 
            "error", err,
            "user_id", userID)
        return fmt.Errorf("failed to find user with ID %d: %w", userID, err)
    }

    oldBalance := user.CharacterBalance
    user.AddCharacters(amountToAdd) // This only affects the current balance.

    if err := s.userRepo.Update(ctx, user); err != nil {
        s.logger.Error("failed to save balance top-up", 
            "error", err,
            "user_id", userID,
            "amount", amountToAdd)
        return fmt.Errorf("failed to top-up balance: %w", err)
    }

    s.logger.Info("balance topped up successfully",
        "user_id", userID,
        "amount_added", amountToAdd,
        "old_balance", oldBalance,
        "new_balance", user.CharacterBalance)
    return nil
}
