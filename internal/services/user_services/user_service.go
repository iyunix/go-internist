package user_services

import (
    "context"
    "errors"
    "fmt"

    "github.com/iyunix/go-internist/internal/domain"
    "github.com/iyunix/go-internist/internal/repository/user"
)

// UserService handles core user management operations
type UserService struct {
    userRepo     user.UserRepository
    jwtSecretKey string
    adminPhone   string
    logger       Logger
}

// NewUserService creates a new user service
func NewUserService(userRepo user.UserRepository, jwtSecretKey, adminPhone string, logger Logger) *UserService {
    return &UserService{
        userRepo:     userRepo,
        jwtSecretKey: jwtSecretKey,
        adminPhone:   adminPhone,
        logger:       logger,
    }
}


// GetUserByUsername retrieves a user by their username
func (s *UserService) GetUserByUsername(ctx context.Context, username string) (*domain.User, error) {
    if username == "" {
        s.logger.Warn("user lookup attempted with empty username")
        return nil, errors.New("username must be provided")
    }

    s.logger.Debug("retrieving user by username", "username", username)

    user, err := s.userRepo.FindByUsername(ctx, username)
    if err != nil {
        s.logger.Error("failed to find user by username",
            "error", err,
            "username", username)
        return nil, fmt.Errorf("failed to find user: %w", err)
    }

    s.logger.Debug("user retrieved successfully",
        "user_id", user.ID,
        "username", user.Username,
        "is_verified", user.IsVerified,
        "is_admin", user.IsAdmin)

    return user, nil
}


// GetUserByID retrieves a user by their ID
func (s *UserService) GetUserByID(ctx context.Context, userID uint) (*domain.User, error) {
    if userID == 0 {
        s.logger.Warn("user lookup attempted with invalid user ID", "user_id", userID)
        return nil, errors.New("user ID must be provided")
    }

    s.logger.Debug("retrieving user by ID", "user_id", userID)

    user, err := s.userRepo.FindByID(ctx, userID)
    if err != nil {
        s.logger.Error("failed to find user by ID", 
            "error", err,
            "user_id", userID)
        return nil, fmt.Errorf("failed to find user: %w", err)
    }

    s.logger.Debug("user retrieved successfully", 
        "user_id", userID,
        "phone", user.PhoneNumber[:min(4, len(user.PhoneNumber))]+"****",  // FIXED
        "is_verified", user.IsVerified,
        "is_admin", user.IsAdmin,
        "subscription_plan", user.SubscriptionPlan)

    return user, nil
}

// GetUserByPhone retrieves a user by their phone number
func (s *UserService) GetUserByPhone(ctx context.Context, phone string) (*domain.User, error) {
    if phone == "" {
        s.logger.Warn("user lookup attempted with empty phone number")
        return nil, errors.New("phone number must be provided")
    }

    s.logger.Debug("retrieving user by phone", 
        "phone", phone[:min(4, len(phone))]+"****")

    user, err := s.userRepo.FindByPhone(ctx, phone)
    if err != nil {
        s.logger.Error("failed to find user by phone", 
            "error", err,
            "phone", phone[:min(4, len(phone))]+"****")
        return nil, fmt.Errorf("failed to find user: %w", err)
    }

    s.logger.Debug("user retrieved successfully", 
        "user_id", user.ID,
        "phone", phone[:min(4, len(phone))]+"****",
        "is_verified", user.IsVerified,
        "is_admin", user.IsAdmin)

    return user, nil
}

// UpdateUser updates user information
func (s *UserService) UpdateUser(ctx context.Context, user *domain.User) error {
    if user == nil {
        s.logger.Warn("user update attempted with nil user")
        return errors.New("user cannot be nil")
    }

    if user.ID == 0 {
        s.logger.Warn("user update attempted with invalid user ID", "user_id", user.ID)
        return errors.New("user ID must be provided")
    }

    s.logger.Info("updating user", 
        "user_id", user.ID,
        "phone", user.PhoneNumber[:min(4, len(user.PhoneNumber))]+"****",  // FIXED
        "is_verified", user.IsVerified,
        "subscription_plan", user.SubscriptionPlan)

    // Validate phone number if changed
    if user.PhoneNumber == "" {  // FIXED
        s.logger.Warn("user update attempted with empty phone", "user_id", user.ID)
        return errors.New("phone number cannot be empty")
    }

    // Check if phone is already taken by another user
    existingUser, err := s.userRepo.FindByPhone(ctx, user.PhoneNumber)  // FIXED
    if err == nil && existingUser != nil && existingUser.ID != user.ID {
        s.logger.Warn("user update failed - phone already taken", 
            "user_id", user.ID,
            "phone", user.PhoneNumber[:min(4, len(user.PhoneNumber))]+"****",  // FIXED
            "existing_user_id", existingUser.ID)
        return errors.New("phone number is already taken by another user")
    }

    if err := s.userRepo.Update(ctx, user); err != nil {
        s.logger.Error("failed to update user", 
            "error", err,
            "user_id", user.ID,
            "phone", user.PhoneNumber[:min(4, len(user.PhoneNumber))]+"****")  // FIXED
        return fmt.Errorf("failed to update user: %w", err)
    }

    s.logger.Info("user updated successfully", 
        "user_id", user.ID,
        "phone", user.PhoneNumber[:min(4, len(user.PhoneNumber))]+"****",  // FIXED
        "is_verified", user.IsVerified,
        "subscription_plan", user.SubscriptionPlan,
        "current_balance", user.CharacterBalance)

    return nil
}

