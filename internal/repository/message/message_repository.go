// G:\go_internist\internal\repository\message\message_repository.go

package message

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

var ErrMessageNotFound = errors.New("message not found")
var ErrUnauthorizedMessageAccess = errors.New("unauthorized access to message")

type gormMessageRepository struct {
    db *gorm.DB
}

func NewMessageRepository(db *gorm.DB) MessageRepository {
    return &gormMessageRepository{db: db}
}

// ===== ORIGINAL METHODS WITH PRODUCTION ENHANCEMENTS =====

// Create - Enhanced with comprehensive input validation and secure logging
func (r *gormMessageRepository) Create(ctx context.Context, message *domain.Message) (*domain.Message, error) {
    // Comprehensive input validation
    if err := r.validateMessageInput(message); err != nil {
        log.Printf("[MessageRepository] Validation failed: %v", err)
        return nil, fmt.Errorf("validation failed: %w", err)
    }
    
    err := r.db.WithContext(ctx).Create(message).Error
    if err != nil {
        // Secure logging - no sensitive medical content exposed
        log.Printf("[MessageRepository] Database error during message creation for chat ID %d: %v", message.ChatID, err)
        return nil, errors.New("database error creating message")
    }
    
    log.Printf("[MessageRepository] Message created successfully with ID: %d for chat: %d", message.ID, message.ChatID)
    return message, nil
}

// FindByChatID - Enhanced with memory safety warning (deprecated)
func (r *gormMessageRepository) FindByChatID(ctx context.Context, chatID uint) ([]domain.Message, error) {
    log.Printf("[MessageRepository] WARNING: FindByChatID() loads all messages into memory. Use FindByChatIDWithPagination() for production.")
    
    if chatID == 0 {
        return nil, errors.New("invalid chat ID")
    }
    
    var messages []domain.Message
    err := r.db.WithContext(ctx).
        Where("chat_id = ?", chatID).
        Order("created_at asc").
        Find(&messages).Error
    
    if err != nil {
        log.Printf("[MessageRepository] Database error finding messages for chat ID %d: %v", chatID, err)
        return nil, errors.New("database error fetching messages")
    }
    
    return messages, nil
}

// ===== NEW PRODUCTION-READY METHODS =====

// FindByChatIDWithPagination - Memory safety: prevents OOM with large conversations
func (r *gormMessageRepository) FindByChatIDWithPagination(ctx context.Context, chatID uint, limit, offset int) ([]domain.Message, int64, error) {
    if chatID == 0 {
        return nil, 0, errors.New("invalid chat ID")
    }
    
    // Memory safety: enforce maximum limit
    if limit <= 0 || limit > 1000 {
        return nil, 0, errors.New("invalid limit: must be between 1 and 1000")
    }
    if offset < 0 {
        return nil, 0, errors.New("invalid offset: must be >= 0")
    }
    
    var messages []domain.Message
    var total int64
    
    // Efficient counting without loading data
    if err := r.db.WithContext(ctx).Model(&domain.Message{}).Where("chat_id = ?", chatID).Count(&total).Error; err != nil {
        log.Printf("[MessageRepository] Database error counting messages for chat ID %d: %v", chatID, err)
        return nil, 0, errors.New("database error counting messages")
    }
    
    // Load only requested page
    err := r.db.WithContext(ctx).
        Where("chat_id = ?", chatID).
        Order("created_at asc").
        Limit(limit).
        Offset(offset).
        Find(&messages).Error
    
    if err != nil {
        log.Printf("[MessageRepository] Database error in paginated query for chat ID %d: %v", chatID, err)
        return nil, 0, errors.New("database error retrieving paginated messages")
    }
    
    return messages, total, nil
}

// FindByID - Complete CRUD operation
func (r *gormMessageRepository) FindByID(ctx context.Context, messageID uint) (*domain.Message, error) {
    if messageID == 0 {
        return nil, errors.New("invalid message ID")
    }
    
    var message domain.Message
    err := r.db.WithContext(ctx).First(&message, messageID).Error
    return r.handleFindError(err, &message, "FindByID")
}

