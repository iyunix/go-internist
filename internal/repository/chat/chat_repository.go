// G:\go_internist\internal\repository\chat\chat_repository.go

package chat

import (
    "context"
    "errors"
    "fmt"
    "log"
    "strings"
    "time"
    "github.com/iyunix/go-internist/internal/domain"
    "gorm.io/gorm"
)

var ErrChatNotFound = errors.New("chat not found")
var ErrUnauthorizedAccess = errors.New("unauthorized access to chat")

type gormChatRepository struct {
    db *gorm.DB
}

func NewChatRepository(db *gorm.DB) ChatRepository {
    return &gormChatRepository{db: db}
}

// ===== ORIGINAL METHODS WITH PRODUCTION ENHANCEMENTS =====

// Create - Enhanced with input validation and secure logging
func (r *gormChatRepository) Create(ctx context.Context, chat *domain.Chat) (*domain.Chat, error) {
    // Input validation
    if err := r.validateChatInput(chat); err != nil {
        log.Printf("[ChatRepository] Validation failed: %v", err)
        return nil, fmt.Errorf("validation failed: %w", err)
    }
    
    err := r.db.WithContext(ctx).Create(chat).Error
    if err != nil {
        // Secure logging - no sensitive data exposed
        log.Printf("[ChatRepository] Database error during chat creation for user ID %d: %v", chat.UserID, err)
        return nil, errors.New("database error creating chat")
    }
    
    log.Printf("[ChatRepository] Chat created successfully with ID: %d for user: %d", chat.ID, chat.UserID)
    return chat, nil
}

// FindByID - Enhanced with secure error handling
func (r *gormChatRepository) FindByID(ctx context.Context, chatID uint) (*domain.Chat, error) {
    if chatID == 0 {
        return nil, errors.New("invalid chat ID")
    }
    
    var chat domain.Chat
    err := r.db.WithContext(ctx).First(&chat, chatID).Error
    return r.handleFindError(err, &chat, "FindByID")
}

// FindByUserID - Enhanced with memory safety warning (deprecated)
func (r *gormChatRepository) FindByUserID(ctx context.Context, userID uint) ([]domain.Chat, error) {
    log.Printf("[ChatRepository] WARNING: FindByUserID() loads all chats into memory. Use FindByUserIDWithPagination() for production.")
    
    if userID == 0 {
        return nil, errors.New("invalid user ID")
    }
    
    var chats []domain.Chat
    err := r.db.WithContext(ctx).
        Where("user_id = ?", userID).
        Order("updated_at DESC, id DESC").
        Find(&chats).Error
    
    if err != nil {
        log.Printf("[ChatRepository] Database error finding chats for user ID %d: %v", userID, err)
        return nil, errors.New("database error fetching chats")
    }
    
    return chats, nil
}

// Delete - Enhanced with validation and secure logging
func (r *gormChatRepository) Delete(ctx context.Context, chatID, userID uint) error {
    if chatID == 0 || userID == 0 {
        return errors.New("invalid chat ID or user ID")
    }
    
    result := r.db.WithContext(ctx).
        Where("id = ? AND user_id = ?", chatID, userID).
        Delete(&domain.Chat{})
    
    if result.Error != nil {
        log.Printf("[ChatRepository] Database error deleting chat ID %d for user ID %d: %v", chatID, userID, result.Error)
        return errors.New("database error deleting chat")
    }
    
    if result.RowsAffected == 0 {
        return ErrUnauthorizedAccess
    }
    
    log.Printf("[ChatRepository] Chat deleted successfully: ID %d for user %d", chatID, userID)
    return nil
}

// TouchUpdatedAt - Enhanced with validation and error handling
func (r *gormChatRepository) TouchUpdatedAt(ctx context.Context, chatID uint) error {
    if chatID == 0 {
        return errors.New("invalid chat ID")
    }
    
    result := r.db.WithContext(ctx).
        Model(&domain.Chat{}).
        Where("id = ?", chatID).
        Update("updated_at", gorm.Expr("CURRENT_TIMESTAMP"))
    
    if result.Error != nil {
        log.Printf("[ChatRepository] Database error updating timestamp for chat ID %d: %v", chatID, result.Error)
        return errors.New("database error updating chat timestamp")
    }
    
    if result.RowsAffected == 0 {
        return ErrChatNotFound
    }
    
    return nil
}

// ===== NEW PRODUCTION-READY METHODS =====