// DeleteUser soft deletes a user (admin only)
func (s *UserService) DeleteUser(ctx context.Context, userID uint, adminUserID uint) error {
    if userID == 0 {
        s.logger.Warn("user deletion attempted with invalid user ID", "user_id", userID)
        return errors.New("user ID must be provided")
    }

    if adminUserID == 0 {
        s.logger.Warn("user deletion attempted with invalid admin ID", 
            "user_id", userID,
            "admin_user_id", adminUserID)
        return errors.New("admin user ID must be provided")
    }

    s.logger.Info("attempting user deletion", 
        "user_id", userID,
        "admin_user_id", adminUserID)

    // Verify admin permissions
    adminUser, err := s.userRepo.FindByID(ctx, adminUserID)
    if err != nil {
        s.logger.Error("failed to find admin user for deletion", 
            "error", err,
            "admin_user_id", adminUserID)
        return fmt.Errorf("failed to verify admin user: %w", err)
    }

    if !adminUser.IsAdmin {
        s.logger.Warn("non-admin user attempted user deletion", 
            "admin_user_id", adminUserID,
            "target_user_id", userID)
        return errors.New("only admin users can delete accounts")
    }

    // Find target user
    targetUser, err := s.userRepo.FindByID(ctx, userID)
    if err != nil {
        s.logger.Error("failed to find target user for deletion", 
            "error", err,
            "user_id", userID,
            "admin_user_id", adminUserID)
        return fmt.Errorf("failed to find target user: %w", err)
    }

    // Prevent admin from deleting themselves
    if userID == adminUserID {
        s.logger.Warn("admin user attempted to delete themselves", 
            "admin_user_id", adminUserID)
        return errors.New("admin users cannot delete their own account")
    }

    if err := s.userRepo.Delete(ctx, userID); err != nil {
        s.logger.Error("failed to delete user", 
            "error", err,
            "user_id", userID,
            "admin_user_id", adminUserID)
        return fmt.Errorf("failed to delete user: %w", err)
    }

    s.logger.Info("user deleted successfully", 
        "user_id", userID,
        "admin_user_id", adminUserID,
        "deleted_phone", targetUser.PhoneNumber[:min(4, len(targetUser.PhoneNumber))]+"****")  // FIXED

    return nil
}

// IsAdmin checks if a user has admin privileges
func (s *UserService) IsAdmin(ctx context.Context, userID uint) (bool, error) {
    if userID == 0 {
        s.logger.Warn("admin check attempted with invalid user ID", "user_id", userID)
        return false, errors.New("user ID must be provided")
    }

    user, err := s.userRepo.FindByID(ctx, userID)
    if err != nil {
        s.logger.Error("failed to find user for admin check", 
            "error", err,
            "user_id", userID)
        return false, fmt.Errorf("failed to find user: %w", err)
    }

    s.logger.Debug("admin check completed", 
        "user_id", userID,
        "is_admin", user.IsAdmin)

    return user.IsAdmin, nil
}

// GetUserStats returns user statistics for admin dashboard
func (s *UserService) GetUserStats(ctx context.Context, adminUserID uint) (*UserStats, error) {
    if adminUserID == 0 {
        s.logger.Warn("user stats requested with invalid admin ID", "admin_user_id", adminUserID)
        return nil, errors.New("admin user ID must be provided")
    }

    // Verify admin permissions
    isAdmin, err := s.IsAdmin(ctx, adminUserID)
    if err != nil {
        return nil, fmt.Errorf("failed to verify admin status: %w", err)
    }

    if !isAdmin {
        s.logger.Warn("non-admin user requested user stats", "admin_user_id", adminUserID)
        return nil, errors.New("only admin users can access user statistics")
    }

    s.logger.Info("retrieving user statistics", "admin_user_id", adminUserID)

    // Get all users for statistics
    users, err := s.userRepo.FindAll(ctx)
    if err != nil {
        s.logger.Error("failed to retrieve users for statistics", 
            "error", err,
            "admin_user_id", adminUserID)
        return nil, fmt.Errorf("failed to retrieve users: %w", err)
    }

    stats := &UserStats{
        TotalUsers:       len(users),
        VerifiedUsers:    0,
        AdminUsers:       0,
        PlanDistribution: make(map[domain.SubscriptionPlan]int),
    }

    // Calculate statistics
    for _, user := range users {
        if user.IsVerified {
            stats.VerifiedUsers++
        }
        if user.IsAdmin {
            stats.AdminUsers++
        }
        stats.PlanDistribution[user.SubscriptionPlan]++
    }

    s.logger.Info("user statistics calculated", 
        "admin_user_id", adminUserID,
        "total_users", stats.TotalUsers,
        "verified_users", stats.VerifiedUsers,
        "admin_users", stats.AdminUsers)

    return stats, nil
}

