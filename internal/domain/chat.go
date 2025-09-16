// File: internal/domain/chat.go
package domain

import (
    "time"
    "gorm.io/gorm"
)

// Chat represents a single conversation thread.
type Chat struct {
    ID        uint      `gorm:"primarykey"`
    UserID    uint      `gorm:"not null;index"` // Added index for user's chat queries
    Title     string    `gorm:"size:200"`       // Reasonable title length limit
    
    // Timestamps
    CreatedAt time.Time
    UpdatedAt time.Time
    DeletedAt gorm.DeletedAt `gorm:"index" json:"-"` // ADDED: Soft delete protection
    
    // Relationships
    Messages  []Message `gorm:"foreignKey:ChatID" json:"-"` // Preload relationship
}

// GetDisplayTitle returns a truncated title for display
func (c *Chat) GetDisplayTitle() string {
    if len(c.Title) > 50 {
        return c.Title[:47] + "..."
    }
    return c.Title
}
