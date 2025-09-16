// File: internal/dtos/user.go
package dtos

import (
    "fmt"
    "time"
    "github.com/iyunix/go-internist/internal/domain"
)

// Pricing Configuration - Adjustable pricing per character
const (
    CharCostInToman = 5  // Cost per character in Toman (adjustable)
)

// UserResponseDTO defines what fields to expose in user API responses.
// Sensitive fields like password, verification codes, or internal security flags are excluded.
type UserResponseDTO struct {
    ID               uint   `json:"id"`
    Username         string `json:"username"`
    PhoneNumber      string `json:"phone_number"`      // Masked for privacy
    IsVerified       bool   `json:"is_verified"`
    Status           string `json:"status"`
    SubscriptionPlan string `json:"subscription_plan"`
    PlanDisplayName  string `json:"plan_display_name"`
    CharacterBalance int    `json:"character_balance"`
    TotalAllowance   int    `json:"total_allowance"`   // Plan's total character limit
    IsAdmin          bool   `json:"is_admin"`
    CreatedAt        string `json:"created_at"`
    UpdatedAt        string `json:"updated_at"`
}

// UserCreateRequestDTO represents the expected payload to create a new user.
type UserCreateRequestDTO struct {
    Username    string `json:"username" validate:"required,min=3,max=20,alphanum"`
    PhoneNumber string `json:"phone_number" validate:"required,e164"`
    Password    string `json:"password" validate:"required,min=8,max=128"`
}

// UserUpdateRequestDTO represents the payload to update user information.
type UserUpdateRequestDTO struct {
    Username *string `json:"username,omitempty" validate:"omitempty,min=3,max=20,alphanum"`
}

// UserLoginRequestDTO represents the login payload.
type UserLoginRequestDTO struct {
    PhoneNumber string `json:"phone_number" validate:"required,e164"`
    Password    string `json:"password" validate:"required,min=1"`
}

// UserLoginResponseDTO represents the login response.
type UserLoginResponseDTO struct {
    User  UserResponseDTO `json:"user"`
    Token string          `json:"token"`
}

// UserBalanceResponseDTO represents user balance information.
type UserBalanceResponseDTO struct {
    UserID                uint   `json:"user_id"`
    CharacterBalance      int    `json:"character_balance"`
    TotalCharacterBalance int    `json:"total_character_balance"`
    SubscriptionPlan      string `json:"subscription_plan"`
    PlanDisplayName       string `json:"plan_display_name"`
    PlanAllowance         int    `json:"plan_allowance"`
    MonthlyPrice          string `json:"monthly_price"` // Price in Toman
}

// UserVerificationRequestDTO represents SMS verification payload.
type UserVerificationRequestDTO struct {
    PhoneNumber string `json:"phone_number" validate:"required,e164"`
    Code        string `json:"code" validate:"required,len=6,numeric"`
}

// UserPasswordResetRequestDTO represents password reset request payload.
type UserPasswordResetRequestDTO struct {
    PhoneNumber string `json:"phone_number" validate:"required,e164"`
}

// UserPasswordResetConfirmDTO represents password reset confirmation payload.
type UserPasswordResetConfirmDTO struct {
    PhoneNumber string `json:"phone_number" validate:"required,e164"`
    Code        string `json:"code" validate:"required,len=6,numeric"`
    NewPassword string `json:"new_password" validate:"required,min=8,max=128"`
}

// SubscriptionPlanDTO represents subscription plan information with dynamic pricing
type SubscriptionPlanDTO struct {
    Name            string `json:"name"`
    Value           string `json:"value"`
    CharacterLimit  int    `json:"character_limit"`
    MonthlyPrice    string `json:"monthly_price"`    // Calculated price in Toman
    PricePerChar    int    `json:"price_per_char"`   // Current rate per character
    IsCurrentPlan   bool   `json:"is_current_plan"`
    CanUpgrade      bool   `json:"can_upgrade"`
    Savings         string `json:"savings,omitempty"` // Compared to basic plan
}

