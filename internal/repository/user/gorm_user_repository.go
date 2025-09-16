// G:\go_internist\internal\repository\user\gorm_user_repository.go

package user

import (
    "context"
    "errors"
    "fmt"
    "log"
    "strings"
    "github.com/iyunix/go-internist/internal/domain"
    "gorm.io/gorm"
)



// FindByPhoneNumber - Enhanced with validation (alias for FindByPhone for compatibility)
func (r *gormUserRepository) FindByPhoneNumber(ctx context.Context, phoneNumber string) (*domain.User, error) {
    if err := r.validatePhone(phoneNumber); err != nil {
        return nil, fmt.Errorf("phone validation failed: %w", err)
    }
    
    var user domain.User
    err := r.db.WithContext(ctx).Where("phone_number = ?", phoneNumber).First(&user).Error
    return r.handleFindError(err, &user)
}


var ErrUserNotFound = errors.New("user not found")

type gormUserRepository struct {
    db *gorm.DB
}

func NewGormUserRepository(db *gorm.DB) UserRepository {
    return &gormUserRepository{db: db}
}

// ===== ORIGINAL METHODS WITH PRODUCTION ENHANCEMENTS =====

// Create - Enhanced with input validation and secure logging
func (r *gormUserRepository) Create(ctx context.Context, user *domain.User) (*domain.User, error) {
    // Input validation (SQL injection protection)
    if err := r.validateUserInput(user); err != nil {
        log.Printf("[UserRepository] Validation failed: %v", err)
        return nil, fmt.Errorf("validation failed: %w", err)
    }
    
    if err := r.db.WithContext(ctx).Create(user).Error; err != nil {
        // Secure logging - no sensitive data exposed
        log.Printf("[UserRepository] Database error during user creation: %v", err)
        return nil, errors.New("database error creating user")
    }
    
    // Success logging with safe information
    log.Printf("[UserRepository] User created successfully with ID: %d", user.ID)
    return user, nil
}

// Update - Enhanced with validation and secure logging
func (r *gormUserRepository) Update(ctx context.Context, user *domain.User) error {
    if user.ID == 0 {
        return errors.New("invalid user ID")
    }
    
    if err := r.validateUserInput(user); err != nil {
        return fmt.Errorf("validation failed: %w", err)
    }
    
    if err := r.db.WithContext(ctx).Save(user).Error; err != nil {
        log.Printf("[UserRepository] Database error during user update for ID %d: %v", user.ID, err)
        return errors.New("database error updating user")
    }
    
    log.Printf("[UserRepository] User updated successfully with ID: %d", user.ID)
    return nil
}

// FindByID - Enhanced with secure error handling
func (r *gormUserRepository) FindByID(ctx context.Context, id uint) (*domain.User, error) {
    if id == 0 {
        return nil, errors.New("invalid user ID")
    }
    
    var user domain.User
    err := r.db.WithContext(ctx).First(&user, id).Error
    return r.handleFindError(err, &user)
}

// FindByUsername - Enhanced with input validation
func (r *gormUserRepository) FindByUsername(ctx context.Context, username string) (*domain.User, error) {
    if err := r.validateUsername(username); err != nil {
        return nil, fmt.Errorf("validation failed: %w", err)
    }
    
    var user domain.User
    err := r.db.WithContext(ctx).Where("username = ?", username).First(&user).Error
    return r.handleFindError(err, &user)
}

// FindByUsernameOrPhone - Enhanced with validation
func (r *gormUserRepository) FindByUsernameOrPhone(ctx context.Context, username, phone string) (*domain.User, error) {
    if err := r.validateUsername(username); err != nil {
        return nil, fmt.Errorf("username validation failed: %w", err)
    }
    if err := r.validatePhone(phone); err != nil {
        return nil, fmt.Errorf("phone validation failed: %w", err)
    }
    
    var user domain.User
    err := r.db.WithContext(ctx).Where("username = ? OR phone_number = ?", username, phone).First(&user).Error
    return r.handleFindError(err, &user)
}

// FindByPhoneAndStatus - Enhanced with validation
func (r *gormUserRepository) FindByPhoneAndStatus(ctx context.Context, phone string, status domain.UserStatus) (*domain.User, error) {
    if err := r.validatePhone(phone); err != nil {
        return nil, fmt.Errorf("phone validation failed: %w", err)
    }
    
    var user domain.User
    err := r.db.WithContext(ctx).Where("phone_number = ? AND status = ?", phone, status).First(&user).Error
    return r.handleFindError(err, &user)
}

