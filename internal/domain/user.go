// File: internal/domain/user.go
package domain

import (
	"errors"
	"time"
)

// UserStatus defines the state of a user account.
type UserStatus string

const (
	// UserStatusPending means the user has registered but not yet verified their phone number.
	UserStatusPending UserStatus = "pending"
	// UserStatusActive means the user is fully registered and can log in.
	UserStatusActive UserStatus = "active"
)

// User represents a user in the system.
type User struct {
	ID        uint       `gorm:"primaryKey" json:"id"`
	Username  string     `gorm:"uniqueIndex;not null;size:20" json:"username"`
	PhoneNumber string   `gorm:"uniqueIndex;not null;size:15" json:"phone_number"`
	Password  string     `gorm:"not null" json:"-"`

	// Fields for SMS verification and account status
	Status                UserStatus `gorm:"default:'pending';not null;size:10" json:"status"`
	VerificationCode      string     `gorm:"index" json:"-"`
	VerificationExpiresAt time.Time  `gorm:"default:null" json:"-"`

	// Fields for database-driven login lockout
	FailedLoginAttempts int       `gorm:"default:0" json:"-"`
	LockedUntil         time.Time `gorm:"default:null" json:"-"`
	
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// IsValid performs basic validation on the User model.
func (u *User) IsValid() error {
	if len(u.Username) < 3 {
		return errors.New("username must be at least 3 characters")
	}
	if u.PhoneNumber == "" {
		return errors.New("phone number is required")
	}
	return nil
}