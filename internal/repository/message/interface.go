// File: internal/repository/message/interface.go
package message

import (
	"context"
	"time"

	"github.com/iyunix/go-internist/internal/domain"
)

type MessageRepository interface {
	Create(ctx context.Context, message *domain.Message) (*domain.Message, error)
	FindByChatID(ctx context.Context, chatID uint) ([]domain.Message, error)
	FindByChatIDWithPagination(ctx context.Context, chatID uint, limit, offset int) ([]domain.Message, int64, error)
	FindByID(ctx context.Context, messageID uint) (*domain.Message, error)
	Update(ctx context.Context, message *domain.Message) error
	Delete(ctx context.Context, messageID, chatID uint) error
	CreateInBatch(ctx context.Context, messages []*domain.Message, batchSize int) error
	DeleteMultipleByChatID(ctx context.Context, messageIDs []uint, chatID uint) error
	ExistsByID(ctx context.Context, messageID uint) (bool, error)
	ExistsByIDAndChatID(ctx context.Context, messageID, chatID uint) (bool, error)
	VerifyMessageOwnership(ctx context.Context, messageID, chatID uint) (bool, error)
	CountByChatID(ctx context.Context, chatID uint) (int64, error)
	CountTotalMessages(ctx context.Context) (int64, error)
	CountMessagesByType(ctx context.Context, chatID uint, messageType string) (int64, error)
	FindRecentMessages(ctx context.Context, chatID uint, limit int) ([]domain.Message, error)
	FindMessagesByDateRange(ctx context.Context, chatID uint, startDate, endDate time.Time) ([]domain.Message, error)
	FindMessagesByType(ctx context.Context, chatID uint, messageType string, limit int) ([]domain.Message, error)
	SearchMessageContent(ctx context.Context, chatID uint, searchTerm string, limit int) ([]domain.Message, error)
	FindLongMessages(ctx context.Context, chatID uint, minLength int, limit int) ([]domain.Message, error)
	DeleteOldMessages(ctx context.Context, chatID uint, olderThan time.Time) (int64, error)
	ArchiveMessagesByChatID(ctx context.Context, chatID uint) (int64, error)
	UpdateMultipleTimestamps(ctx context.Context, messageIDs []uint) error
	BulkUpdateMessageType(ctx context.Context, messageIDs []uint, newType string) error

	// --- ADD THIS NEW METHOD ---
	DeleteByChatID(ctx context.Context, chatID uint) error
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