// FindByPhone - Enhanced with validation
func (r *gormUserRepository) FindByPhone(ctx context.Context, phone string) (*domain.User, error) {
    if err := r.validatePhone(phone); err != nil {
        return nil, fmt.Errorf("phone validation failed: %w", err)
    }
    
    var user domain.User
    err := r.db.WithContext(ctx).Where("phone_number = ?", phone).First(&user).Error
    return r.handleFindError(err, &user)
}

// ResetFailedAttempts - Enhanced with validation and atomic operation
func (r *gormUserRepository) ResetFailedAttempts(ctx context.Context, id uint) error {
    if id == 0 {
        return errors.New("invalid user ID")
    }
    
    result := r.db.WithContext(ctx).Model(&domain.User{}).
        Where("id = ?", id).
        Update("failed_login_attempts", 0)
    
    if result.Error != nil {
        log.Printf("[UserRepository] Database error resetting failed attempts for user ID %d: %v", id, result.Error)
        return errors.New("database error resetting failed attempts")
    }
    
    if result.RowsAffected == 0 {
        return ErrUserNotFound
    }
    
    return nil
}

// Delete - Enhanced with validation and secure logging
func (r *gormUserRepository) Delete(ctx context.Context, userID uint) error {
    if userID == 0 {
        return errors.New("invalid user ID")
    }
    
    result := r.db.WithContext(ctx).Delete(&domain.User{}, userID)
    if result.Error != nil {
        log.Printf("[UserRepository] Database error deleting user ID %d: %v", userID, result.Error)
        return errors.New("database error deleting user")
    }
    
    if result.RowsAffected == 0 {
        return ErrUserNotFound
    }
    
    log.Printf("[UserRepository] User deleted successfully with ID: %d", userID)
    return nil
}

// GetCharacterBalance - Enhanced with validation
func (r *gormUserRepository) GetCharacterBalance(ctx context.Context, userID uint) (int, error) {
    if userID == 0 {
        return 0, errors.New("invalid user ID")
    }
    
    var balance int
    err := r.db.WithContext(ctx).Model(&domain.User{}).
        Select("character_balance").
        Where("id = ?", userID).
        Scan(&balance).Error
    
    if err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return 0, ErrUserNotFound
        }
        log.Printf("[UserRepository] Database error getting balance for user ID %d: %v", userID, err)
        return 0, errors.New("database error getting character balance")
    }
    
    return balance, nil
}

// UpdateCharacterBalance - Enhanced with atomic operation and validation
func (r *gormUserRepository) UpdateCharacterBalance(ctx context.Context, userID uint, newBalance int) error {
    if userID == 0 {
        return errors.New("invalid user ID")
    }
    
    if newBalance < 0 {
        return errors.New("balance cannot be negative")
    }
    
    result := r.db.WithContext(ctx).Model(&domain.User{}).
        Where("id = ?", userID).
        Update("character_balance", newBalance)
    
    if result.Error != nil {
        log.Printf("[UserRepository] Database error updating balance for user ID %d: %v", userID, result.Error)
        return errors.New("database error updating character balance")
    }
    
    if result.RowsAffected == 0 {
        return ErrUserNotFound
    }
    
    return nil
}

// FindAll - Enhanced with memory safety warning (deprecated in favor of pagination)
func (r *gormUserRepository) FindAll(ctx context.Context) ([]domain.User, error) {
    log.Printf("[UserRepository] WARNING: FindAll() loads all users into memory. Use FindAllWithPagination() for production.")
    
    var users []domain.User
    err := r.db.WithContext(ctx).Find(&users).Error
    if err != nil {
        log.Printf("[UserRepository] Database error finding all users: %v", err)
        return nil, errors.New("database error retrieving users")
    }
    
    return users, nil
}

// ===== NEW PRODUCTION-READY METHODS =====