// AdminUserResponseDTO includes additional fields for admin endpoints.
type AdminUserResponseDTO struct {
    ID                    uint    `json:"id"`
    Username              string  `json:"username"`
    PhoneNumber           string  `json:"phone_number"` // Not masked for admin
    IsVerified            bool    `json:"is_verified"`
    Status                string  `json:"status"`
    SubscriptionPlan      string  `json:"subscription_plan"`
    PlanDisplayName       string  `json:"plan_display_name"`
    CharacterBalance      int     `json:"character_balance"`
    TotalCharacterBalance int     `json:"total_character_balance"`
    PlanAllowance         int     `json:"plan_allowance"`
    MonthlyPrice          string  `json:"monthly_price"`
    IsAdmin               bool    `json:"is_admin"`
    FailedLoginAttempts   int     `json:"failed_login_attempts"`
    IsLocked              bool    `json:"is_locked"`
    CreatedAt             string  `json:"created_at"`
    UpdatedAt             string  `json:"updated_at"`
    LastFailedLoginAt     *string `json:"last_failed_login_at,omitempty"`
}

// SubscriptionUpgradeRequestDTO represents plan upgrade request
type SubscriptionUpgradeRequestDTO struct {
    NewPlan string `json:"new_plan" validate:"required,oneof=basic pro premium ultra"`
}

// PricingInfoDTO provides current pricing information
type PricingInfoDTO struct {
    CharCostInToman   int    `json:"char_cost_in_toman"`
    Currency          string `json:"currency"`
    LastUpdated       string `json:"last_updated"`
    BasicPlanPrice    string `json:"basic_plan_price"`
    ProPlanPrice      string `json:"pro_plan_price"`
    PremiumPlanPrice  string `json:"premium_plan_price"`
    UltraPlanPrice    string `json:"ultra_plan_price"`
}

// Pricing Calculation Functions

// CalculateMonthlyPrice calculates the monthly price in Toman for given character limit
func CalculateMonthlyPrice(charLimit int) int {
    return charLimit * CharCostInToman
}

// FormatPriceInToman formats price with Toman currency
func FormatPriceInToman(price int) string {
    if price == 0 {
        return "Free" // Changed from Persian "رایگان" to English "Free"
    }
    return fmt.Sprintf("%d تومان", price)
}

// CalculateSavings calculates savings compared to basic plan
func CalculateSavings(currentPlanChars, basicPlanChars int) string {
    if currentPlanChars <= basicPlanChars {
        return ""
    }
    
    currentPrice := CalculateMonthlyPrice(currentPlanChars)
    basicPrice := CalculateMonthlyPrice(basicPlanChars)
    
    if basicPrice == 0 {
        return ""
    }
    
    savings := currentPrice - basicPrice
    savingsPerChar := float64(savings) / float64(currentPlanChars - basicPlanChars)
    
    return fmt.Sprintf("%.1f Toman per character savings", savingsPerChar)
}

// Mapping Functions

// FromDomain maps a domain.User to UserResponseDTO for public API responses.
func FromDomain(user domain.User) UserResponseDTO {
    return UserResponseDTO{
        ID:               user.ID,
        Username:         user.Username,
        PhoneNumber:      maskPhoneNumber(user.PhoneNumber),
        IsVerified:       user.IsVerified,
        Status:           string(user.Status),
        SubscriptionPlan: string(user.SubscriptionPlan),
        PlanDisplayName:  user.GetPlanName(),
        CharacterBalance: user.CharacterBalance,
        TotalAllowance:   user.GetPlanAllowance(),
        IsAdmin:          user.IsAdmin,
        CreatedAt:        user.CreatedAt.Format(time.RFC3339),
        UpdatedAt:        user.UpdatedAt.Format(time.RFC3339),
    }
}

// ToAdminDomain maps a domain.User to AdminUserResponseDTO for admin endpoints.
func ToAdminDomain(user domain.User) AdminUserResponseDTO {
    monthlyPrice := CalculateMonthlyPrice(user.GetPlanAllowance())
    
    dto := AdminUserResponseDTO{
        ID:                    user.ID,
        Username:              user.Username,
        PhoneNumber:           user.PhoneNumber, // Not masked for admin
        IsVerified:            user.IsVerified,
        Status:                string(user.Status),
        SubscriptionPlan:      string(user.SubscriptionPlan),
        PlanDisplayName:       user.GetPlanName(),
        CharacterBalance:      user.CharacterBalance,
        TotalCharacterBalance: user.TotalCharacterBalance,
        PlanAllowance:         user.GetPlanAllowance(),
        MonthlyPrice:          FormatPriceInToman(monthlyPrice),
        IsAdmin:               user.IsAdmin,
        FailedLoginAttempts:   user.FailedLoginAttempts,
        IsLocked:              user.IsAccountLocked(),
        CreatedAt:             user.CreatedAt.Format(time.RFC3339),
        UpdatedAt:             user.UpdatedAt.Format(time.RFC3339),
    }

    // Format optional timestamp
    if user.LastFailedLoginAt != nil {
        formatted := user.LastFailedLoginAt.Format(time.RFC3339)
        dto.LastFailedLoginAt = &formatted
    }

    return dto
}

