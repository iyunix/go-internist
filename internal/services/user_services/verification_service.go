// G:\go_internist\internal\services\user_services\verification_service.go
package user_services

import (
    "context"
    "errors"
    "fmt"
    "math/rand"
    "strconv"
    "time"
    "github.com/iyunix/go-internist/internal/repository"
)

// SMSService interface for sending verification codes
type SMSService interface {
    SendVerificationCode(ctx context.Context, phone, code string) error
}

// VerificationService handles SMS verification workflows
type VerificationService struct {
    userRepo   repository.UserRepository
    smsService SMSService
    logger     Logger
}

// NewVerificationService creates a new verification service
func NewVerificationService(userRepo repository.UserRepository, smsService SMSService, logger Logger) *VerificationService {
    return &VerificationService{
        userRepo:   userRepo,
        smsService: smsService,
        logger:     logger,
    }
}

// SendVerificationCode generates and sends a verification code to the user
func (s *VerificationService) SendVerificationCode(ctx context.Context, userID uint) error {
    if userID == 0 {
        s.logger.Warn("verification code send attempted with invalid user ID", "user_id", userID)
        return errors.New("user ID must be provided")
    }

    s.logger.Info("sending verification code", "user_id", userID)

    // Find user
    user, err := s.userRepo.FindByID(ctx, userID)
    if err != nil {
        s.logger.Error("failed to find user for verification", 
            "error", err,
            "user_id", userID)
        return fmt.Errorf("failed to find user: %w", err)
    }

    // Check if user is already verified
    if user.IsVerified {
        s.logger.Info("verification code requested for already verified user", 
            "user_id", userID,
            "phone", user.PhoneNumber[:min(4, len(user.PhoneNumber))]+"****")  // FIXED
        return errors.New("user is already verified")
    }

    // Check rate limiting (prevent spam)
    if user.VerificationCodeSentAt != nil {
        timeSinceLastCode := time.Since(*user.VerificationCodeSentAt)
        if timeSinceLastCode < time.Minute {
            s.logger.Warn("verification code rate limited", 
                "user_id", userID,
                "time_since_last", timeSinceLastCode.String(),
                "required_wait", "1 minute")
            return errors.New("please wait before requesting another code")
        }
    }

    // Generate verification code
    code := s.generateVerificationCode()
    now := time.Now()

    // Update user with verification code
    user.VerificationCode = code
    user.VerificationCodeSentAt = &now
    user.VerificationCodeExpiresAt = &[]time.Time{now.Add(10 * time.Minute)}[0] // 10 minutes

    if err := s.userRepo.Update(ctx, user); err != nil {
        s.logger.Error("failed to save verification code", 
            "error", err,
            "user_id", userID)
        return fmt.Errorf("failed to save verification code: %w", err)
    }

    // Send SMS
    if err := s.smsService.SendVerificationCode(ctx, user.PhoneNumber, code); err != nil {  // FIXED
        s.logger.Error("SMS sending failed", 
            "error", err,
            "user_id", userID,
            "phone", user.PhoneNumber[:min(4, len(user.PhoneNumber))]+"****")  // FIXED
        return fmt.Errorf("failed to send SMS: %w", err)
    }

    s.logger.Info("verification code sent successfully", 
        "user_id", userID,
        "phone", user.PhoneNumber[:min(4, len(user.PhoneNumber))]+"****",  // FIXED
        "code_length", len(code),
        "expires_at", user.VerificationCodeExpiresAt.Format(time.RFC3339))

    return nil
}