// FindByUserIDWithPagination - Memory safety: prevents OOM with large chat histories
func (r *gormChatRepository) FindByUserIDWithPagination(ctx context.Context, userID uint, limit, offset int) ([]domain.Chat, int64, error) {
    if userID == 0 {
        return nil, 0, errors.New("invalid user ID")
    }
    
    // Memory safety: enforce maximum limit
    if limit <= 0 || limit > 1000 {
        return nil, 0, errors.New("invalid limit: must be between 1 and 1000")
    }
    if offset < 0 {
        return nil, 0, errors.New("invalid offset: must be >= 0")
    }
    
    var chats []domain.Chat
    var total int64
    
    // Efficient counting without loading data
    if err := r.db.WithContext(ctx).Model(&domain.Chat{}).Where("user_id = ?", userID).Count(&total).Error; err != nil {
        log.Printf("[ChatRepository] Database error counting chats for user ID %d: %v", userID, err)
        return nil, 0, errors.New("database error counting chats")
    }
    
    // Load only requested page
    err := r.db.WithContext(ctx).
        Where("user_id = ?", userID).
        Order("updated_at DESC, id DESC").
        Limit(limit).
        Offset(offset).
        Find(&chats).Error
    
    if err != nil {
        log.Printf("[ChatRepository] Database error in paginated query for user ID %d: %v", userID, err)
        return nil, 0, errors.New("database error retrieving paginated chats")
    }
    
    return chats, total, nil
}

// CreateInBatch - Performance optimization: bulk chat creation
func (r *gormChatRepository) CreateInBatch(ctx context.Context, chats []*domain.Chat, batchSize int) error {
    if len(chats) == 0 {
        return nil
    }
    
    // Optimize batch size
    if batchSize <= 0 || batchSize > 1000 {
        batchSize = 100
    }
    
    // Pre-validate ALL chats (fail fast)
    for i, chat := range chats {
        if err := r.validateChatInput(chat); err != nil {
            return fmt.Errorf("validation failed for chat %d: %w", i, err)
        }
    }
    
    // Process in optimized batches
    for i := 0; i < len(chats); i += batchSize {
        end := i + batchSize
        if end > len(chats) {
            end = len(chats)
        }
        
        batch := chats[i:end]
        if err := r.db.WithContext(ctx).CreateInBatches(batch, batchSize).Error; err != nil {
            log.Printf("[ChatRepository] Batch creation failed for batch %d-%d: %v", i, end, err)
            return fmt.Errorf("database error creating batch %d-%d: %w", i, end, err)
        }
    }
    
    log.Printf("[ChatRepository] Successfully created %d chats in batches", len(chats))
    return nil
}

// DeleteMultipleByUserID - Performance: bulk deletion with security
func (r *gormChatRepository) DeleteMultipleByUserID(ctx context.Context, chatIDs []uint, userID uint) error {
    if len(chatIDs) == 0 {
        return nil
    }
    
    if userID == 0 {
        return errors.New("invalid user ID")
    }
    
    // Validate all chat IDs
    for _, chatID := range chatIDs {
        if chatID == 0 {
            return errors.New("invalid chat ID in batch")
        }
    }
    
    result := r.db.WithContext(ctx).
        Where("id IN ? AND user_id = ?", chatIDs, userID).
        Delete(&domain.Chat{})
    
    if result.Error != nil {
        log.Printf("[ChatRepository] Database error in bulk delete for user ID %d: %v", userID, result.Error)
        return errors.New("database error in bulk chat deletion")
    }
    
    log.Printf("[ChatRepository] Bulk deleted %d chats for user %d", result.RowsAffected, userID)
    return nil
}

// ExistsByID - Security: check existence without data exposure
func (r *gormChatRepository) ExistsByID(ctx context.Context, chatID uint) (bool, error) {
    if chatID == 0 {
        return false, errors.New("invalid chat ID")
    }
    
    var count int64
    err := r.db.WithContext(ctx).Model(&domain.Chat{}).Where("id = ?", chatID).Count(&count).Error
    if err != nil {
        log.Printf("[ChatRepository] Database error checking chat existence for ID %d: %v", chatID, err)
        return false, errors.New("database error checking chat existence")
    }
    
    return count > 0, nil
}

// ExistsByIDAndUserID - Security: ownership validation
func (r *gormChatRepository) ExistsByIDAndUserID(ctx context.Context, chatID, userID uint) (bool, error) {
    if chatID == 0 || userID == 0 {
        return false, errors.New("invalid chat ID or user ID")
    }
    
    var count int64
    err := r.db.WithContext(ctx).Model(&domain.Chat{}).Where("id = ? AND user_id = ?", chatID, userID).Count(&count).Error
    if err != nil {
        log.Printf("[ChatRepository] Database error checking chat ownership for chat ID %d, user ID %d: %v", chatID, userID, err)
        return false, errors.New("database error checking chat ownership")
    }
    
    return count > 0, nil
}

