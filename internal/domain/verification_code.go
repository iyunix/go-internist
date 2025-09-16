// File: internal/domain/verification_code.go
package domain

import (
    "time"
    "gorm.io/gorm"
)

// VerificationCodeType defines different types of verification codes
type VerificationCodeType string

const (
    VerificationTypeSMS      VerificationCodeType = "sms"
    VerificationTypePassword VerificationCodeType = "password_reset"
)

// VerificationCode handles all temporary verification codes
type VerificationCode struct {
    ID          uint                 `gorm:"primaryKey"`
    PhoneNumber string               `gorm:"index;not null;size:15"`
    Code        string               `gorm:"not null;size:10"`
    Type        VerificationCodeType `gorm:"not null;size:20;index"` // SMS, password_reset, etc.
    
    // Security and rate limiting
    ExpiresAt   time.Time `gorm:"index;not null"`
    Attempts    int       `gorm:"not null;default:0"`
    MaxAttempts int       `gorm:"not null;default:3"`
    
    // Usage tracking
    UsedAt      *time.Time `gorm:"default:null"`
    IsUsed      bool       `gorm:"default:false;index"`
    
    // Timestamps
    CreatedAt   time.Time
    UpdatedAt   time.Time
    DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"` // ADDED: Soft delete protection
}

// IsValid checks if the verification code is still valid
func (v *VerificationCode) IsValid() bool {
    return !v.IsUsed && v.Attempts < v.MaxAttempts && time.Now().Before(v.ExpiresAt)
}

// CanAttempt checks if more attempts are allowed
func (v *VerificationCode) CanAttempt() bool {
    return v.Attempts < v.MaxAttempts && !v.IsUsed
}

// UseCode marks the code as used
func (v *VerificationCode) UseCode() {
    now := time.Now()
    v.IsUsed = true
    v.UsedAt = &now
}

// IncrementAttempt increments the attempt counter
func (v *VerificationCode) IncrementAttempt() {
    v.Attempts++
}
