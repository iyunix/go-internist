// G:\go_internist\internal\repository\message\interface.go

package message

import (
    "context"
    "time"
    "github.com/iyunix/go-internist/internal/domain"
)

// MessageRepository handles message data operations with production-ready enhancements.
type MessageRepository interface {
    // ===== EXISTING METHODS (enhanced with validation) =====
    Create(ctx context.Context, message *domain.Message) (*domain.Message, error)
    FindByChatID(ctx context.Context, chatID uint) ([]domain.Message, error) // [DEPRECATED: Use FindByChatIDWithPagination]

    // ===== PRODUCTION-READY METHODS =====
    
    // Memory Safety: Pagination for large medical conversations
    FindByChatIDWithPagination(ctx context.Context, chatID uint, limit, offset int) ([]domain.Message, int64, error)
    
    // CRUD Operations: Complete message lifecycle
    FindByID(ctx context.Context, messageID uint) (*domain.Message, error)
    Update(ctx context.Context, message *domain.Message) error
    Delete(ctx context.Context, messageID, chatID uint) error
    
    // Performance: Batch operations for bulk processing
    CreateInBatch(ctx context.Context, messages []*domain.Message, batchSize int) error
    DeleteMultipleByChatID(ctx context.Context, messageIDs []uint, chatID uint) error
    
    // Security: Existence and ownership validation
    ExistsByID(ctx context.Context, messageID uint) (bool, error)
    ExistsByIDAndChatID(ctx context.Context, messageID, chatID uint) (bool, error)
    VerifyMessageOwnership(ctx context.Context, messageID, chatID uint) (bool, error)
    
    // Performance: Efficient counting and metrics
    CountByChatID(ctx context.Context, chatID uint) (int64, error)
    CountTotalMessages(ctx context.Context) (int64, error)
    CountMessagesByType(ctx context.Context, chatID uint, messageType string) (int64, error)
    
    // Medical AI Analytics: Advanced querying
    FindRecentMessages(ctx context.Context, chatID uint, limit int) ([]domain.Message, error)
    FindMessagesByDateRange(ctx context.Context, chatID uint, startDate, endDate time.Time) ([]domain.Message, error)
    FindMessagesByType(ctx context.Context, chatID uint, messageType string, limit int) ([]domain.Message, error)
    
    // Medical Content Search: AI/medical content analysis
    SearchMessageContent(ctx context.Context, chatID uint, searchTerm string, limit int) ([]domain.Message, error)
    FindLongMessages(ctx context.Context, chatID uint, minLength int, limit int) ([]domain.Message, error)
    
    // Maintenance: Cleanup operations for medical data retention
    DeleteOldMessages(ctx context.Context, chatID uint, olderThan time.Time) (int64, error)
    ArchiveMessagesByChatID(ctx context.Context, chatID uint) (int64, error)
    
    // Data Integrity: Bulk operations
    UpdateMultipleTimestamps(ctx context.Context, messageIDs []uint) error
    BulkUpdateMessageType(ctx context.Context, messageIDs []uint, newType string) error
}

// Supporting types for enhanced functionality
type MessageMetrics struct {
    TotalMessages     int64
    UserMessages      int64
    AIMessages        int64
    SystemMessages    int64
    AverageLength     float64
    MessagesToday     int64
    MessagesThisWeek  int64
}

type MessageSearchFilter struct {
    ChatID      uint
    MessageType string
    SearchTerm  string
    StartDate   *time.Time
    EndDate     *time.Time
    MinLength   int
    MaxLength   int
    Limit       int
    Offset      int
}
