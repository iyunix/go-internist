// G:\go_internist\internal\services\user_services\balance_service.go
package user_services

import (
    "context"
    "errors"
    "fmt"

    "github.com/iyunix/go-internist/internal/domain"
    "github.com/iyunix/go-internist/internal/repository"
)

// BalanceService handles user credit and balance management
type BalanceService struct {
    userRepo repository.UserRepository
    logger   Logger
}

// NewBalanceService creates a new balance service
func NewBalanceService(userRepo repository.UserRepository, logger Logger) *BalanceService {
    return &BalanceService{
        userRepo: userRepo,
        logger:   logger,
    }
}

// GetUserBalance retrieves the current balance for a user
func (s *BalanceService) GetUserBalance(ctx context.Context, userID uint) (*BalanceInfo, error) {
    if userID == 0 {
        s.logger.Warn("balance check attempted with invalid user ID", "user_id", userID)
        return nil, errors.New("user ID must be provided")
    }

    s.logger.Debug("retrieving user balance", "user_id", userID)

    user, err := s.userRepo.FindByID(ctx, userID)
    if err != nil {
        s.logger.Error("failed to find user for balance check", 
            "error", err,
            "user_id", userID)
        return nil, fmt.Errorf("failed to find user: %w", err)
    }

    balanceInfo := &BalanceInfo{
        UserID:               userID,
        CurrentBalance:       user.CharacterBalance,
        TotalBalance:        user.TotalCharacterBalance,
        SubscriptionPlan:    user.SubscriptionPlan,
        UsedCredits:         user.TotalCharacterBalance - user.CharacterBalance,
        UsagePercentage:     calculateUsagePercentage(user.CharacterBalance, user.TotalCharacterBalance),
    }

    s.logger.Debug("balance retrieved successfully", 
        "user_id", userID,
        "current_balance", balanceInfo.CurrentBalance,
        "total_balance", balanceInfo.TotalBalance,
        "plan", balanceInfo.SubscriptionPlan,
        "usage_percentage", balanceInfo.UsagePercentage)

    return balanceInfo, nil
}

// DeductCredits deducts credits from user's balance
func (s *BalanceService) DeductCredits(ctx context.Context, userID uint, amount int, operation string) error {
    if userID == 0 {
        s.logger.Warn("credit deduction attempted with invalid user ID", "user_id", userID)
        return errors.New("user ID must be provided")
    }

    if amount <= 0 {
        s.logger.Warn("invalid deduction amount", 
            "user_id", userID,
            "amount", amount)
        return errors.New("deduction amount must be positive")
    }

    if amount > 10000 { // Reasonable safety limit
        s.logger.Warn("excessive deduction amount attempted", 
            "user_id", userID,
            "amount", amount,
            "operation", operation)
        return errors.New("deduction amount too large")
    }

    if operation == "" {
        operation = "unknown"
    }

    s.logger.Info("deducting user credits", 
        "user_id", userID,
        "amount", amount,
        "operation", operation)

    user, err := s.userRepo.FindByID(ctx, userID)
    if err != nil {
        s.logger.Error("failed to find user for credit deduction", 
            "error", err,
            "user_id", userID,
            "operation", operation)
        return fmt.Errorf("failed to find user: %w", err)
    }

    // Check if user has sufficient balance
    if user.CharacterBalance < amount {
        s.logger.Warn("insufficient balance for deduction", 
            "user_id", userID,
            "current_balance", user.CharacterBalance,
            "requested_amount", amount,
            "operation", operation)
        return &InsufficientBalanceError{
            UserID:          userID,
            CurrentBalance:  user.CharacterBalance,
            RequestedAmount: amount,
            Operation:       operation,
        }
    }

    oldBalance := user.CharacterBalance

    // Deduct credits
    user.CharacterBalance -= amount

    if err := s.userRepo.Update(ctx, user); err != nil {
        s.logger.Error("failed to save balance deduction", 
            "error", err,
            "user_id", userID,
            "amount", amount,
            "operation", operation)
        return fmt.Errorf("failed to save balance deduction: %w", err)
    }

    s.logger.Info("credits deducted successfully", 
        "user_id", userID,
        "amount", amount,
        "operation", operation,
        "old_balance", oldBalance,
        "new_balance", user.CharacterBalance,
        "remaining_percentage", calculateUsagePercentage(user.CharacterBalance, user.TotalCharacterBalance))

    return nil
}

// AddCredits adds credits to user's balance
func (s *BalanceService) AddCredits(ctx context.Context, userID uint, amount int, operation string) error {
    if userID == 0 {
        s.logger.Warn("credit addition attempted with invalid user ID", "user_id", userID)
        return errors.New("user ID must be provided")
    }

    if amount <= 0 {
        s.logger.Warn("invalid addition amount", 
            "user_id", userID,
            "amount", amount)
        return errors.New("addition amount must be positive")
    }

    if amount > 50000 { // Reasonable safety limit
        s.logger.Warn("excessive addition amount attempted", 
            "user_id", userID,
            "amount", amount,
            "operation", operation)
        return errors.New("addition amount too large")
    }

    if operation == "" {
        operation = "unknown"
    }

    s.logger.Info("adding user credits", 
        "user_id", userID,
        "amount", amount,
        "operation", operation)

    user, err := s.userRepo.FindByID(ctx, userID)
    if err != nil {
        s.logger.Error("failed to find user for credit addition", 
            "error", err,
            "user_id", userID,
            "operation", operation)
        return fmt.Errorf("failed to find user: %w", err)
    }

    oldBalance := user.CharacterBalance

    // Add credits using domain method
    user.AddCharacters(amount)

    if err := s.userRepo.Update(ctx, user); err != nil {
        s.logger.Error("failed to save balance addition", 
            "error", err,
            "user_id", userID,
            "amount", amount,
            "operation", operation)
        return fmt.Errorf("failed to save balance addition: %w", err)
    }

    s.logger.Info("credits added successfully", 
        "user_id", userID,
        "amount", amount,
        "operation", operation,
        "old_balance", oldBalance,
        "new_balance", user.CharacterBalance)

    return nil
}

