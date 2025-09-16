// File: internal/repository/verification/verification_repository.go
package verification

import (
    "context"
    "errors"
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
    return r.db.WithContext(ctx).Create(verification).Error
}

// FindByPhoneAndType finds a verification code by phone number and type
func (r *GormVerificationRepository) FindByPhoneAndType(ctx context.Context, phone string, codeType domain.VerificationCodeType) (*domain.VerificationCode, error) {
    var verification domain.VerificationCode
    err := r.db.WithContext(ctx).
        Where("phone_number = ? AND type = ? AND deleted_at IS NULL", phone, codeType).
        First(&verification).Error
    
    if err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, nil
        }
        return nil, err
    }
    return &verification, nil
}

// DeleteByPhone deletes verification codes by phone and type
func (r *GormVerificationRepository) DeleteByPhone(ctx context.Context, phone string, codeType domain.VerificationCodeType) error {
    return r.db.WithContext(ctx).
        Where("phone_number = ? AND type = ?", phone, codeType).
        Delete(&domain.VerificationCode{}).Error
}

// Update updates a verification code record
func (r *GormVerificationRepository) Update(ctx context.Context, verification *domain.VerificationCode) error {
    return r.db.WithContext(ctx).Save(verification).Error
}

// DeleteExpired removes expired verification codes (cleanup job)
func (r *GormVerificationRepository) DeleteExpired(ctx context.Context) error {
    return r.db.WithContext(ctx).
        Where("expires_at < ?", "NOW()").
        Delete(&domain.VerificationCode{}).Error
}
