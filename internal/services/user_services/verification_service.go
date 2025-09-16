// File: internal/services/user_services/verification_service.go
package user_services

import (
    "context"
    "crypto/rand"
    "errors"
    "fmt"
    "math/big"
    "time"

    "github.com/iyunix/go-internist/internal/domain"
    "github.com/iyunix/go-internist/internal/repository/user"
    "github.com/iyunix/go-internist/internal/repository/verification"
    "github.com/iyunix/go-internist/internal/services"
    "golang.org/x/crypto/bcrypt"
)

// VerificationService handles SMS verification workflows using VerificationCode table
type VerificationService struct {
    userRepo         user.UserRepository
    verificationRepo verification.VerificationRepository
    smsService       *services.SMSService
    authService      *AuthService
    logger           services.Logger
}

// NewVerificationService creates a new verification service
func NewVerificationService(
    userRepo user.UserRepository, 
    verificationRepo verification.VerificationRepository,
    smsService *services.SMSService, 
    authService *AuthService, 
    logger services.Logger,
) *VerificationService {
    return &VerificationService{
        userRepo:         userRepo,
        verificationRepo: verificationRepo,
        smsService:       smsService,
        authService:      authService,
        logger:           logger,
    }
}

// SendVerificationCode generates and sends a verification code to the user
func (s *VerificationService) SendVerificationCode(ctx context.Context, userID uint) error {
    if userID == 0 {
        s.logger.Warn("verification code send attempted with invalid user ID", "user_id", userID)
        return errors.New("user ID must be provided")
    }

    s.logger.Info("sending verification code", "user_id", userID)
    
    user, err := s.userRepo.FindByID(ctx, userID)
    if err != nil {
        s.logger.Error("failed to find user for verification", "error", err, "user_id", userID)
        return fmt.Errorf("failed to find user: %w", err)
    }

    if user.IsVerified {
        s.logger.Info("verification code requested for already verified user", "user_id", userID)
        return errors.New("user is already verified")
    }

    // Check for existing verification code and rate limiting
    existingCode, err := s.verificationRepo.FindByPhoneAndType(ctx, user.PhoneNumber, domain.VerificationTypeSMS)
    if err != nil {
        s.logger.Error("failed to check existing verification code", "error", err)
        return fmt.Errorf("failed to check existing verification code: %w", err)
    }

    if existingCode != nil && time.Since(existingCode.CreatedAt) < time.Minute {
        s.logger.Warn("verification code rate limited", "user_id", userID)
        return errors.New("please wait before requesting another code")
    }

    // Delete any existing verification codes
    if err := s.verificationRepo.DeleteByPhone(ctx, user.PhoneNumber, domain.VerificationTypeSMS); err != nil {
        s.logger.Warn("failed to delete existing verification codes", "error", err)
    }

    // Generate new verification code
    code, err := s.generateVerificationCode()
    if err != nil {
        s.logger.Error("failed to generate verification code", "error", err)
        return fmt.Errorf("failed to generate verification code: %w", err)
    }

    // Create verification record
    verification := &domain.VerificationCode{
        PhoneNumber: user.PhoneNumber,
        Code:        code,
        Type:        domain.VerificationTypeSMS,
        ExpiresAt:   time.Now().Add(10 * time.Minute),
        MaxAttempts: 3,
    }

    if err := s.verificationRepo.Create(ctx, verification); err != nil {
        s.logger.Error("failed to save verification code", "error", err, "user_id", userID)
        return fmt.Errorf("failed to save verification code: %w", err)
    }

    // Send SMS
    if err := s.smsService.SendVerificationCode(ctx, user.PhoneNumber, code); err != nil {
        s.logger.Error("SMS sending failed", "error", err, "user_id", userID)
        return fmt.Errorf("failed to send SMS: %w", err)
    }

    s.logger.Info("verification code sent successfully", "user_id", userID)
    return nil
}