// Update - Complete CRUD operation with validation
func (r *gormMessageRepository) Update(ctx context.Context, message *domain.Message) error {
    if message.ID == 0 {
        return errors.New("invalid message ID")
    }
    
    if err := r.validateMessageInput(message); err != nil {
        return fmt.Errorf("validation failed: %w", err)
    }
    
    result := r.db.WithContext(ctx).Save(message)
    if result.Error != nil {
        log.Printf("[MessageRepository] Database error updating message ID %d: %v", message.ID, result.Error)
        return errors.New("database error updating message")
    }
    
    if result.RowsAffected == 0 {
        return ErrMessageNotFound
    }
    
    log.Printf("[MessageRepository] Message updated successfully with ID: %d", message.ID)
    return nil
}

// Delete - Complete CRUD operation with security
func (r *gormMessageRepository) Delete(ctx context.Context, messageID, chatID uint) error {
    if messageID == 0 || chatID == 0 {
        return errors.New("invalid message ID or chat ID")
    }
    
    result := r.db.WithContext(ctx).
        Where("id = ? AND chat_id = ?", messageID, chatID).
        Delete(&domain.Message{})
    
    if result.Error != nil {
        log.Printf("[MessageRepository] Database error deleting message ID %d for chat ID %d: %v", messageID, chatID, result.Error)
        return errors.New("database error deleting message")
    }
    
    if result.RowsAffected == 0 {
        return ErrUnauthorizedMessageAccess
    }
    
    log.Printf("[MessageRepository] Message deleted successfully: ID %d for chat %d", messageID, chatID)
    return nil
}

// CreateInBatch - Performance optimization: bulk message creation
func (r *gormMessageRepository) CreateInBatch(ctx context.Context, messages []*domain.Message, batchSize int) error {
    if len(messages) == 0 {
        return nil
    }
    
    // Optimize batch size
    if batchSize <= 0 || batchSize > 1000 {
        batchSize = 100
    }
    
    // Pre-validate ALL messages (fail fast)
    for i, message := range messages {
        if err := r.validateMessageInput(message); err != nil {
            return fmt.Errorf("validation failed for message %d: %w", i, err)
        }
    }
    
    // Process in optimized batches
    for i := 0; i < len(messages); i += batchSize {
        end := i + batchSize
        if end > len(messages) {
            end = len(messages)
        }
        
        batch := messages[i:end]
        if err := r.db.WithContext(ctx).CreateInBatches(batch, batchSize).Error; err != nil {
            log.Printf("[MessageRepository] Batch creation failed for batch %d-%d: %v", i, end, err)
            return fmt.Errorf("database error creating batch %d-%d: %w", i, end, err)
        }
    }
    
    log.Printf("[MessageRepository] Successfully created %d messages in batches", len(messages))
    return nil
}

// DeleteMultipleByChatID - Performance: bulk deletion with security
func (r *gormMessageRepository) DeleteMultipleByChatID(ctx context.Context, messageIDs []uint, chatID uint) error {
    if len(messageIDs) == 0 {
        return nil
    }
    
    if chatID == 0 {
        return errors.New("invalid chat ID")
    }
    
    // Validate all message IDs
    for _, messageID := range messageIDs {
        if messageID == 0 {
            return errors.New("invalid message ID in batch")
        }
    }
    
    result := r.db.WithContext(ctx).
        Where("id IN ? AND chat_id = ?", messageIDs, chatID).
        Delete(&domain.Message{})
    
    if result.Error != nil {
        log.Printf("[MessageRepository] Database error in bulk delete for chat ID %d: %v", chatID, result.Error)
        return errors.New("database error in bulk message deletion")
    }
    
    log.Printf("[MessageRepository] Bulk deleted %d messages for chat %d", result.RowsAffected, chatID)
    return nil
}

