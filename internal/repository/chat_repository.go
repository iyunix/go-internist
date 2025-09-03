// File: internal/repository/chat_repository.go
package repository

import (
	"context"

	"github.com/iyunix/go-internist/internal/domain"
	"gorm.io/gorm"
)

type chatRepository struct {
	db *gorm.DB
}

func NewChatRepository(db *gorm.DB) ChatRepository {
	return &chatRepository{db: db}
}

// Add this function to chat_repository.go
func (r *chatRepository) Delete(ctx context.Context, chatID uint, userID uint) error {
	// We include UserID in the where clause to ensure a user can only delete their own chats.
	return r.db.WithContext(ctx).Where("id = ? AND user_id = ?", chatID, userID).Delete(&domain.Chat{}).Error
}

func (r *chatRepository) Create(ctx context.Context, chat *domain.Chat) (*domain.Chat, error) {
	if err := r.db.WithContext(ctx).Create(chat).Error; err != nil {
		return nil, err
	}
	return chat, nil
}

func (r *chatRepository) FindByID(ctx context.Context, id uint) (*domain.Chat, error) {
	var chat domain.Chat
	err := r.db.WithContext(ctx).First(&chat, id).Error
	return &chat, err
}

func (r *chatRepository) FindByUserID(ctx context.Context, userID uint) ([]domain.Chat, error) {
	var chats []domain.Chat
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("updated_at DESC").Find(&chats).Error
	return chats, err
}