// ToDomain maps UserCreateRequestDTO to a domain.User for persistence.
func (dto UserCreateRequestDTO) ToDomain() domain.User {
    return domain.User{
        Username:    dto.Username,
        PhoneNumber: dto.PhoneNumber,
        Password:    dto.Password, // Note: Hash password before saving in service layer!
        Status:      domain.UserStatusPending,
        SubscriptionPlan: domain.PlanBasic,
        CharacterBalance: domain.DefaultRegistrationChars, // 2,500 characters
        TotalCharacterBalance: domain.DefaultRegistrationChars,
    }
}

// ToBalanceResponse maps a domain.User to UserBalanceResponseDTO.
func ToBalanceResponse(user domain.User) UserBalanceResponseDTO {
    monthlyPrice := CalculateMonthlyPrice(user.GetPlanAllowance())
    
    return UserBalanceResponseDTO{
        UserID:                user.ID,
        CharacterBalance:      user.CharacterBalance,
        TotalCharacterBalance: user.TotalCharacterBalance,
        SubscriptionPlan:      string(user.SubscriptionPlan),
        PlanDisplayName:       user.GetPlanName(),
        PlanAllowance:         user.GetPlanAllowance(),
        MonthlyPrice:          FormatPriceInToman(monthlyPrice),
    }
}

// GetAvailablePlans returns all available subscription plans with calculated Toman pricing.
func GetAvailablePlans(currentPlan domain.SubscriptionPlan) []SubscriptionPlanDTO {
    basicChars := domain.PlanCredits[domain.PlanBasic]
    
    plans := []SubscriptionPlanDTO{
        {
            Name:           "Basic", // Changed from Persian "پایه"
            Value:          "basic",
            CharacterLimit: 25000,
            MonthlyPrice:   FormatPriceInToman(CalculateMonthlyPrice(25000)),
            PricePerChar:   CharCostInToman,
            IsCurrentPlan:  currentPlan == domain.PlanBasic,
            CanUpgrade:     currentPlan != domain.PlanBasic,
        },
        {
            Name:           "Pro", // Changed from Persian "حرفه‌ای"
            Value:          "pro", 
            CharacterLimit: 50000,
            MonthlyPrice:   FormatPriceInToman(CalculateMonthlyPrice(50000)),
            PricePerChar:   CharCostInToman,
            IsCurrentPlan:  currentPlan == domain.PlanPro,
            CanUpgrade:     canUpgradeTo(currentPlan, domain.PlanPro),
            Savings:        CalculateSavings(50000, basicChars),
        },
        {
            Name:           "Premium", // Changed from Persian "پریمیوم"
            Value:          "premium",
            CharacterLimit: 100000,
            MonthlyPrice:   FormatPriceInToman(CalculateMonthlyPrice(100000)),
            PricePerChar:   CharCostInToman,
            IsCurrentPlan:  currentPlan == domain.PlanPremium,
            CanUpgrade:     canUpgradeTo(currentPlan, domain.PlanPremium),
            Savings:        CalculateSavings(100000, basicChars),
        },
        {
            Name:           "Ultra", // Changed from Persian "فوق‌العاده"
            Value:          "ultra",
            CharacterLimit: 500000,
            MonthlyPrice:   FormatPriceInToman(CalculateMonthlyPrice(500000)),
            PricePerChar:   CharCostInToman,
            IsCurrentPlan:  currentPlan == domain.PlanUltra,
            CanUpgrade:     canUpgradeTo(currentPlan, domain.PlanUltra),
            Savings:        CalculateSavings(500000, basicChars),
        },
    }
    return plans
}