// VerifyCode verifies the provided code and marks user as verified
func (s *VerificationService) VerifyCode(ctx context.Context, userID uint, code string) error {
    if userID == 0 || code == "" || len(code) != 6 {
        return errors.New("invalid input for verification")
    }

    s.logger.Info("verifying code", "user_id", userID)
    
    user, err := s.userRepo.FindByID(ctx, userID)
    if err != nil {
        return fmt.Errorf("failed to find user: %w", err)
    }

    if user.IsVerified {
        return errors.New("user is already verified")
    }

    // Find verification code
    verification, err := s.verificationRepo.FindByPhoneAndType(ctx, user.PhoneNumber, domain.VerificationTypeSMS)
    if err != nil {
        s.logger.Error("failed to find verification code", "error", err)
        return fmt.Errorf("failed to find verification code: %w", err)
    }

    if verification == nil {
        return errors.New("no verification code found")
    }

    // Check if code is valid
    if !verification.IsValid() {
        return errors.New("verification code has expired or is invalid")
    }

    if !verification.CanAttempt() {
        return errors.New("maximum verification attempts exceeded")
    }

    // Verify code
    if verification.Code != code {
        verification.IncrementAttempt()
        s.verificationRepo.Update(ctx, verification)
        return errors.New("invalid verification code")
    }

    // Mark user as verified
    user.IsVerified = true
    user.Status = domain.UserStatusActive
    now := time.Now()
    user.VerifiedAt = &now

    if err := s.userRepo.Update(ctx, user); err != nil {
        return fmt.Errorf("failed to save verification status: %w", err)
    }

    // Mark verification code as used and cleanup
    verification.UseCode()
    s.verificationRepo.Update(ctx, verification)
    s.verificationRepo.DeleteByPhone(ctx, user.PhoneNumber, domain.VerificationTypeSMS)

    s.logger.Info("user verified successfully", "user_id", userID)
    return nil
}

// ResendVerificationCode sends a new verification code
func (s *VerificationService) ResendVerificationCode(ctx context.Context, userID uint) error {
    if userID == 0 {
        return errors.New("user ID must be provided")
    }

    s.logger.Info("resending verification code", "user_id", userID)
    
    user, err := s.userRepo.FindByID(ctx, userID)
    if err != nil {
        return fmt.Errorf("failed to find user: %w", err)
    }

    // Check rate limiting
    existingCode, err := s.verificationRepo.FindByPhoneAndType(ctx, user.PhoneNumber, domain.VerificationTypeSMS)
    if err == nil && existingCode != nil && time.Since(existingCode.CreatedAt) < 2*time.Minute {
        return errors.New("please wait before requesting another code")
    }

    return s.SendVerificationCode(ctx, userID)
}

// SendPasswordResetCode finds a user by phone, generates a reset code, and sends it
func (s *VerificationService) SendPasswordResetCode(ctx context.Context, phone string) error {
    s.logger.Info("password reset requested", "phone", phone)
    
    user, err := s.userRepo.FindByPhoneNumber(ctx, phone)
    if err != nil {
        // Security: Do not reveal if user exists
        s.logger.Warn("password reset requested for non-existent phone number", "phone", phone, "error", err)
        return nil // Return success to avoid user enumeration
    }

    // Delete any existing password reset codes
    if err := s.verificationRepo.DeleteByPhone(ctx, phone, domain.VerificationTypePassword); err != nil {
        s.logger.Warn("failed to delete existing password reset codes", "error", err)
    }

    // Generate reset code
    code, err := s.generateVerificationCode()
    if err != nil {
        s.logger.Error("failed to generate password reset code", "error", err)
        return err
    }

    // Create password reset verification record
    verification := &domain.VerificationCode{
        PhoneNumber: phone,
        Code:        code,
        Type:        domain.VerificationTypePassword,
        ExpiresAt:   time.Now().Add(10 * time.Minute),
        MaxAttempts: 3,
    }

    if err := s.verificationRepo.Create(ctx, verification); err != nil {
        s.logger.Error("failed to save password reset code", "error", err, "user_id", user.ID)
        return err
    }

    // Send SMS
    if err := s.smsService.SendVerificationCode(ctx, user.PhoneNumber, code); err != nil {
        s.logger.Error("failed to send password reset SMS", "error", err, "user_id", user.ID)
        return err
    }

    s.logger.Info("password reset code sent successfully", "user_id", user.ID)
    return nil
}

