// File: internal/repository/gorm_user_repository.go
package repository

import (
    "context"
    "errors"
    "log"

    "github.com/iyunix/go-internist/internal/domain"
    "gorm.io/gorm"
)

type gormUserRepository struct {
    db *gorm.DB
}

func NewGormUserRepository(db *gorm.DB) UserRepository {
    return &gormUserRepository{db: db}
}

// Create inserts a new user and logs any DB errors
func (r *gormUserRepository) Create(ctx context.Context, user *domain.User) (*domain.User, error) {
    if err := r.db.WithContext(ctx).Create(user).Error; err != nil {
        log.Printf("[UserRepository] Create error for username %s: %v", user.Username, err)
        return nil, errors.New("database error creating user")
    }
    // Optionally add audit logic here for creating users
    return user, nil
}

// FindByUsername safely searches user by username and logs errors
func (r *gormUserRepository) FindByUsername(ctx context.Context, username string) (*domain.User, error) {
    var user domain.User
    err := r.db.WithContext(ctx).Where("username = ?", username).First(&user).Error
    if err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, errors.New("user not found")
        }
        log.Printf("[UserRepository] FindByUsername error for %s: %v", username, err)
        return nil, errors.New("database error finding user")
    }
    return &user, nil
}

// FindByID fetches user by ID and logs errors
func (r *gormUserRepository) FindByID(ctx context.Context, id uint) (*domain.User, error) {
    var user domain.User
    err := r.db.WithContext(ctx).First(&user, id).Error
    if err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, errors.New("user not found")
        }
        log.Printf("[UserRepository] FindByID error for %d: %v", id, err)
        return nil, errors.New("database error finding user")
    }
    return &user, nil
}