// VerifyOwnership - Security: explicit ownership verification
func (r *gormChatRepository) VerifyOwnership(ctx context.Context, chatID, userID uint) (bool, error) {
    return r.ExistsByIDAndUserID(ctx, chatID, userID)
}

// CountByUserID - Performance: efficient user chat counting
func (r *gormChatRepository) CountByUserID(ctx context.Context, userID uint) (int64, error) {
    if userID == 0 {
        return 0, errors.New("invalid user ID")
    }
    
    var count int64
    err := r.db.WithContext(ctx).Model(&domain.Chat{}).Where("user_id = ?", userID).Count(&count).Error
    if err != nil {
        log.Printf("[ChatRepository] Database error counting chats for user ID %d: %v", userID, err)
        return 0, errors.New("database error counting user chats")
    }
    
    return count, nil
}

// CountTotalChats - Performance: system-wide metrics
func (r *gormChatRepository) CountTotalChats(ctx context.Context) (int64, error) {
    var count int64
    err := r.db.WithContext(ctx).Model(&domain.Chat{}).Count(&count).Error
    if err != nil {
        log.Printf("[ChatRepository] Database error counting total chats: %v", err)
        return 0, errors.New("database error counting total chats")
    }
    
    return count, nil
}

// CountActiveChats - Analytics: activity-based metrics
func (r *gormChatRepository) CountActiveChats(ctx context.Context, since time.Time) (int64, error) {
    var count int64
    err := r.db.WithContext(ctx).Model(&domain.Chat{}).Where("updated_at >= ?", since).Count(&count).Error
    if err != nil {
        log.Printf("[ChatRepository] Database error counting active chats since %v: %v", since, err)
        return 0, errors.New("database error counting active chats")
    }
    
    return count, nil
}

// FindRecentChats - Analytics: recent activity tracking
func (r *gormChatRepository) FindRecentChats(ctx context.Context, userID uint, limit int) ([]domain.Chat, error) {
    if userID == 0 {
        return nil, errors.New("invalid user ID")
    }
    
    if limit <= 0 || limit > 100 {
        limit = 10 // Safe default
    }
    
    var chats []domain.Chat
    err := r.db.WithContext(ctx).
        Where("user_id = ?", userID).
        Order("updated_at DESC").
        Limit(limit).
        Find(&chats).Error
    
    if err != nil {
        log.Printf("[ChatRepository] Database error finding recent chats for user ID %d: %v", userID, err)
        return nil, errors.New("database error finding recent chats")
    }
    
    return chats, nil
}

// FindChatsByDateRange - Analytics: historical analysis
func (r *gormChatRepository) FindChatsByDateRange(ctx context.Context, userID uint, startDate, endDate time.Time) ([]domain.Chat, error) {
    if userID == 0 {
        return nil, errors.New("invalid user ID")
    }
    
    if startDate.After(endDate) {
        return nil, errors.New("start date must be before end date")
    }
    
    var chats []domain.Chat
    err := r.db.WithContext(ctx).
        Where("user_id = ? AND created_at BETWEEN ? AND ?", userID, startDate, endDate).
        Order("created_at DESC").
        Find(&chats).Error
    
    if err != nil {
        log.Printf("[ChatRepository] Database error finding chats by date range for user ID %d: %v", userID, err)
        return nil, errors.New("database error finding chats by date range")
    }
    
    return chats, nil
}

// FindOldestChats - Analytics: historical data management
func (r *gormChatRepository) FindOldestChats(ctx context.Context, userID uint, limit int) ([]domain.Chat, error) {
    if userID == 0 {
        return nil, errors.New("invalid user ID")
    }
    
    if limit <= 0 || limit > 1000 {
        limit = 100
    }
    
    var chats []domain.Chat
    err := r.db.WithContext(ctx).
        Where("user_id = ?", userID).
        Order("created_at ASC").
        Limit(limit).
        Find(&chats).Error
    
    if err != nil {
        log.Printf("[ChatRepository] Database error finding oldest chats for user ID %d: %v", userID, err)
        return nil, errors.New("database error finding oldest chats")
    }
    
    return chats, nil
}

// DeleteOldChats - Maintenance: automated cleanup
func (r *gormChatRepository) DeleteOldChats(ctx context.Context, userID uint, olderThan time.Time) (int64, error) {
    if userID == 0 {
        return 0, errors.New("invalid user ID")
    }
    
    result := r.db.WithContext(ctx).
        Where("user_id = ? AND created_at < ?", userID, olderThan).
        Delete(&domain.Chat{})
    
    if result.Error != nil {
        log.Printf("[ChatRepository] Database error deleting old chats for user ID %d: %v", userID, result.Error)
        return 0, errors.New("database error deleting old chats")
    }
    
    log.Printf("[ChatRepository] Deleted %d old chats for user %d", result.RowsAffected, userID)
    return result.RowsAffected, nil
}

