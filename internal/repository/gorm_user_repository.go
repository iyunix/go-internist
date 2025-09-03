// File: internal/repository/gorm_user_repository.go
package repository

import (
	"context"
	"errors"

	// Don't forget to use your own module path!
	"github.com/iyunix/go-internist/internal/domain"
	"gorm.io/gorm"
)

// gormUserRepository is the real librarian that uses GORM.
type gormUserRepository struct {
	db *gorm.DB // It holds a connection to the database.
}

// NewGormUserRepository is a factory function to create a new librarian.
func NewGormUserRepository(db *gorm.DB) UserRepository {
	return &gormUserRepository{db: db}
}

// Create implements the Create method from our interface.
func (r *gormUserRepository) Create(ctx context.Context, user *domain.User) error {
	// We ask GORM to create a new record in the database based on our user struct.
	return r.db.WithContext(ctx).Create(user).Error
}

// FindByID implements the FindByID method from our interface.
func (r *gormUserRepository) FindByID(ctx context.Context, id uint) (*domain.User, error) {
	var user domain.User
	// We ask GORM to find the first record with this ID and put the data into our user variable.
	err := r.db.WithContext(ctx).First(&user, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return &user, nil
}

// FindByUsername implements the FindByUsername method from our interface.
func (r *gormUserRepository) FindByUsername(ctx context.Context, username string) (*domain.User, error) {
	var user domain.User
	// We ask GORM to find a user where the 'username' column matches the one we provide.
	err := r.db.WithContext(ctx).Where("username = ?", username).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return &user, nil
}