// ExistsByID - Security: check existence without data exposure
func (r *gormMessageRepository) ExistsByID(ctx context.Context, messageID uint) (bool, error) {
    if messageID == 0 {
        return false, errors.New("invalid message ID")
    }
    
    var count int64
    err := r.db.WithContext(ctx).Model(&domain.Message{}).Where("id = ?", messageID).Count(&count).Error
    if err != nil {
        log.Printf("[MessageRepository] Database error checking message existence for ID %d: %v", messageID, err)
        return false, errors.New("database error checking message existence")
    }
    
    return count > 0, nil
}

// ExistsByIDAndChatID - Security: ownership validation
func (r *gormMessageRepository) ExistsByIDAndChatID(ctx context.Context, messageID, chatID uint) (bool, error) {
    if messageID == 0 || chatID == 0 {
        return false, errors.New("invalid message ID or chat ID")
    }
    
    var count int64
    err := r.db.WithContext(ctx).Model(&domain.Message{}).Where("id = ? AND chat_id = ?", messageID, chatID).Count(&count).Error
    if err != nil {
        log.Printf("[MessageRepository] Database error checking message ownership for message ID %d, chat ID %d: %v", messageID, chatID, err)
        return false, errors.New("database error checking message ownership")
    }
    
    return count > 0, nil
}

// VerifyMessageOwnership - Security: explicit ownership verification
func (r *gormMessageRepository) VerifyMessageOwnership(ctx context.Context, messageID, chatID uint) (bool, error) {
    return r.ExistsByIDAndChatID(ctx, messageID, chatID)
}

// CountByChatID - Performance: efficient message counting
func (r *gormMessageRepository) CountByChatID(ctx context.Context, chatID uint) (int64, error) {
    if chatID == 0 {
        return 0, errors.New("invalid chat ID")
    }
    
    var count int64
    err := r.db.WithContext(ctx).Model(&domain.Message{}).Where("chat_id = ?", chatID).Count(&count).Error
    if err != nil {
        log.Printf("[MessageRepository] Database error counting messages for chat ID %d: %v", chatID, err)
        return 0, errors.New("database error counting chat messages")
    }
    
    return count, nil
}

// CountTotalMessages - Performance: system-wide metrics
func (r *gormMessageRepository) CountTotalMessages(ctx context.Context) (int64, error) {
    var count int64
    err := r.db.WithContext(ctx).Model(&domain.Message{}).Count(&count).Error
    if err != nil {
        log.Printf("[MessageRepository] Database error counting total messages: %v", err)
        return 0, errors.New("database error counting total messages")
    }
    
    return count, nil
}

// CountMessagesByType - Analytics: type-based metrics
func (r *gormMessageRepository) CountMessagesByType(ctx context.Context, chatID uint, messageType string) (int64, error) {
    if chatID == 0 {
        return 0, errors.New("invalid chat ID")
    }
    
    if err := r.validateMessageType(messageType); err != nil {
        return 0, err
    }
    
    var count int64
    err := r.db.WithContext(ctx).Model(&domain.Message{}).
        Where("chat_id = ? AND message_type = ?", chatID, messageType).
        Count(&count).Error
    
    if err != nil {
        log.Printf("[MessageRepository] Database error counting messages by type for chat ID %d: %v", chatID, err)
        return 0, errors.New("database error counting messages by type")
    }
    
    return count, nil
}

// FindRecentMessages - Analytics: recent activity tracking
func (r *gormMessageRepository) FindRecentMessages(ctx context.Context, chatID uint, limit int) ([]domain.Message, error) {
    if chatID == 0 {
        return nil, errors.New("invalid chat ID")
    }
    
    if limit <= 0 || limit > 100 {
        limit = 10 // Safe default
    }
    
    var messages []domain.Message
    err := r.db.WithContext(ctx).
        Where("chat_id = ?", chatID).
        Order("created_at DESC").
        Limit(limit).
        Find(&messages).Error
    
    if err != nil {
        log.Printf("[MessageRepository] Database error finding recent messages for chat ID %d: %v", chatID, err)
        return nil, errors.New("database error finding recent messages")
    }
    
    return messages, nil
}

