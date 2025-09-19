// File: internal/domain/message.go
package domain

import (
    "time"
    "gorm.io/gorm"
)

// Message represents a chat message in the medical AI system
type Message struct {
    ID          uint      `gorm:"primaryKey" json:"id"`
    ChatID      uint      `gorm:"not null;index" json:"chat_id"` // Critical index
    Content     string    `gorm:"type:text;not null" json:"content"`
    
    // Message classification
    MessageType string    `gorm:"size:50;index;default:'user'" json:"messageType"`
    
    // Data management for large tables
    Archived    bool      `gorm:"default:false;index" json:"archived"` // Added index for archival queries
    
    // Performance tracking (optional)
    TokenCount  *int      `gorm:"default:null" json:"token_count,omitempty"` // For billing/analytics
    
    // Timestamps
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
    DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"` // ADDED: Soft delete protection
    
    // Foreign key relationship
    Chat        Chat      `gorm:"foreignKey:ChatID" json:"-"`
}

// Medical AI Message Types
const (
    MessageTypeUser       = "user"
    MessageTypeAssistant  = "assistant"  
    MessageTypeSystem     = "system"
    MessageTypeMedicalAI  = "medical_ai"
    MessageTypeDiagnostic = "diagnostic"
    MessageTypeTreatment  = "treatment"
    MessageTypeFollowUp   = "follow_up"
    MessageTypeInternalContext = "internal_context" // ADD THIS LINE

)

// IsValidMessageType checks if the message type is valid
func (m *Message) IsValidMessageType() bool {
    validTypes := []string{
        MessageTypeUser, MessageTypeAssistant, MessageTypeSystem,
        MessageTypeMedicalAI, MessageTypeDiagnostic, MessageTypeTreatment, MessageTypeFollowUp,
        MessageTypeInternalContext, // ADD THIS LINE
    }
    
    for _, validType := range validTypes {
        if m.MessageType == validType {
            return true
        }
    }
    return false
}

// GetContentPreview returns a truncated version for display
func (m *Message) GetContentPreview(maxLength int) string {
    if len(m.Content) <= maxLength {
        return m.Content
    }
    return m.Content[:maxLength-3] + "..."
}