// GetCurrentPricingInfo returns current pricing configuration
func GetCurrentPricingInfo() PricingInfoDTO {
    return PricingInfoDTO{
        CharCostInToman:  CharCostInToman,
        Currency:         "تومان",
        LastUpdated:      time.Now().Format(time.RFC3339),
        BasicPlanPrice:   FormatPriceInToman(CalculateMonthlyPrice(25000)),
        ProPlanPrice:     FormatPriceInToman(CalculateMonthlyPrice(50000)),
        PremiumPlanPrice: FormatPriceInToman(CalculateMonthlyPrice(100000)),
        UltraPlanPrice:   FormatPriceInToman(CalculateMonthlyPrice(500000)),
    }
}

// Helper Functions

// maskPhoneNumber partially masks phone numbers for privacy in public responses.
func maskPhoneNumber(phone string) string {
    if len(phone) < 7 {
        return phone // Too short to mask meaningfully
    }
    
    // Show first 3 and last 2 digits: +98937****77
    if len(phone) >= 10 {
        return phone[:5] + "****" + phone[len(phone)-2:]
    }
    
    // Fallback for shorter numbers
    return phone[:3] + "****" + phone[len(phone)-2:]
}

// canUpgradeTo checks if current plan can be upgraded to target plan
func canUpgradeTo(currentPlan, targetPlan domain.SubscriptionPlan) bool {
    currentCredits := domain.PlanCredits[currentPlan]
    targetCredits := domain.PlanCredits[targetPlan]
    return targetCredits > currentCredits
}

// Batch Mapping Functions

// FromDomainSlice maps a slice of domain.User to []UserResponseDTO.
func FromDomainSlice(users []domain.User) []UserResponseDTO {
    dtos := make([]UserResponseDTO, len(users))
    for i, user := range users {
        dtos[i] = FromDomain(user)
    }
    return dtos
}

// ToAdminDomainSlice maps a slice of domain.User to []AdminUserResponseDTO.
func ToAdminDomainSlice(users []domain.User) []AdminUserResponseDTO {
    dtos := make([]AdminUserResponseDTO, len(users))
    for i, user := range users {
        dtos[i] = ToAdminDomain(user)
    }
    return dtos
}

// Validation & Business Logic Helpers

// IsValidPhoneNumber performs basic phone number validation.
func IsValidPhoneNumber(phone string) bool {
    return len(phone) >= 10 && len(phone) <= 15
}

// IsValidSubscriptionPlan checks if a plan string is valid
func IsValidSubscriptionPlan(plan string) bool {
    validPlans := []string{"basic", "pro", "premium", "ultra"}
    for _, validPlan := range validPlans {
        if plan == validPlan {
            return true
        }
    }
    return false
}

// GetPlanByValue returns SubscriptionPlan enum from string value
func GetPlanByValue(planValue string) (domain.SubscriptionPlan, bool) {
    switch planValue {
    case "basic":
        return domain.PlanBasic, true
    case "pro":
        return domain.PlanPro, true
    case "premium":
        return domain.PlanPremium, true
    case "ultra":
        return domain.PlanUltra, true
    default:
        return "", false
    }
}

// Response wrapper DTOs for consistent API responses

// SuccessResponse represents a successful API response
type SuccessResponse struct {
    Success bool        `json:"success"`
    Data    interface{} `json:"data"`
    Message string      `json:"message,omitempty"`
}

// ErrorResponse represents an error API response
type ErrorResponse struct {
    Success bool     `json:"success"`
    Error   string   `json:"error"`
    Details []string `json:"details,omitempty"`
}

// PaginatedResponse represents a paginated API response
type PaginatedResponse struct {
    Success    bool        `json:"success"`
    Data       interface{} `json:"data"`
    Pagination struct {
        Page       int `json:"page"`
        PerPage    int `json:"per_page"`
        Total      int `json:"total"`
        TotalPages int `json:"total_pages"`
    } `json:"pagination"`
}

// CreateSuccessResponse creates a standard success response
func CreateSuccessResponse(data interface{}, message string) SuccessResponse {
    return SuccessResponse{
        Success: true,
        Data:    data,
        Message: message,
    }
}

// CreateErrorResponse creates a standard error response
func CreateErrorResponse(error string, details []string) ErrorResponse {
    return ErrorResponse{
        Success: false,
        Error:   error,
        Details: details,
    }
}
