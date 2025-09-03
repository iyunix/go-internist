// File: internal/repository/message_repository.go
package repository

import (
	"context"

	"github.com/iyunix/go-internist/internal/domain"
	"gorm.io/gorm"
)

type messageRepository struct {
	db *gorm.DB
}

func NewMessageRepository(db *gorm.DB) MessageRepository {
	return &messageRepository{db: db}
}

func (r *messageRepository) Create(ctx context.Context, message *domain.Message) error {
	return r.db.WithContext(ctx).Create(message).Error
}

func (r *messageRepository) FindByChatID(ctx context.Context, chatID uint) ([]domain.Message, error) {
	var messages []domain.Message
	err := r.db.WithContext(ctx).Where("chat_id = ?", chatID).Order("created_at ASC").Find(&messages).Error
	return messages, err
}