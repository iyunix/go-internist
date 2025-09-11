package message

import (
    "context"
    "errors"
    "log"

    "github.com/iyunix/go-internist/internal/domain"
    "gorm.io/gorm"
)

type gormMessageRepository struct {
    db *gorm.DB
}

func NewMessageRepository(db *gorm.DB) MessageRepository {
    return &gormMessageRepository{db: db}
}

func (r *gormMessageRepository) FindByChatID(ctx context.Context, chatID uint) ([]domain.Message, error) {
    var messages []domain.Message
    err := r.db.WithContext(ctx).
        Where("chat_id = ?", chatID).
        Order("created_at asc").
        Find(&messages).Error
    if err != nil {
        log.Printf("[MessageRepository] FindByChatID error: chat_id=%d %v", chatID, err)
        return nil, errors.New("database error fetching messages")
    }
    return messages, nil
}

func (r *gormMessageRepository) Create(ctx context.Context, message *domain.Message) (*domain.Message, error) {
    if message.Content == "" {
        return nil, errors.New("message content cannot be empty")
    }
    err := r.db.WithContext(ctx).Create(message).Error
    if err != nil {
        log.Printf("[MessageRepository] Create error: %v", err)
        return nil, errors.New("database error creating message")
    }
    return message, nil
}
