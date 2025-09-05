package domain

import "time"

type VerificationCode struct {
    ID          uint      `gorm:"primaryKey"`
    PhoneNumber string    `gorm:"index;not null"`
    Code        string    `gorm:"not null"`
    ExpiresAt   time.Time `gorm:"index;not null"`
    Attempts    int       `gorm:"not null;default:0"`
    CreatedAt   time.Time
}
