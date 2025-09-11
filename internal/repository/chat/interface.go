// G:\go_internist\internal\repository\chat\interface.go

package chat

import (
    "context"
    "time"
    "github.com/iyunix/go-internist/internal/domain"
)

// ChatRepository handles chat data operations with production-ready enhancements.
type ChatRepository interface {
    // ===== EXISTING METHODS (enhanced with validation) =====
    Create(ctx context.Context, chat *domain.Chat) (*domain.Chat, error)
    FindByID(ctx context.Context, id uint) (*domain.Chat, error)
    FindByUserID(ctx context.Context, userID uint) ([]domain.Chat, error) // [DEPRECATED: Use FindByUserIDWithPagination]
    Delete(ctx context.Context, chatID uint, userID uint) error
    TouchUpdatedAt(ctx context.Context, chatID uint) error

    // ===== PRODUCTION-READY METHODS =====
    
    // Memory Safety: Pagination for large chat histories
    FindByUserIDWithPagination(ctx context.Context, userID uint, limit, offset int) ([]domain.Chat, int64, error)
    
    // Performance: Batch operations for bulk management
    CreateInBatch(ctx context.Context, chats []*domain.Chat, batchSize int) error
    DeleteMultipleByUserID(ctx context.Context, chatIDs []uint, userID uint) error
    
    // Security: Existence and ownership validation
    ExistsByID(ctx context.Context, chatID uint) (bool, error)
    ExistsByIDAndUserID(ctx context.Context, chatID, userID uint) (bool, error)
    VerifyOwnership(ctx context.Context, chatID, userID uint) (bool, error)
    
    // Performance: Efficient counting and metrics
    CountByUserID(ctx context.Context, userID uint) (int64, error)
    CountTotalChats(ctx context.Context) (int64, error)
    CountActiveChats(ctx context.Context, since time.Time) (int64, error)
    
    // Analytics: Advanced querying for medical AI insights
    FindRecentChats(ctx context.Context, userID uint, limit int) ([]domain.Chat, error)
    FindChatsByDateRange(ctx context.Context, userID uint, startDate, endDate time.Time) ([]domain.Chat, error)
    FindOldestChats(ctx context.Context, userID uint, limit int) ([]domain.Chat, error)
    
    // Maintenance: Cleanup operations
    DeleteOldChats(ctx context.Context, userID uint, olderThan time.Time) (int64, error)
    ArchiveInactiveChats(ctx context.Context, inactiveSince time.Time) (int64, error)
    
    // Data Integrity: Bulk timestamp updates
    UpdateMultipleTimestamps(ctx context.Context, chatIDs []uint) error
    
    // Search: Title-based search for chat organization
    SearchChatsByTitle(ctx context.Context, userID uint, titlePattern string, limit int) ([]domain.Chat, error)
}

// Supporting types for enhanced functionality
type ChatSearchFilter struct {
    UserID      uint
    TitlePattern string
    StartDate   *time.Time
    EndDate     *time.Time
    Limit       int
    Offset      int
}

type ChatMetrics struct {
    TotalChats      int64
    ActiveChats     int64
    ChatsToday      int64
    ChatsThisWeek   int64
    ChatsThisMonth  int64
}
