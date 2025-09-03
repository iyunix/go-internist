// File: internal/domain/message.go
package domain

import "time"

// Message represents a single message within a chat.
type Message struct {
    ID        uint      `gorm:"primarykey"`
    ChatID    uint      `gorm:"not null"` // The ID of the chat this message belongs to
    Role      string    `gorm:"not null"` // "user" or "assistant"
    Content   string    `gorm:"not null"`
    CreatedAt time.Time
}