// CheckSufficientBalance checks if user has enough credits
func (s *BalanceService) CheckSufficientBalance(ctx context.Context, userID uint, requiredAmount int) (bool, error) {
    if userID == 0 {
        s.logger.Warn("balance check attempted with invalid user ID", "user_id", userID)
        return false, errors.New("user ID must be provided")
    }

    if requiredAmount < 0 {
        s.logger.Warn("balance check attempted with negative amount", 
            "user_id", userID,
            "required_amount", requiredAmount)
        return false, errors.New("required amount cannot be negative")
    }

    s.logger.Debug("checking sufficient balance", 
        "user_id", userID,
        "required_amount", requiredAmount)

    user, err := s.userRepo.FindByID(ctx, userID)
    if err != nil {
        s.logger.Error("failed to find user for balance check", 
            "error", err,
            "user_id", userID)
        return false, fmt.Errorf("failed to find user: %w", err)
    }

    hasSufficientBalance := user.CharacterBalance >= requiredAmount

    s.logger.Debug("balance check completed", 
        "user_id", userID,
        "current_balance", user.CharacterBalance,
        "required_amount", requiredAmount,
        "sufficient", hasSufficientBalance)

    return hasSufficientBalance, nil
}

// RefreshBalance resets user balance to their subscription plan amount
func (s *BalanceService) RefreshBalance(ctx context.Context, userID uint) error {
    if userID == 0 {
        s.logger.Warn("balance refresh attempted with invalid user ID", "user_id", userID)
        return errors.New("user ID must be provided")
    }

    s.logger.Info("refreshing user balance", "user_id", userID)

    user, err := s.userRepo.FindByID(ctx, userID)
    if err != nil {
        s.logger.Error("failed to find user for balance refresh", 
            "error", err,
            "user_id", userID)
        return fmt.Errorf("failed to find user: %w", err)
    }

    // Get credits for current plan
    planCredits, ok := domain.PlanCredits[user.SubscriptionPlan]
    if !ok {
        s.logger.Error("user has unknown subscription plan", 
            "user_id", userID,
            "unknown_plan", user.SubscriptionPlan)
        return fmt.Errorf("unknown subscription plan: %s", user.SubscriptionPlan)
    }

    oldBalance := user.CharacterBalance
    oldTotalBalance := user.TotalCharacterBalance

    // Refresh balances
    user.CharacterBalance = planCredits
    user.TotalCharacterBalance = planCredits

    if err := s.userRepo.Update(ctx, user); err != nil {
        s.logger.Error("failed to save balance refresh", 
            "error", err,
            "user_id", userID)
        return fmt.Errorf("failed to save balance refresh: %w", err)
    }

    s.logger.Info("balance refreshed successfully", 
        "user_id", userID,
        "plan", user.SubscriptionPlan,
        "old_balance", oldBalance,
        "old_total_balance", oldTotalBalance,
        "new_balance", planCredits,
        "new_total_balance", planCredits)

    return nil
}

// calculateUsagePercentage calculates the percentage of credits used
func calculateUsagePercentage(current, total int) float64 {
    if total == 0 {
        return 0
    }
    used := total - current
    return (float64(used) / float64(total)) * 100
}

// BalanceInfo represents user balance information
type BalanceInfo struct {
    UserID           uint                      `json:"user_id"`
    CurrentBalance   int                       `json:"current_balance"`
    TotalBalance     int                       `json:"total_balance"`
    SubscriptionPlan domain.SubscriptionPlan  `json:"subscription_plan"`
    UsedCredits      int                       `json:"used_credits"`
    UsagePercentage  float64                   `json:"usage_percentage"`
}

// InsufficientBalanceError represents insufficient balance error
type InsufficientBalanceError struct {
    UserID          uint   `json:"user_id"`
    CurrentBalance  int    `json:"current_balance"`
    RequestedAmount int    `json:"requested_amount"`
    Operation       string `json:"operation"`
}

func (e *InsufficientBalanceError) Error() string {
    return fmt.Sprintf("insufficient balance: user %d has %d credits but needs %d for operation '%s'",
        e.UserID, e.CurrentBalance, e.RequestedAmount, e.Operation)
}


// GetUserBalanceInfo retrieves both current and total balance for a user
func (s *BalanceService) GetUserBalanceInfo(ctx context.Context, userID uint) (int, int, error) {
    if userID == 0 {
        s.logger.Warn("balance info requested with invalid user ID", "user_id", userID)
        return 0, 0, errors.New("user ID must be provided")
    }

    s.logger.Debug("retrieving user balance info", "user_id", userID)

    user, err := s.userRepo.FindByID(ctx, userID)
    if err != nil {
        s.logger.Error("failed to find user for balance info", 
            "error", err,
            "user_id", userID)
        return 0, 0, fmt.Errorf("failed to find user: %w", err)
    }

    currentBalance := user.CharacterBalance
    totalBalance := user.TotalCharacterBalance

    s.logger.Debug("balance info retrieved successfully", 
        "user_id", userID,
        "current_balance", currentBalance,
        "total_balance", totalBalance,
        "subscription_plan", user.SubscriptionPlan)

    return currentBalance, totalBalance, nil
}
