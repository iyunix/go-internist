// File: internal/domain/user.go
package domain

import (
    "errors"
    "time"
    "gorm.io/gorm"
)

// UserStatus defines the state of a user account.
type UserStatus string

const (
    UserStatusPending UserStatus = "pending"
    UserStatusActive  UserStatus = "active"
)

// SubscriptionPlan defines the type for user subscription tiers.
type SubscriptionPlan string

const (
    PlanBasic   SubscriptionPlan = "basic"
    PlanPro     SubscriptionPlan = "pro"
    PlanPremium SubscriptionPlan = "premium"
    PlanUltra   SubscriptionPlan = "ultra"    // NEW: Ultra plan added
)

// PlanCredits maps each subscription plan to its corresponding credit amount.
var PlanCredits = map[SubscriptionPlan]int{
    PlanBasic:   25000,   // Updated from 2,500
    PlanPro:     50000,   // Updated from 5,000  
    PlanPremium: 100000,  // Updated from 10,000
    PlanUltra:   500000,  // NEW: Ultra plan with 500,000 characters
}

// Character usage constants
const (
    MinCharacterCharge        = 100
    MaxQuestionLength         = 1000
    DefaultRegistrationChars  = 2500  // NEW: Initial chars for new users
)

// User represents a user in the system - UPDATED WITH NEW PLANS
type User struct {
    ID          uint      `gorm:"primaryKey" json:"id"`
    Username    string    `gorm:"uniqueIndex;not null;size:20" json:"username"`
    PhoneNumber string    `gorm:"uniqueIndex;not null;size:15" json:"phone_number"`
    Password    string    `gorm:"not null" json:"-"`

    // Account status and verification (FINAL STATE ONLY)
    Status     UserStatus `gorm:"default:'pending';not null;size:10" json:"status"`
    IsVerified bool       `gorm:"default:false;not null" json:"is_verified"`
    VerifiedAt *time.Time `gorm:"default:null" json:"-"`

    // Security and login lockout
    FailedLoginAttempts int        `gorm:"default:0" json:"-"`
    LastFailedLoginAt   *time.Time `gorm:"default:null" json:"-"`
    LockedUntil         *time.Time `gorm:"default:null" json:"-"`
    IsAdmin             bool       `gorm:"default:false;not null" json:"-"`

    // Subscription and billing - UPDATED DEFAULTS
    SubscriptionPlan      SubscriptionPlan `gorm:"default:'basic';not null;size:15" json:"subscription_plan"`
    CharacterBalance      int              `gorm:"default:2500;not null" json:"character_balance"`      // Updated default
    TotalCharacterBalance int              `gorm:"default:2500;not null" json:"total_character_balance"` // Updated default

    // Timestamps
    CreatedAt time.Time       `json:"created_at"`
    UpdatedAt time.Time       `json:"updated_at"`
    DeletedAt gorm.DeletedAt  `gorm:"index" json:"-"` // Soft delete protection
}

// NewUser creates a new user with initial character allocation
func NewUser(username, phoneNumber, hashedPassword string) *User {
    return &User{
        Username:              username,
        PhoneNumber:           phoneNumber,
        Password:              hashedPassword, // Should be pre-hashed
        Status:                UserStatusPending,
        SubscriptionPlan:      PlanBasic,
        CharacterBalance:      DefaultRegistrationChars, // 2,500 characters
        TotalCharacterBalance: DefaultRegistrationChars, // 2,500 characters
        CreatedAt:             time.Now(),
        UpdatedAt:             time.Now(),
    }
}

// Business logic methods
func (u *User) IsValid() error {
    if len(u.Username) < 3 {
        return errors.New("username must be at least 3 characters")
    }
    if u.PhoneNumber == "" {
        return errors.New("phone number is required")
    }
    return nil
}

func (u *User) CanAskQuestion() bool {
    return u.CharacterBalance >= MinCharacterCharge
}

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

func (u *User) CalculateChargeForQuestion(questionLength int) int {
    if questionLength < MinCharacterCharge {
        return MinCharacterCharge
    }
    return questionLength
}

func (u *User) AddCharacters(amount int) {
    if amount > 0 {
        u.CharacterBalance += amount
    }
}

func (u *User) GetCharacterBalance() int {
    return u.CharacterBalance
}

// ValidateSubscriptionPlan ensures the plan is valid - UPDATED
func (u *User) ValidateSubscriptionPlan() error {
    switch u.SubscriptionPlan {
    case PlanBasic, PlanPro, PlanPremium, PlanUltra: // Added PlanUltra
        return nil
    default:
        return errors.New("invalid subscription plan")
    }
}

// UpgradeSubscription upgrades user to a new subscription plan
func (u *User) UpgradeSubscription(newPlan SubscriptionPlan) error {
    if err := u.ValidateNewPlan(newPlan); err != nil {
        return err
    }
    
    // Add the difference in character allocation
    currentPlanChars := PlanCredits[u.SubscriptionPlan]
    newPlanChars := PlanCredits[newPlan]
    
    if newPlanChars > currentPlanChars {
        additionalChars := newPlanChars - currentPlanChars
        u.CharacterBalance += additionalChars
        u.TotalCharacterBalance += additionalChars
    }
    
    u.SubscriptionPlan = newPlan
    return nil
}

// ValidateNewPlan checks if the new plan is valid and is an upgrade
func (u *User) ValidateNewPlan(newPlan SubscriptionPlan) error {
    // Validate the new plan exists
    if _, exists := PlanCredits[newPlan]; !exists {
        return errors.New("invalid subscription plan")
    }
    
    // Check if it's actually an upgrade
    currentPlanChars := PlanCredits[u.SubscriptionPlan]
    newPlanChars := PlanCredits[newPlan]
    
    if newPlanChars <= currentPlanChars {
        return errors.New("new plan must be an upgrade from current plan")
    }
    
    return nil
}

// GetPlanName returns the human-readable plan name
func (u *User) GetPlanName() string {
    switch u.SubscriptionPlan {
    case PlanBasic:
        return "Basic"
    case PlanPro:
        return "Pro"
    case PlanPremium:
        return "Premium"
    case PlanUltra:
        return "Ultra"
    default:
        return "Unknown"
    }
}

// GetPlanAllowance returns the total character allowance for current plan
func (u *User) GetPlanAllowance() int {
    return PlanCredits[u.SubscriptionPlan]
}

// IsAccountLocked checks if the user account is currently locked
func (u *User) IsAccountLocked() bool {
    return u.LockedUntil != nil && time.Now().Before(*u.LockedUntil)
}

// LockAccount locks the user account for a specified duration
func (u *User) LockAccount(duration time.Duration) {
    lockUntil := time.Now().Add(duration)
    u.LockedUntil = &lockUntil
}

// UnlockAccount unlocks the user account
func (u *User) UnlockAccount() {
    u.LockedUntil = nil
    u.FailedLoginAttempts = 0
}

// IncrementFailedLogin increments failed login attempts
func (u *User) IncrementFailedLogin() {
    u.FailedLoginAttempts++
    now := time.Now()
    u.LastFailedLoginAt = &now
}

type BalanceUpdate struct {
    UserID uint
    Amount int
}
