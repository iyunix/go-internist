// G:\go_internist\internal\domain\user.go
package domain

import (
    "errors"
    "time"
)

// UserStatus defines the state of a user account.
type UserStatus string

const (
    UserStatusPending UserStatus = "pending"
    UserStatusActive  UserStatus = "active"
)

// --- 1. DEFINE SUBSCRIPTION PLANS ---
// SubscriptionPlan defines the type for user subscription tiers.
type SubscriptionPlan string

const (
    PlanBasic   SubscriptionPlan = "basic"
    PlanPro     SubscriptionPlan = "pro"
    PlanPremium SubscriptionPlan = "premium"
)

// PlanCredits maps each subscription plan to its corresponding credit amount.
var PlanCredits = map[SubscriptionPlan]int{
    PlanBasic:   2500,
    PlanPro:     5000,
    PlanPremium: 10000,
}

// Character usage constants
const (
    MinCharacterCharge = 100
    MaxQuestionLength  = 1000
)

// User represents a user in the system.
type User struct {
    ID          uint      `gorm:"primaryKey" json:"id"`
    Username    string    `gorm:"uniqueIndex;not null;size:20" json:"username"`
    PhoneNumber string    `gorm:"uniqueIndex;not null;size:15" json:"phone_number"`
    Password    string    `gorm:"not null" json:"-"`

    // Fields for SMS verification and account status
    Status                UserStatus `gorm:"default:'pending';not null;size:10" json:"status"`
    VerificationCode      string     `gorm:"index" json:"-"`
    VerificationExpiresAt time.Time  `gorm:"default:null" json:"-"`
       
    // ADD MISSING VERIFICATION FIELDS
    IsVerified                bool       `gorm:"default:false;not null" json:"is_verified"`
    VerificationCodeSentAt    *time.Time `gorm:"default:null" json:"-"`
    VerificationCodeExpiresAt *time.Time `gorm:"default:null" json:"-"`
    VerifiedAt                *time.Time `gorm:"default:null" json:"-"`

    // Fields for database-driven login lockout
    FailedLoginAttempts int        `gorm:"default:0" json:"-"`
    LastFailedLoginAt   *time.Time `gorm:"default:null" json:"-"`  // ADD MISSING FIELD
    LockedUntil         *time.Time `gorm:"default:null" json:"-"`  // CHANGE TO POINTER

    IsAdmin             bool       `gorm:"default:false;not null" json:"-"`
      
    // --- 2. ADD SUBSCRIPTION PLAN FIELD TO USER ---
    SubscriptionPlan      SubscriptionPlan `gorm:"default:'basic';not null;size:15" json:"subscription_plan"`
    
    // Character usage tracking
    CharacterBalance      int `gorm:"default:2500;not null" json:"character_balance"`
    TotalCharacterBalance int `gorm:"default:2500;not null" json:"total_character_balance"`
    
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}

// --- All helper functions below remain unchanged ---

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

// CanAskQuestion checks if user has enough characters to ask a question
func (u *User) CanAskQuestion() bool {
    return u.CharacterBalance >= MinCharacterCharge
}

// DeductCharacters deducts characters from user balance
func (u *User) DeductCharacters(questionLength int) error {
    if !u.CanAskQuestion() {
        return errors.New("insufficient character balance")
    }
    chargeAmount := questionLength
    if chargeAmount < MinCharacterCharge {
        chargeAmount = MinCharacterCharge
    }
    if u.CharacterBalance < chargeAmount {
        return errors.New("insufficient character balance")
    }
    u.CharacterBalance -= chargeAmount
    return nil
}

// CalculateChargeForQuestion returns how many characters will be charged
func (u *User) CalculateChargeForQuestion(questionLength int) int {
    if questionLength < MinCharacterCharge {
        return MinCharacterCharge
    }
    return questionLength
}

// AddCharacters adds characters to user balance (for future admin functionality)
func (u *User) AddCharacters(amount int) {
    if amount > 0 {
        u.CharacterBalance += amount
    }
}

// GetCharacterBalance returns current character balance
func (u *User) GetCharacterBalance() int {
    return u.CharacterBalance
}


type BalanceUpdate struct {
    UserID uint
    Amount int
}