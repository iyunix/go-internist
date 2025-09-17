// File: internal/repository/verification/verification_repository.go
package verification

import (
	"context"
	"errors"
	"log"
	"time"

	"gorm.io/gorm"

	"github.com/iyunix/go-internist/internal/domain"
)

// VerificationRepository interface for verification code operations
type VerificationRepository interface {
	Create(ctx context.Context, verification *domain.VerificationCode) error
	FindByPhoneAndType(ctx context.Context, phone string, codeType domain.VerificationCodeType) (*domain.VerificationCode, error)
	DeleteByPhone(ctx context.Context, phone string, codeType domain.VerificationCodeType) error
	Update(ctx context.Context, verification *domain.VerificationCode) error
	DeleteExpired(ctx context.Context) error
}

// GormVerificationRepository implements VerificationRepository using GORM
type GormVerificationRepository struct {
	db *gorm.DB
}

// NewGormVerificationRepository creates a new verification repository
func NewGormVerificationRepository(db *gorm.DB) VerificationRepository {
	return &GormVerificationRepository{db: db}
}

// Create creates a new verification code record
func (r *GormVerificationRepository) Create(ctx context.Context, verification *domain.VerificationCode) error {
	if verification == nil {
		return errors.New("verification code is nil")
	}
	// Optional: Add phone validation helper here if available
	// if err := validatePhone(verification.PhoneNumber); err != nil {
	//     return fmt.Errorf("invalid phone number: %w", err)
	// }

	err := r.db.WithContext(ctx).Create(verification).Error
	if err == nil {
		log.Printf("[VerificationRepository] Created verification code for phone: %s, type: %s", verification.PhoneNumber, verification.Type)
	}
	return err
}

// FindByPhoneAndType finds a verification code by phone number and type
func (r *GormVerificationRepository) FindByPhoneAndType(ctx context.Context, phone string, codeType domain.VerificationCodeType) (*domain.VerificationCode, error) {
	var verification domain.VerificationCode
	err := r.db.WithContext(ctx).
		Where("phone_number = ? AND type = ? AND deleted_at IS NULL", phone, codeType).
		First(&verification).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // not found is not an error for business logic
		}
		return nil, err
	}
	return &verification, nil
}

// DeleteByPhone deletes verification codes by phone and type
func (r *GormVerificationRepository) DeleteByPhone(ctx context.Context, phone string, codeType domain.VerificationCodeType) error {
	result := r.db.WithContext(ctx).
		Where("phone_number = ? AND type = ?", phone, codeType).
		Delete(&domain.VerificationCode{})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected > 0 {
		log.Printf("[VerificationRepository] Deleted %d verification code(s) for phone: %s, type: %s", result.RowsAffected, phone, codeType)
	}

	return nil
}

// Update updates a verification code record
func (r *GormVerificationRepository) Update(ctx context.Context, verification *domain.VerificationCode) error {
	if verification == nil {
		return errors.New("verification code is nil")
	}

	err := r.db.WithContext(ctx).Save(verification).Error
	if err == nil {
		log.Printf("[VerificationRepository] Updated verification code for phone: %s, type: %s", verification.PhoneNumber, verification.Type)
	}
	return err
}

// DeleteExpired removes expired verification codes (cleanup job)
func (r *GormVerificationRepository) DeleteExpired(ctx context.Context) error {
	now := time.Now()
	result := r.db.WithContext(ctx).
		Where("expires_at < ?", now).
		Delete(&domain.VerificationCode{})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected > 0 {
		log.Printf("[VerificationRepository] Deleted %d expired verification code(s) at %v", result.RowsAffected, now)
	} else {
		log.Printf("[VerificationRepository] No expired verification codes to delete at %v", now)
	}

	return nil
}