// FindMessagesByDateRange - Analytics: historical analysis
func (r *gormMessageRepository) FindMessagesByDateRange(ctx context.Context, chatID uint, startDate, endDate time.Time) ([]domain.Message, error) {
    if chatID == 0 {
        return nil, errors.New("invalid chat ID")
    }
    
    if startDate.After(endDate) {
        return nil, errors.New("start date must be before end date")
    }
    
    var messages []domain.Message
    err := r.db.WithContext(ctx).
        Where("chat_id = ? AND created_at BETWEEN ? AND ?", chatID, startDate, endDate).
        Order("created_at ASC").
        Find(&messages).Error
    
    if err != nil {
        log.Printf("[MessageRepository] Database error finding messages by date range for chat ID %d: %v", chatID, err)
        return nil, errors.New("database error finding messages by date range")
    }
    
    return messages, nil
}

// FindMessagesByType - Analytics: type-based filtering
func (r *gormMessageRepository) FindMessagesByType(ctx context.Context, chatID uint, messageType string, limit int) ([]domain.Message, error) {
    if chatID == 0 {
        return nil, errors.New("invalid chat ID")
    }
    
    if err := r.validateMessageType(messageType); err != nil {
        return nil, err
    }
    
    if limit <= 0 || limit > 1000 {
        limit = 100
    }
    
    var messages []domain.Message
    err := r.db.WithContext(ctx).
        Where("chat_id = ? AND message_type = ?", chatID, messageType).
        Order("created_at ASC").
        Limit(limit).
        Find(&messages).Error
    
    if err != nil {
        log.Printf("[MessageRepository] Database error finding messages by type for chat ID %d: %v", chatID, err)
        return nil, errors.New("database error finding messages by type")
    }
    
    return messages, nil
}

// SearchMessageContent - Medical AI: content search for medical analysis
func (r *gormMessageRepository) SearchMessageContent(ctx context.Context, chatID uint, searchTerm string, limit int) ([]domain.Message, error) {
    if chatID == 0 {
        return nil, errors.New("invalid chat ID")
    }
    
    if err := r.validateSearchTerm(searchTerm); err != nil {
        return nil, fmt.Errorf("invalid search term: %w", err)
    }
    
    if limit <= 0 || limit > 100 {
        limit = 20
    }
    
    var messages []domain.Message
    searchPattern := fmt.Sprintf("%%%s%%", searchTerm)
    
    err := r.db.WithContext(ctx).
        Where("chat_id = ? AND content LIKE ?", chatID, searchPattern).
        Order("created_at DESC").
        Limit(limit).
        Find(&messages).Error
    
    if err != nil {
        log.Printf("[MessageRepository] Database error searching message content for chat ID %d: %v", chatID, err)
        return nil, errors.New("database error searching message content")
    }
    
    return messages, nil
}

// FindLongMessages - Analytics: identify detailed medical responses
func (r *gormMessageRepository) FindLongMessages(ctx context.Context, chatID uint, minLength int, limit int) ([]domain.Message, error) {
    if chatID == 0 {
        return nil, errors.New("invalid chat ID")
    }
    
    if minLength < 0 {
        minLength = 500 // Default for detailed medical responses
    }
    
    if limit <= 0 || limit > 100 {
        limit = 20
    }
    
    var messages []domain.Message
    err := r.db.WithContext(ctx).
        Where("chat_id = ? AND LENGTH(content) >= ?", chatID, minLength).
        Order("created_at DESC").
        Limit(limit).
        Find(&messages).Error
    
    if err != nil {
        log.Printf("[MessageRepository] Database error finding long messages for chat ID %d: %v", chatID, err)
        return nil, errors.New("database error finding long messages")
    }
    
    return messages, nil
}

