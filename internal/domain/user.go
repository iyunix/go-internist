// File: internal/domain/user.go
package domain

import (
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt" // <-- New tool for password hashing
)

type User struct {
    ID          uint      `json:"id"`
    Username    string    `json:"username"`
    Password    string    `json:"-"`
    PhoneNumber string    `json:"phone_number"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}

// HashPassword securely hashes the user's password.
func (u *User) HashPassword(password string) error {
	if len(password) < 8 {
		return errors.New("password must be at least 8 characters")
	}
	// This command hashes the password securely.
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.Password = string(hashed)
	return nil
}

// ValidatePassword compares a plain-text password with the user's hashed password.
func (u *User) ValidatePassword(password string) error {
	// bcrypt.CompareHashAndPassword does all the complex work for us.
	return bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
}

func (u *User) IsValid() error {
    if len(u.Username) < 3 {
        return errors.New("username must be at least 3 characters")
    }
    if u.PhoneNumber == "" {
        return errors.New("phone number is required")
    }
    return nil
}