// FindAllWithPagination - Memory safety: prevents OOM with large datasets
func (r *gormUserRepository) FindAllWithPagination(ctx context.Context, limit, offset int) ([]domain.User, int64, error) {
    var users []domain.User
    var total int64
    
    // Memory safety: enforce maximum limit
    if limit <= 0 || limit > 1000 {
        return nil, 0, errors.New("invalid limit: must be between 1 and 1000")
    }
    if offset < 0 {
        return nil, 0, errors.New("invalid offset: must be >= 0")
    }
    
    // Efficient counting without loading data
    if err := r.db.WithContext(ctx).Model(&domain.User{}).Count(&total).Error; err != nil {
        log.Printf("[UserRepository] Database error counting users: %v", err)
        return nil, 0, errors.New("database error counting users")
    }
    
    // Load only requested page
    err := r.db.WithContext(ctx).
        Order("id asc").
        Limit(limit).
        Offset(offset).
        Find(&users).Error
    
    if err != nil {
        log.Printf("[UserRepository] Database error in paginated query: %v", err)
        return nil, 0, errors.New("database error retrieving paginated users")
    }
    
    return users, total, nil
}

// CreateInBatch - Performance optimization: bulk operations
func (r *gormUserRepository) CreateInBatch(ctx context.Context, users []*domain.User, batchSize int) error {
    if len(users) == 0 {
        return nil
    }
    
    // Optimize batch size
    if batchSize <= 0 || batchSize > 1000 {
        batchSize = 100 // Sweet spot for most databases
    }
    
    // Pre-validate ALL users (fail fast)
    for i, user := range users {
        if err := r.validateUserInput(user); err != nil {
            return fmt.Errorf("validation failed for user %d: %w", i, err)
        }
    }
    
    // Process in optimized batches
    for i := 0; i < len(users); i += batchSize {
        end := i + batchSize
        if end > len(users) {
            end = len(users)
        }
        
        batch := users[i:end]
        if err := r.db.WithContext(ctx).CreateInBatches(batch, batchSize).Error; err != nil {
            log.Printf("[UserRepository] Batch creation failed for batch %d-%d: %v", i, end, err)
            return fmt.Errorf("database error creating batch %d-%d: %w", i, end, err)
        }
    }
    
    log.Printf("[UserRepository] Successfully created %d users in batches", len(users))
    return nil
}

// ExistsByUsername - Security: check without exposing data
func (r *gormUserRepository) ExistsByUsername(ctx context.Context, username string) (bool, error) {
    if err := r.validateUsername(username); err != nil {
        return false, err
    }
    
    var count int64
    err := r.db.WithContext(ctx).Model(&domain.User{}).Where("username = ?", username).Count(&count).Error
    if err != nil {
        log.Printf("[UserRepository] Database error checking username existence: %v", err)
        return false, errors.New("database error checking username existence")
    }
    
    return count > 0, nil
}

// ExistsByPhone - Security: check without exposing data
func (r *gormUserRepository) ExistsByPhone(ctx context.Context, phone string) (bool, error) {
    if err := r.validatePhone(phone); err != nil {
        return false, err
    }
    
    var count int64
    err := r.db.WithContext(ctx).Model(&domain.User{}).Where("phone_number = ?", phone).Count(&count).Error
    if err != nil {
        log.Printf("[UserRepository] Database error checking phone existence: %v", err)
        return false, errors.New("database error checking phone existence")
    }
    
    return count > 0, nil
}

// CountUsers - Performance: efficient counting
func (r *gormUserRepository) CountUsers(ctx context.Context) (int64, error) {
    var count int64
    err := r.db.WithContext(ctx).Model(&domain.User{}).Count(&count).Error
    if err != nil {
        log.Printf("[UserRepository] Database error counting users: %v", err)
        return 0, errors.New("database error counting users")
    }
    return count, nil
}

// CountActiveUsers - Performance: efficient filtered counting
func (r *gormUserRepository) CountActiveUsers(ctx context.Context) (int64, error) {
    var count int64
    err := r.db.WithContext(ctx).Model(&domain.User{}).Where("status = ?", "active").Count(&count).Error
    if err != nil {
        log.Printf("[UserRepository] Database error counting active users: %v", err)
        return 0, errors.New("database error counting active users")
    }
    return count, nil
}

// IncrementFailedAttempts - Security: rate limiting support
func (r *gormUserRepository) IncrementFailedAttempts(ctx context.Context, userID uint) error {
    if userID == 0 {
        return errors.New("invalid user ID")
    }
    
    result := r.db.WithContext(ctx).Model(&domain.User{}).
        Where("id = ?", userID).
        Update("failed_login_attempts", gorm.Expr("failed_login_attempts + 1"))
    
    if result.Error != nil {
        log.Printf("[UserRepository] Database error incrementing failed attempts for user ID %d: %v", userID, result.Error)
        return errors.New("database error incrementing failed attempts")
    }
    
    if result.RowsAffected == 0 {
        return ErrUserNotFound
    }
    
    return nil
}