// DeleteOldMessages - Maintenance: automated cleanup for medical data retention
func (r *gormMessageRepository) DeleteOldMessages(ctx context.Context, chatID uint, olderThan time.Time) (int64, error) {
    if chatID == 0 {
        return 0, errors.New("invalid chat ID")
    }
    
    result := r.db.WithContext(ctx).
        Where("chat_id = ? AND created_at < ?", chatID, olderThan).
        Delete(&domain.Message{})
    
    if result.Error != nil {
        log.Printf("[MessageRepository] Database error deleting old messages for chat ID %d: %v", chatID, result.Error)
        return 0, errors.New("database error deleting old messages")
    }
    
    log.Printf("[MessageRepository] Deleted %d old messages for chat %d", result.RowsAffected, chatID)
    return result.RowsAffected, nil
}

// ArchiveMessagesByChatID - Maintenance: chat-based archiving
func (r *gormMessageRepository) ArchiveMessagesByChatID(ctx context.Context, chatID uint) (int64, error) {
    if chatID == 0 {
        return 0, errors.New("invalid chat ID")
    }
    
    // Mark messages as archived (implementation depends on your archival strategy)
    result := r.db.WithContext(ctx).
        Model(&domain.Message{}).
        Where("chat_id = ?", chatID).
        Update("archived", true)
    
    if result.Error != nil {
        log.Printf("[MessageRepository] Database error archiving messages for chat ID %d: %v", chatID, result.Error)
        return 0, errors.New("database error archiving messages")
    }
    
    log.Printf("[MessageRepository] Archived %d messages for chat %d", result.RowsAffected, chatID)
    return result.RowsAffected, nil
}

// UpdateMultipleTimestamps - Data integrity: bulk timestamp updates
func (r *gormMessageRepository) UpdateMultipleTimestamps(ctx context.Context, messageIDs []uint) error {
    if len(messageIDs) == 0 {
        return nil
    }
    
    // Validate all message IDs
    for _, messageID := range messageIDs {
        if messageID == 0 {
            return errors.New("invalid message ID in timestamp update")
        }
    }
    
    result := r.db.WithContext(ctx).
        Model(&domain.Message{}).
        Where("id IN ?", messageIDs).
        Update("updated_at", gorm.Expr("CURRENT_TIMESTAMP"))
    
    if result.Error != nil {
        log.Printf("[MessageRepository] Database error in bulk timestamp update: %v", result.Error)
        return errors.New("database error updating multiple timestamps")
    }
    
    log.Printf("[MessageRepository] Updated timestamps for %d messages", result.RowsAffected)
    return nil
}

// BulkUpdateMessageType - Data integrity: bulk type updates
func (r *gormMessageRepository) BulkUpdateMessageType(ctx context.Context, messageIDs []uint, newType string) error {
    if len(messageIDs) == 0 {
        return nil
    }
    
    if err := r.validateMessageType(newType); err != nil {
        return err
    }
    
    // Validate all message IDs
    for _, messageID := range messageIDs {
        if messageID == 0 {
            return errors.New("invalid message ID in type update")
        }
    }
    
    result := r.db.WithContext(ctx).
        Model(&domain.Message{}).
        Where("id IN ?", messageIDs).
        Update("message_type", newType)
    
    if result.Error != nil {
        log.Printf("[MessageRepository] Database error in bulk type update: %v", result.Error)
        return errors.New("database error updating multiple message types")
    }
    
    log.Printf("[MessageRepository] Updated message type for %d messages", result.RowsAffected)
    return nil
}

// ===== SECURITY VALIDATION HELPERS =====

// validateMessageInput - Comprehensive input validation
func (r *gormMessageRepository) validateMessageInput(message *domain.Message) error {
    if message == nil {
        return errors.New("message cannot be nil")
    }
    
    if message.ChatID == 0 {
        return errors.New("chat ID is required")
    }
    
    if err := r.validateMessageContent(message.Content); err != nil {
        return fmt.Errorf("content validation: %w", err)
    }
    
    if message.MessageType != "" {
        if err := r.validateMessageType(message.MessageType); err != nil {
            return fmt.Errorf("message type validation: %w", err)
        }
    }
    
    return nil
}