// ArchiveInactiveChats - Maintenance: system-wide cleanup
func (r *gormChatRepository) ArchiveInactiveChats(ctx context.Context, inactiveSince time.Time) (int64, error) {
    // Implementation depends on your archival strategy
    // For now, we'll mark them as archived or move to archive table
    result := r.db.WithContext(ctx).
        Model(&domain.Chat{}).
        Where("updated_at < ?", inactiveSince).
        Update("archived", true)
    
    if result.Error != nil {
        log.Printf("[ChatRepository] Database error archiving inactive chats: %v", result.Error)
        return 0, errors.New("database error archiving inactive chats")
    }
    
    log.Printf("[ChatRepository] Archived %d inactive chats", result.RowsAffected)
    return result.RowsAffected, nil
}

// UpdateMultipleTimestamps - Data integrity: bulk timestamp updates
func (r *gormChatRepository) UpdateMultipleTimestamps(ctx context.Context, chatIDs []uint) error {
    if len(chatIDs) == 0 {
        return nil
    }
    
    // Validate all chat IDs
    for _, chatID := range chatIDs {
        if chatID == 0 {
            return errors.New("invalid chat ID in timestamp update")
        }
    }
    
    result := r.db.WithContext(ctx).
        Model(&domain.Chat{}).
        Where("id IN ?", chatIDs).
        Update("updated_at", gorm.Expr("CURRENT_TIMESTAMP"))
    
    if result.Error != nil {
        log.Printf("[ChatRepository] Database error in bulk timestamp update: %v", result.Error)
        return errors.New("database error updating multiple timestamps")
    }
    
    log.Printf("[ChatRepository] Updated timestamps for %d chats", result.RowsAffected)
    return nil
}

// SearchChatsByTitle - Search: title-based chat organization
func (r *gormChatRepository) SearchChatsByTitle(ctx context.Context, userID uint, titlePattern string, limit int) ([]domain.Chat, error) {
    if userID == 0 {
        return nil, errors.New("invalid user ID")
    }
    
    if err := r.validateSearchPattern(titlePattern); err != nil {
        return nil, fmt.Errorf("invalid search pattern: %w", err)
    }
    
    if limit <= 0 || limit > 100 {
        limit = 20
    }
    
    var chats []domain.Chat
    searchPattern := fmt.Sprintf("%%%s%%", titlePattern)
    
    err := r.db.WithContext(ctx).
        Where("user_id = ? AND title LIKE ?", userID, searchPattern).
        Order("updated_at DESC").
        Limit(limit).
        Find(&chats).Error
    
    if err != nil {
        log.Printf("[ChatRepository] Database error searching chats by title for user ID %d: %v", userID, err)
        return nil, errors.New("database error searching chats")
    }
    
    return chats, nil
}

// ===== SECURITY VALIDATION HELPERS =====

// validateChatInput - Comprehensive input validation
func (r *gormChatRepository) validateChatInput(chat *domain.Chat) error {
    if chat == nil {
        return errors.New("chat cannot be nil")
    }
    
    if chat.UserID == 0 {
        return errors.New("user ID is required")
    }
    
    if err := r.validateChatTitle(chat.Title); err != nil {
        return fmt.Errorf("title validation: %w", err)
    }
    
    return nil
}

// validateChatTitle - Title validation with security checks
func (r *gormChatRepository) validateChatTitle(title string) error {
    if len(title) > 200 {
        return errors.New("title must be 200 characters or less")
    }
    
    // Basic XSS protection
    if strings.Contains(title, "<script") || strings.Contains(title, "javascript:") {
        return errors.New("invalid characters detected in title")
    }
    
    return nil
}

// validateSearchPattern - Search pattern validation
func (r *gormChatRepository) validateSearchPattern(pattern string) error {
    if len(pattern) > 100 {
        return errors.New("search pattern too long")
    }
    
    // Prevent SQL injection in LIKE queries
    if strings.Contains(pattern, "%") || strings.Contains(pattern, "_") || strings.Contains(pattern, "\\") {
        return errors.New("invalid characters in search pattern")
    }
    
    return nil
}

// ===== ERROR HANDLING HELPERS =====

// handleFindError - Secure error handling without data leakage
func (r *gormChatRepository) handleFindError(err error, chat *domain.Chat, operation string) (*domain.Chat, error) {
    if err == nil {
        return chat, nil
    }
    
    if errors.Is(err, gorm.ErrRecordNotFound) {
        return nil, ErrChatNotFound
    }
    
    // Log technical details for debugging
    log.Printf("[ChatRepository] %s database error: %v", operation, err)
    
    // Return generic error for security
    return nil, errors.New("database query failed")
}
