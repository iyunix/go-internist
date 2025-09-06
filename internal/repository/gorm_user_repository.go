// File: internal/repository/gorm_user_repository.go
package repository

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/iyunix/go-internist/internal/domain"
	"gorm.io/gorm"
)

// NEW: Export a standard error for not found, so services can check for it.
var ErrUserNotFound = errors.New("user not found")

type gormUserRepository struct {
	db *gorm.DB
}

func NewGormUserRepository(db *gorm.DB) UserRepository {
	return &gormUserRepository{db: db}
}

// Create inserts a new user record.
func (r *gormUserRepository) Create(ctx context.Context, user *domain.User) (*domain.User, error) {
	if err := r.db.WithContext(ctx).Create(user).Error; err != nil {
		log.Printf("[UserRepository] Create error for username %s: %v", user.Username, err)
		return nil, errors.New("database error creating user")
	}
	return user, nil
}

// NEW: Update saves changes to an existing user record.
func (r *gormUserRepository) Update(ctx context.Context, user *domain.User) error {
	if err := r.db.WithContext(ctx).Save(user).Error; err != nil {
		log.Printf("[UserRepository] Update error for user ID %d: %v", user.ID, err)
		return errors.New("database error updating user")
	}
	return nil
}

// FindByUsername finds a user by their username.
func (r *gormUserRepository) FindByUsername(ctx context.Context, username string) (*domain.User, error) {
	var user domain.User
	err := r.db.WithContext(ctx).Where("username = ?", username).First(&user).Error
	return r.handleFindError(err, &user, "FindByUsername", username)
}

// FindByID finds a user by their ID.
func (r *gormUserRepository) FindByID(ctx context.Context, id uint) (*domain.User, error) {
	var user domain.User
	err := r.db.WithContext(ctx).First(&user, id).Error
	return r.handleFindError(err, &user, "FindByID", id)
}

// NEW: FindByUsernameOrPhone finds a user by either their username or phone number.
func (r *gormUserRepository) FindByUsernameOrPhone(ctx context.Context, username, phone string) (*domain.User, error) {
	var user domain.User
	err := r.db.WithContext(ctx).Where("username = ? OR phone_number = ?", username, phone).First(&user).Error
	return r.handleFindError(err, &user, "FindByUsernameOrPhone", username)
}

// NEW: FindByPhoneAndStatus finds a user with a specific phone number and status.
func (r *gormUserRepository) FindByPhoneAndStatus(ctx context.Context, phone string, status domain.UserStatus) (*domain.User, error) {
	var user domain.User
	err := r.db.WithContext(ctx).Where("phone_number = ? AND status = ?", phone, status).First(&user).Error
	return r.handleFindError(err, &user, "FindByPhoneAndStatus", phone)
}

// NEW: ResetFailedAttempts resets lockout fields for a user after a successful login.
func (r *gormUserRepository) ResetFailedAttempts(ctx context.Context, id uint) error {
	err := r.db.WithContext(ctx).Model(&domain.User{}).Where("id = ?", id).Updates(map[string]interface{}{
		"failed_login_attempts": 0,
		"locked_until":          time.Time{}, // Resets to zero value
	}).Error

	if err != nil {
		log.Printf("[UserRepository] ResetFailedAttempts error for user ID %d: %v", id, err)
		return errors.New("database error resetting failed attempts")
	}
	return nil
}

// NEW: GetCharacterBalance retrieves a user's current character balance.
func (r *gormUserRepository) GetCharacterBalance(ctx context.Context, userID uint) (int, error) {
	var balance int
	err := r.db.WithContext(ctx).Model(&domain.User{}).Where("id = ?", userID).Select("character_balance").Scan(&balance).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, ErrUserNotFound
		}
		log.Printf("[UserRepository] GetCharacterBalance error for user ID %d: %v", userID, err)
		return 0, errors.New("database error getting character balance")
	}
	return balance, nil
}

// NEW: UpdateCharacterBalance updates a user's character balance.
func (r *gormUserRepository) UpdateCharacterBalance(ctx context.Context, userID uint, newBalance int) error {
	result := r.db.WithContext(ctx).Model(&domain.User{}).Where("id = ?", userID).Update("character_balance", newBalance)

	if result.Error != nil {
		log.Printf("[UserRepository] UpdateCharacterBalance error for user ID %d: %v", userID, result.Error)
		return errors.New("database error updating character balance")
	}

	if result.RowsAffected == 0 {
		return ErrUserNotFound
	}

	return nil
}

// --- ADD THIS NEW FUNCTION ---
// FindAll retrieves all users from the database for the admin panel.
func (r *gormUserRepository) FindAll(ctx context.Context) ([]domain.User, error) {
	var users []domain.User
	// We order by ID to ensure a consistent, predictable order.
	if err := r.db.WithContext(ctx).Order("id asc").Find(&users).Error; err != nil {
		log.Printf("[UserRepository] FindAll error: %v", err)
		return nil, errors.New("database error retrieving all users")
	}
	return users, nil
}
// --- END OF NEW FUNCTION ---

// NEW: handleFindError is a helper to reduce repetitive error handling code.
func (r *gormUserRepository) handleFindError(err error, user *domain.User, methodName string, identifier interface{}) (*domain.User, error) {
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound // Return our specific error
		}
		log.Printf("[UserRepository] %s error for %v: %v", methodName, identifier, err)
		return nil, errors.New("database error finding user")
	}
	return user, nil
}