// validateMessageContent - Content validation with security checks
func (r *gormMessageRepository) validateMessageContent(content string) error {
    if strings.TrimSpace(content) == "" {
        return errors.New("message content cannot be empty")
    }
    
    if len(content) > 10000 {
        return errors.New("message content too long (max 10000 characters)")
    }
    
    // Basic XSS protection for medical content
    if strings.Contains(content, "<script") || strings.Contains(content, "javascript:") {
        return errors.New("invalid characters detected in message content")
    }
    
    return nil
}

// validateMessageType - Message type validation
func (r *gormMessageRepository) validateMessageType(messageType string) error {
    if len(messageType) > 50 {
        return errors.New("message type too long")
    }
    
    // Define allowed message types for medical AI
    allowedTypes := map[string]bool{
        "user":        true,
        "user_en":     true,   // <<< Add this line!
        "assistant":   true,
        "system":      true,
        "medical_ai":  true,
        "diagnostic":  true,
        "treatment":   true,
        "follow_up":   true,
    }
    
    if messageType != "" && !allowedTypes[messageType] {
        return errors.New("invalid message type")
    }
    
    return nil
}

// validateSearchTerm - Search term validation
func (r *gormMessageRepository) validateSearchTerm(term string) error {
    if len(term) > 100 {
        return errors.New("search term too long")
    }
    
    // Prevent SQL injection in LIKE queries
    if strings.Contains(term, "%") || strings.Contains(term, "_") || strings.Contains(term, "\\") {
        return errors.New("invalid characters in search term")
    }
    
    return nil
}

// ===== ERROR HANDLING HELPERS =====

// handleFindError - Secure error handling without data leakage
func (r *gormMessageRepository) handleFindError(err error, message *domain.Message, operation string) (*domain.Message, error) {
    if err == nil {
        return message, nil
    }
    
    if errors.Is(err, gorm.ErrRecordNotFound) {
        return nil, ErrMessageNotFound
    }
    
    // Log technical details for debugging
    log.Printf("[MessageRepository] %s database error: %v", operation, err)
    
    // Return generic error for security
    return nil, errors.New("database query failed")
}

// DeleteByChatID performs a bulk deletion of all messages associated with a given chatID.
func (r *gormMessageRepository) DeleteByChatID(ctx context.Context, chatID uint) error {
	if chatID == 0 {
		return errors.New("invalid chat ID")
	}

	result := r.db.WithContext(ctx).Where("chat_id = ?", chatID).Delete(&domain.Message{})
	if result.Error != nil {
		log.Printf("[MessageRepository] Database error deleting messages for chat ID %d: %v", chatID, result.Error)
		return errors.New("database error deleting messages by chat ID")
	}

	log.Printf("[MessageRepository] Deleted %d messages for chat %d", result.RowsAffected, chatID)
	return nil
}


// Get recent N user questions (descending order, exclude current)
func (r *gormMessageRepository) FindRecentUserAndAssistantMessages(
    ctx context.Context,
    chatID uint,
    userLimit int,
) ([]domain.Message, *domain.Message, error) {
    if chatID == 0 || userLimit <= 0 {
        return nil, nil, errors.New("invalid parameters")
    }

    // Fetch only the needed user messages (DESC)
    var users []domain.Message
    if err := r.db.WithContext(ctx).
        Where("chat_id = ? AND message_type = ?", chatID, domain.MessageTypeUser).
        Order("created_at DESC").
        Limit(userLimit).
        Find(&users).Error; err != nil {
        return nil, nil, err
    }

    // Fetch only the last assistant message
    var lastAssistant domain.Message
    if err := r.db.WithContext(ctx).
        Where("chat_id = ? AND message_type = ?", chatID, domain.MessageTypeAssistant).
        Order("created_at DESC").
        Limit(1).
        Find(&lastAssistant).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
        return users, nil, err
    }

    // Reverse user messages to oldest→newest
    for i, j := 0, len(users)-1; i < j; i, j = i+1, j-1 {
        users[i], users[j] = users[j], users[i]
    }

    var lastAssistantPtr *domain.Message
    if lastAssistant.ID != 0 {
        lastAssistantPtr = &lastAssistant
    }

    return users, lastAssistantPtr, nil
}