// VerifyCode verifies the provided code and marks user as verified
func (s *VerificationService) VerifyCode(ctx context.Context, userID uint, code string) error {
    if userID == 0 {
        s.logger.Warn("code verification attempted with invalid user ID", "user_id", userID)
        return errors.New("user ID must be provided")
    }

    if code == "" {
        s.logger.Warn("code verification attempted with empty code", "user_id", userID)
        return errors.New("verification code is required")
    }

    if len(code) != 6 {
        s.logger.Warn("code verification attempted with invalid code length", 
            "user_id", userID,
            "code_length", len(code))
        return errors.New("verification code must be 6 digits")
    }

    s.logger.Info("verifying code", 
        "user_id", userID,
        "code_length", len(code))

    // Find user
    user, err := s.userRepo.FindByID(ctx, userID)
    if err != nil {
        s.logger.Error("failed to find user for code verification", 
            "error", err,
            "user_id", userID)
        return fmt.Errorf("failed to find user: %w", err)
    }

    // Check if user is already verified
    if user.IsVerified {
        s.logger.Info("code verification attempted for already verified user", 
            "user_id", userID,
            "phone", user.PhoneNumber[:min(4, len(user.PhoneNumber))]+"****")  // FIXED
        return errors.New("user is already verified")
    }

    // Check if verification code exists
    if user.VerificationCode == "" {
        s.logger.Warn("code verification attempted without active code", 
            "user_id", userID)
        return errors.New("no verification code found")
    }

    // Check if verification code is expired
    if user.VerificationCodeExpiresAt == nil || time.Now().After(*user.VerificationCodeExpiresAt) {
        s.logger.Warn("expired verification code used", 
            "user_id", userID,
            "expired_at", func() string {
                if user.VerificationCodeExpiresAt != nil {
                    return user.VerificationCodeExpiresAt.Format(time.RFC3339)
                }
                return "unknown"
            }())
        return errors.New("verification code has expired")
    }

    // Verify code
    if user.VerificationCode != code {
        s.logger.Warn("invalid verification code attempted", 
            "user_id", userID,
            "phone", user.PhoneNumber[:min(4, len(user.PhoneNumber))]+"****",  // FIXED
            "provided_code_length", len(code))
        return errors.New("invalid verification code")
    }

    // Mark user as verified and clear verification data
    user.IsVerified = true
    user.VerificationCode = ""
    user.VerificationCodeSentAt = nil
    user.VerificationCodeExpiresAt = nil
    user.VerifiedAt = &[]time.Time{time.Now()}[0]

    if err := s.userRepo.Update(ctx, user); err != nil {
        s.logger.Error("failed to save user verification status", 
            "error", err,
            "user_id", userID)
        return fmt.Errorf("failed to save verification status: %w", err)
    }

    s.logger.Info("user verified successfully", 
        "user_id", userID,
        "phone", user.PhoneNumber[:min(4, len(user.PhoneNumber))]+"****",  // FIXED
        "verified_at", user.VerifiedAt.Format(time.RFC3339))

    return nil
}

// ResendVerificationCode sends a new verification code
func (s *VerificationService) ResendVerificationCode(ctx context.Context, userID uint) error {
    if userID == 0 {
        s.logger.Warn("verification code resend attempted with invalid user ID", "user_id", userID)
        return errors.New("user ID must be provided")
    }

    s.logger.Info("resending verification code", "user_id", userID)

    // Check rate limiting more strictly for resends
    user, err := s.userRepo.FindByID(ctx, userID)
    if err != nil {
        s.logger.Error("failed to find user for code resend", 
            "error", err,
            "user_id", userID)
        return fmt.Errorf("failed to find user: %w", err)
    }

    if user.VerificationCodeSentAt != nil {
        timeSinceLastCode := time.Since(*user.VerificationCodeSentAt)
        if timeSinceLastCode < 2*time.Minute { // Stricter rate limiting for resends
            s.logger.Warn("verification code resend rate limited", 
                "user_id", userID,
                "time_since_last", timeSinceLastCode.String(),
                "required_wait", "2 minutes")
            return fmt.Errorf("please wait %v before requesting another code", 
                2*time.Minute-timeSinceLastCode)
        }
    }

    // Use the same logic as SendVerificationCode
    return s.SendVerificationCode(ctx, userID)
}

// CheckVerificationStatus returns the current verification status
func (s *VerificationService) CheckVerificationStatus(ctx context.Context, userID uint) (*VerificationStatus, error) {
    if userID == 0 {
        s.logger.Warn("verification status check with invalid user ID", "user_id", userID)
        return nil, errors.New("user ID must be provided")
    }

    user, err := s.userRepo.FindByID(ctx, userID)
    if err != nil {
        s.logger.Error("failed to find user for status check", 
            "error", err,
            "user_id", userID)
        return nil, fmt.Errorf("failed to find user: %w", err)
    }

    status := &VerificationStatus{
        UserID:     userID,
        IsVerified: user.IsVerified,
        Phone:      user.PhoneNumber[:min(4, len(user.PhoneNumber))] + "****",  // FIXED
    }

    if user.VerificationCodeSentAt != nil {
        status.CodeSentAt = user.VerificationCodeSentAt
    }

    if user.VerificationCodeExpiresAt != nil {
        status.CodeExpiresAt = user.VerificationCodeExpiresAt
        status.IsCodeExpired = time.Now().After(*user.VerificationCodeExpiresAt)
    }

    if user.VerifiedAt != nil {
        status.VerifiedAt = user.VerifiedAt
    }

    s.logger.Debug("verification status checked", 
        "user_id", userID,
        "is_verified", status.IsVerified,
        "has_active_code", status.CodeSentAt != nil)

    return status, nil
}

// generateVerificationCode creates a 6-digit verification code
func (s *VerificationService) generateVerificationCode() string {
    rand.Seed(time.Now().UnixNano())
    code := rand.Intn(900000) + 100000 // Ensures 6 digits
    return strconv.Itoa(code)
}

// VerificationStatus represents the current verification state
type VerificationStatus struct {
    UserID        uint       `json:"user_id"`
    IsVerified    bool       `json:"is_verified"`
    Phone         string     `json:"phone"`
    CodeSentAt    *time.Time `json:"code_sent_at,omitempty"`
    CodeExpiresAt *time.Time `json:"code_expires_at,omitempty"`
    IsCodeExpired bool       `json:"is_code_expired"`
    VerifiedAt    *time.Time `json:"verified_at,omitempty"`
}
