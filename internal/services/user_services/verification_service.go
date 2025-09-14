// File: internal/services/user_services/verification_service.go
package user_services

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"
	"github.com/iyunix/go-internist/internal/repository/user"
	"golang.org/x/crypto/bcrypt" // <-- ADD THIS IMPORT
)

// SMSService interface for sending verification codes
type SMSService interface {
	SendVerificationCode(ctx context.Context, phone, code string) error
}

// VerificationService handles SMS verification workflows
type VerificationService struct {
	userRepo   user.UserRepository
	smsService SMSService
	logger     Logger
	authService *AuthService // Added for password hashing
}

// NewVerificationService creates a new verification service
func NewVerificationService(userRepo user.UserRepository, smsService SMSService, authService *AuthService, logger Logger) *VerificationService {
	return &VerificationService{
		userRepo:   userRepo,
		smsService: smsService,
		authService: authService,
		logger:     logger,
	}
}


// --- EXISTING METHODS (Unchanged but included for completeness) ---

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
	if user.VerificationCodeSentAt != nil {
		if time.Since(*user.VerificationCodeSentAt) < time.Minute {
			s.logger.Warn("verification code rate limited", "user_id", userID)
			return errors.New("please wait before requesting another code")
		}
	}
	code := s.generateVerificationCode()
	now := time.Now()
	expires := now.Add(10 * time.Minute)
	user.VerificationCode = code
	user.VerificationCodeSentAt = &now
	user.VerificationCodeExpiresAt = &expires
	if err := s.userRepo.Update(ctx, user); err != nil {
		s.logger.Error("failed to save verification code", "error", err, "user_id", userID)
		return fmt.Errorf("failed to save verification code: %w", err)
	}
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
	if user.VerificationCode == "" {
		return errors.New("no verification code found")
	}
	if user.VerificationCodeExpiresAt == nil || time.Now().After(*user.VerificationCodeExpiresAt) {
		return errors.New("verification code has expired")
	}
	if user.VerificationCode != code {
		return errors.New("invalid verification code")
	}
	user.IsVerified = true
	user.VerificationCode = ""
	user.VerificationCodeSentAt = nil
	user.VerificationCodeExpiresAt = nil
	now := time.Now()
	user.VerifiedAt = &now
	if err := s.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("failed to save verification status: %w", err)
	}
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
	if user.VerificationCodeSentAt != nil {
		if time.Since(*user.VerificationCodeSentAt) < 2*time.Minute {
			return errors.New("please wait before requesting another code")
		}
	}
	return s.SendVerificationCode(ctx, userID)
}

// --- NEW METHODS FOR PASSWORD RESET ---

// SendPasswordResetCode finds a user by phone, generates a reset code, and sends it.
func (s *VerificationService) SendPasswordResetCode(ctx context.Context, phone string) error {
	s.logger.Info("password reset requested", "phone", phone)
	user, err := s.userRepo.FindByPhone(ctx, phone)
	if err != nil {
		// Security: Do not reveal if the user exists. Log the error and return nil.
		s.logger.Warn("password reset requested for non-existent phone number", "phone", phone, "error", err)
		return nil // Return nil to the handler to show a generic success message
	}

	code := s.generateVerificationCode()
	now := time.Now()
	expires := now.Add(10 * time.Minute) // Reset codes are also valid for 10 minutes

	user.PasswordResetCode = code
	user.PasswordResetExpiresAt = &expires

	if err := s.userRepo.Update(ctx, user); err != nil {
		s.logger.Error("failed to save password reset code", "error", err, "user_id", user.ID)
		return err // Return the actual error here, as something is wrong internally
	}

	if err := s.smsService.SendVerificationCode(ctx, user.PhoneNumber, code); err != nil {
		s.logger.Error("failed to send password reset SMS", "error", err, "user_id", user.ID)
		return err
	}

	s.logger.Info("password reset code sent successfully", "user_id", user.ID)
	return nil
}

// VerifyAndResetPassword validates the reset code and updates the user's password.
func (s *VerificationService) VerifyAndResetPassword(ctx context.Context, phone, code, newPassword string) error {
	s.logger.Info("attempting to reset password", "phone", phone)
	user, err := s.userRepo.FindByPhone(ctx, phone)
	if err != nil {
		s.logger.Warn("password reset attempt for non-existent phone", "phone", phone, "error", err)
		return errors.New("invalid code or phone number")
	}

	if user.PasswordResetCode == "" || user.PasswordResetCode != code {
		s.logger.Warn("invalid password reset code provided", "user_id", user.ID)
		return errors.New("invalid code or phone number")
	}

	if user.PasswordResetExpiresAt == nil || time.Now().After(*user.PasswordResetExpiresAt) {
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
	// IMPORTANT: Clear the reset code so it cannot be used again
	user.PasswordResetCode = ""
	user.PasswordResetExpiresAt = nil

	if err := s.userRepo.Update(ctx, user); err != nil {
		s.logger.Error("failed to save new password", "error", err, "user_id", user.ID)
		return errors.New("failed to update password")
	}

	s.logger.Info("password reset successfully", "user_id", user.ID)
	return nil
}

// generateVerificationCode creates a 6-digit verification code
func (s *VerificationService) generateVerificationCode() string {
	// Seeding is now implicitly handled by Go's math/rand since Go 1.20
	return fmt.Sprintf("%06d", rand.Intn(1000000))
}

// Note: The VerificationStatus struct and CheckVerificationStatus method are omitted for brevity,
// as they are not changed. You can keep them as they are in your file.