// Find recent user messages of a specific type (e.g., "user_en") plus last assistant message.
// FindRecentUserAndAssistantMessagesByType - Efficient retrieval of recent user messages of a specific type + last assistant message
func (r *gormMessageRepository) FindRecentUserAndAssistantMessagesByType(
    ctx context.Context,
    chatID uint,
    userLimit int,
    userType string,
) ([]domain.Message, *domain.Message, error) {
    if chatID == 0 || userLimit <= 0 {
        return nil, nil, errors.New("invalid parameters")
    }

    // 1️⃣ Fetch recent user messages of the given type (descending order)
    var users []domain.Message
    if err := r.db.WithContext(ctx).
        Where("chat_id = ? AND message_type = ?", chatID, userType).
        Order("created_at DESC").
        Limit(userLimit).
        Find(&users).Error; err != nil {
        return nil, nil, err
    }

    // 2️⃣ Fetch only the last assistant message
    var lastAssistant domain.Message
    if err := r.db.WithContext(ctx).
        Where("chat_id = ? AND message_type = ?", chatID, domain.MessageTypeAssistant).
        Order("created_at DESC").
        Limit(1).
        Find(&lastAssistant).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
        return users, nil, err
    }

    // 3️⃣ Reverse users slice to chronological order (oldest → newest)
    for i, j := 0, len(users)-1; i < j; i, j = i+1, j-1 {
        users[i], users[j] = users[j], users[i]
    }

    var lastAssistantPtr *domain.Message
    if lastAssistant.ID != 0 {
        lastAssistantPtr = &lastAssistant
    }

    return users, lastAssistantPtr, nil
}



// FindRecentUserAssistantPairs - Efficiently fetch up to pairLimit recent (user→assistant) pairs.
func (r *gormMessageRepository) FindRecentUserAssistantPairs(
    ctx context.Context,
    chatID uint,
    pairLimit int,
    userType string,
) ([]domain.Message, error) {
    if chatID == 0 {
        return nil, errors.New("invalid chat ID")
    }

    if pairLimit <= 0 {
        return nil, errors.New("pair limit must be > 0")
    }

    // Step 1: Fetch the last N*2 messages of relevant types (user + assistant) in descending order
    // N*2 ensures we have enough messages to form pairs.
    fetchLimit := pairLimit * 2
    var messages []domain.Message
    err := r.db.WithContext(ctx).
        Where("chat_id = ? AND (message_type = ? OR message_type = ?)", chatID, userType, domain.MessageTypeAssistant).
        Order("created_at DESC").
        Limit(fetchLimit * 5). // fetch more to ensure enough valid pairs in sparse chats
        Find(&messages).Error
    if err != nil {
        return nil, err
    }

    // Step 2: Reverse slice to chronological order (oldest → newest)
    for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
        messages[i], messages[j] = messages[j], messages[i]
    }

    // Step 3: Build user→assistant pairs
    var pairs []domain.Message
    var userMsg *domain.Message
    for _, msg := range messages {
        if msg.MessageType == userType {
            userMsg = &msg
        } else if msg.MessageType == domain.MessageTypeAssistant && userMsg != nil {
            // append pair
            pairs = append(pairs, *userMsg, msg)
            userMsg = nil
        }
        if len(pairs)/2 == pairLimit {
            break
        }
    }

    return pairs, nil
}