// UpdateMultipleBalances - Data integrity: transaction support
func (r *gormUserRepository) UpdateMultipleBalances(ctx context.Context, updates []domain.BalanceUpdate) error {
    if len(updates) == 0 {
        return nil
    }
    
    // Use transaction for atomicity
    return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
        for _, update := range updates {
            if update.UserID == 0 {
                return errors.New("invalid user ID in balance update")
            }
            
            result := tx.Model(&domain.User{}).
                Where("id = ?", update.UserID).
                Update("character_balance", gorm.Expr("character_balance + ?", update.Amount))
            
            if result.Error != nil {
                return fmt.Errorf("failed to update balance for user %d: %w", update.UserID, result.Error)
            }
            
            if result.RowsAffected == 0 {
                return fmt.Errorf("user %d not found for balance update", update.UserID)
            }
        }
        return nil
    })
}

// ===== SECURITY VALIDATION HELPERS =====

// validateUserInput - Comprehensive input validation
func (r *gormUserRepository) validateUserInput(user *domain.User) error {
    if user == nil {
        return errors.New("user cannot be nil")
    }
    
    if err := r.validateUsername(user.Username); err != nil {
        return fmt.Errorf("username validation: %w", err)
    }
    
    if err := r.validatePhone(user.PhoneNumber); err != nil {
        return fmt.Errorf("phone validation: %w", err)
    }
    
    return nil
}

// validateUsername - SQL injection protection
func (r *gormUserRepository) validateUsername(username string) error {
    if len(username) < 3 || len(username) > 50 {
        return errors.New("username must be between 3 and 50 characters")
    }
    
    // Detect malicious patterns
    maliciousPatterns := []string{
        "--", "/*", "*/", "xp_", "sp_", 
        "union", "select", "insert", "delete", "drop", "create", "alter",
        "<script", "javascript:", "vbscript:",
    }
    
    lowerUsername := strings.ToLower(username)
    for _, pattern := range maliciousPatterns {
        if strings.Contains(lowerUsername, pattern) {
            return errors.New("invalid characters detected in username")
        }
    }
    
    return nil
}

// validatePhone - Phone number validation
func (r *gormUserRepository) validatePhone(phone string) error {
    if len(phone) < 10 || len(phone) > 15 {
        return errors.New("phone number must be between 10 and 15 digits")
    }
    
    // Check for valid phone characters
    for _, char := range phone {
        if char != '+' && char != '-' && char != ' ' && char != '(' && char != ')' && (char < '0' || char > '9') {
            return errors.New("phone number contains invalid characters")
        }
    }
    
    return nil
}

// ===== ERROR HANDLING HELPERS =====

// handleFindError - Secure error handling without data leakage
func (r *gormUserRepository) handleFindError(err error, user *domain.User) (*domain.User, error) {
    if err == nil {
        return user, nil
    }
    
    if errors.Is(err, gorm.ErrRecordNotFound) {
        return nil, ErrUserNotFound
    }
    
    // Log technical details for debugging
    log.Printf("[UserRepository] Database query error: %v", err)
    
    // Return generic error for security
    return nil, errors.New("database query failed")
}

// FindAllWithPaginationAndSearch provides a memory-safe, paginated, and searchable query for users.
func (r *gormUserRepository) FindAllWithPaginationAndSearch(ctx context.Context, page, limit int, search string) ([]domain.User, int64, error) {
	var users []domain.User
	var total int64

	// Build the base query
	query := r.db.WithContext(ctx).Model(&domain.User{})

	// Apply search filter if a search term is provided
	if search != "" {
		// Sanitize search term
		searchTerm := "%" + strings.ToLower(search) + "%"
		query = query.Where("LOWER(username) LIKE ? OR phone_number LIKE ?", searchTerm, searchTerm)
	}

	// First, count the total number of records that match the query
	if err := query.Count(&total).Error; err != nil {
		log.Printf("[UserRepository] Database error counting users with search: %v", err)
		return nil, 0, errors.New("database error counting users")
	}

	// Now, apply pagination (limit and offset) to the query to fetch the actual data
	offset := (page - 1) * limit
	err := query.Order("id asc").Limit(limit).Offset(offset).Find(&users).Error
	if err != nil {
		log.Printf("[UserRepository] Database error in paginated search query: %v", err)
		return nil, 0, errors.New("database error retrieving paginated users")
	}

	return users, total, nil
}