// VerifyPasswordResetCode checks if a password reset code is valid
func (s *VerificationService) VerifyPasswordResetCode(ctx context.Context, userID uint, code string) error {
    if userID == 0 || code == "" || len(code) != 6 {
        return errors.New("invalid input for reset code verification")
    }

    s.logger.Info("verifying password reset code", "user_id", userID)

    user, err := s.userRepo.FindByID(ctx, userID)
    if err != nil {
        return fmt.Errorf("failed to find user: %w", err)
    }

    // Find password reset verification
    verification, err := s.verificationRepo.FindByPhoneAndType(ctx, user.PhoneNumber, domain.VerificationTypePassword)
    if err != nil {
        return fmt.Errorf("failed to find password reset code: %w", err)
    }

    if verification == nil {
        return errors.New("no password reset has been requested")
    }

    if !verification.IsValid() {
        return errors.New("password reset code has expired")
    }

    if !verification.CanAttempt() {
        return errors.New("maximum password reset attempts exceeded")
    }

    if verification.Code != code {
        verification.IncrementAttempt()
        s.verificationRepo.Update(ctx, verification)
        return errors.New("invalid password reset code")
    }

    return nil
}

// VerifyAndResetPassword validates the reset code and updates the user's password
func (s *VerificationService) VerifyAndResetPassword(ctx context.Context, phone, code, newPassword string) error {
    s.logger.Info("attempting to reset password", "phone", phone)
    
    user, err := s.userRepo.FindByPhoneNumber(ctx, phone)
    if err != nil {
        s.logger.Warn("password reset attempt for non-existent phone", "phone", phone, "error", err)
        return errors.New("invalid code or phone number")
    }

    // Find and verify password reset code
    verification, err := s.verificationRepo.FindByPhoneAndType(ctx, phone, domain.VerificationTypePassword)
    if err != nil || verification == nil {
        s.logger.Warn("no password reset code found", "phone", phone)
        return errors.New("invalid code or phone number")
    }

    if verification.Code != code {
        s.logger.Warn("invalid password reset code provided", "user_id", user.ID)
        return errors.New("invalid code or phone number")
    }

    if !verification.IsValid() {
        s.logger.Warn("expired password reset code used", "user_id", user.ID)
        return errors.New("reset code has expired")
    }

    // Hash the new password securely
    hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
    if err != nil {
        s.logger.Error("failed to hash new password", "error", err, "user_id", user.ID)
        return errors.New("internal server error")
    }

    user.Password = string(hashedPassword)

    if err := s.userRepo.Update(ctx, user); err != nil {
        s.logger.Error("failed to save new password", "error", err, "user_id", user.ID)
        return errors.New("failed to update password")
    }

    // Cleanup - mark as used and delete
    verification.UseCode()
    s.verificationRepo.Update(ctx, verification)
    s.verificationRepo.DeleteByPhone(ctx, phone, domain.VerificationTypePassword)

    s.logger.Info("password reset successfully", "user_id", user.ID)
    return nil
}

// generateVerificationCode creates a secure 6-digit verification code
func (s *VerificationService) generateVerificationCode() (string, error) {
    // Use crypto/rand for secure random generation
    max := big.NewInt(1000000) // 0-999999
    n, err := rand.Int(rand.Reader, max)
    if err != nil {
        return "", err
    }
    return fmt.Sprintf("%06d", n.Int64()), nil
}
