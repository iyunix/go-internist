// File: internal/domain/chat.go
package domain

import "time"

// Chat represents a single conversation thread.
type Chat struct {
    ID        uint      `gorm:"primarykey"`
    UserID    uint      `gorm:"not null"` // The ID of the user who owns the chat
    Title     string    // The title of the chat, e.g., "Capital of Sweden"
    CreatedAt time.Time
    UpdatedAt time.Time
}