// UserStats represents user statistics
type UserStats struct {
    TotalUsers       int                                    `json:"total_users"`
    VerifiedUsers    int                                    `json:"verified_users"`
    AdminUsers       int                                    `json:"admin_users"`
    PlanDistribution map[domain.SubscriptionPlan]int       `json:"plan_distribution"`
}

// Add these methods to your UserService

// GetCharacterBalance retrieves the user's current character balance
func (s *UserService) GetCharacterBalance(ctx context.Context, userID uint) (int, error) {
    if userID == 0 {
        s.logger.Warn("character balance check attempted with invalid user ID", "user_id", userID)
        return 0, errors.New("user ID must be provided")
    }

    s.logger.Debug("retrieving user character balance", "user_id", userID)

    user, err := s.userRepo.FindByID(ctx, userID)
    if err != nil {
        s.logger.Error("failed to find user for balance check", 
            "error", err,
            "user_id", userID)
        return 0, fmt.Errorf("failed to find user: %w", err)
    }

    s.logger.Debug("character balance retrieved successfully", 
        "user_id", userID,
        "balance", user.CharacterBalance)

    return user.CharacterBalance, nil
}

// FIXED: CanUserAskQuestion - Updated signature to match chat handler expectations
func (s *UserService) CanUserAskQuestion(ctx context.Context, userID uint, questionLength int) (bool, int, error) {
    if userID == 0 {
        s.logger.Warn("can ask question check attempted with invalid user ID", "user_id", userID)
        return false, 0, errors.New("user ID must be provided")
    }

    if questionLength <= 0 {
        s.logger.Warn("can ask question check attempted with invalid question length", 
            "user_id", userID,
            "question_length", questionLength)
        return false, 0, errors.New("question length must be positive")
    }

    s.logger.Debug("checking if user can ask question", "user_id", userID, "question_length", questionLength)

    user, err := s.userRepo.FindByID(ctx, userID)
    if err != nil {
        s.logger.Error("failed to find user for question check", 
            "error", err,
            "user_id", userID)
        return false, 0, fmt.Errorf("failed to find user: %w", err)
    }

    canAsk := user.CanAskQuestion()
    chargeAmount := user.CalculateChargeForQuestion(questionLength)

    s.logger.Debug("question eligibility checked", 
        "user_id", userID,
        "can_ask", canAsk,
        "current_balance", user.CharacterBalance,
        "charge_amount", chargeAmount)

    return canAsk, chargeAmount, nil
}

// FIXED: DeductCharactersForQuestion - Updated signature to return 2 values
func (s *UserService) DeductCharactersForQuestion(ctx context.Context, userID uint, questionLength int) (int, error) {
    if userID == 0 {
        s.logger.Warn("character deduction attempted with invalid user ID", "user_id", userID)
        return 0, errors.New("user ID must be provided")
    }

    if questionLength <= 0 {
        s.logger.Warn("character deduction attempted with invalid question length", 
            "user_id", userID,
            "question_length", questionLength)
        return 0, errors.New("question length must be positive")
    }

    s.logger.Info("deducting characters for question", 
        "user_id", userID,
        "question_length", questionLength)

    user, err := s.userRepo.FindByID(ctx, userID)
    if err != nil {
        s.logger.Error("failed to find user for character deduction", 
            "error", err,
            "user_id", userID)
        return 0, fmt.Errorf("failed to find user: %w", err)
    }

    oldBalance := user.CharacterBalance
    chargeAmount := user.CalculateChargeForQuestion(questionLength)

    // Use domain method to deduct characters
    if err := user.DeductCharacters(questionLength); err != nil {
        s.logger.Warn("character deduction failed", 
            "error", err,
            "user_id", userID,
            "question_length", questionLength,
            "current_balance", user.CharacterBalance)
        return 0, err
    }

    // Save updated user
    if err := s.userRepo.Update(ctx, user); err != nil {
        s.logger.Error("failed to save character deduction", 
            "error", err,
            "user_id", userID)
        return 0, fmt.Errorf("failed to update user balance: %w", err)
    }

    s.logger.Info("characters deducted successfully", 
        "user_id", userID,
        "question_length", questionLength,
        "charge_amount", chargeAmount,
        "old_balance", oldBalance,
        "new_balance", user.CharacterBalance)

    return chargeAmount, nil
}

