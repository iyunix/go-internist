// G:\go_internist\internal\domain\message.go
package domain

import "time"

// Message represents a chat message in the medical AI system
type Message struct {
    ID          uint      `gorm:"primaryKey" json:"id"`
    ChatID      uint      `gorm:"not null;index" json:"chat_id"`
    Content     string    `gorm:"type:text;not null" json:"content"`
    
    // Production-ready fields:
    MessageType string    `gorm:"size:50;index;default:'user'" json:"messageType"`
    Archived    bool      `gorm:"default:false" json:"archived"`
    
    // Timestamps
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
    
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
)
