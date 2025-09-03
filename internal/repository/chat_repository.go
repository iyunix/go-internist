// File: internal/repository/chat_repository.go

package repository

import (
    "context"
    "errors"
    "log"

    "github.com/iyunix/go-internist/internal/domain"
    "gorm.io/gorm"
)


type gormChatRepository struct {
    db *gorm.DB
}

func NewChatRepository(db *gorm.DB) ChatRepository {
    return &gormChatRepository{db: db}
}

func (r *gormChatRepository) FindByUserID(ctx context.Context, userID uint) ([]domain.Chat, error) {
    var chats []domain.Chat
    err := r.db.WithContext(ctx).
        Where("user_id = ?", userID).
        Order("updated_at desc").
        Find(&chats).Error
    if err != nil {
        log.Printf("[ChatRepository] FindByUserID error: user_id=%d %v", userID, err)
        return nil, errors.New("database error fetching chats")
    }
    return chats, nil
}

func (r *gormChatRepository) FindByID(ctx context.Context, chatID uint) (*domain.Chat, error) {
    var chat domain.Chat
    err := r.db.WithContext(ctx).First(&chat, chatID).Error
    if err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, errors.New("chat not found")
        }
        log.Printf("[ChatRepository] FindByID error: chat_id=%d %v", chatID, err)
        return nil, errors.New("database error finding chat")
    }
    return &chat, nil
}

func (r *gormChatRepository) Create(ctx context.Context, chat *domain.Chat) (*domain.Chat, error) {
    err := r.db.WithContext(ctx).Create(chat).Error
    if err != nil {
        log.Printf("[ChatRepository] Create error: %v", err)
        return nil, errors.New("database error creating chat")
    }
    // Optionally log/audit
    return chat, nil
}

func (r *gormChatRepository) Delete(ctx context.Context, chatID, userID uint) error {
    // Only delete if the chat belongs to the user
    result := r.db.WithContext(ctx).
        Where("id = ? AND user_id = ?", chatID, userID).
        Delete(&domain.Chat{})
    if result.Error != nil {
        log.Printf("[ChatRepository] Delete error: chat_id=%d user_id=%d %v", chatID, userID, result.Error)
        return errors.New("database error deleting chat")
    }
    if result.RowsAffected == 0 {
        return errors.New("chat not found or unauthorized")
    }
    // Optionally change to soft-delete for recoverability
    return nil
}
