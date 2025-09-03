// File: internal/services/user_service.go
package services

import (
	"context"
	"errors"

	"github.com/iyunix/go-internist/internal/auth"
	"github.com/iyunix/go-internist/internal/domain"
	"github.com/iyunix/go-internist/internal/repository"
)

// UserService now holds the JWT secret key.
type UserService struct {
	userRepo   repository.UserRepository
	jwtSecret  []byte // <-- 1. New field to store the secret
}

// NewUserService now requires the JWT secret key to be provided.
func NewUserService(repo repository.UserRepository, secretKey string) *UserService {
	return &UserService{
		userRepo:  repo,
		jwtSecret: []byte(secretKey), // <-- 2. Store the provided secret
	}
}

// RegisterUser function remains the same.
func (s *UserService) RegisterUser(ctx context.Context, user *domain.User, plainPassword string) (*domain.User, error) {
	_, err := s.userRepo.FindByUsername(ctx, user.Username)
	if err == nil {
		return nil, errors.New("username already exists")
	}

	if err := user.HashPassword(plainPassword); err != nil {
		return nil, err
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

// Login now uses the stored secret to generate the token.
func (s *UserService) Login(ctx context.Context, username, password string) (string, error) {
	user, err := s.userRepo.FindByUsername(ctx, username)
	if err != nil {
		return "", errors.New("invalid credentials")
	}

	if err := user.ValidatePassword(password); err != nil {
		return "", errors.New("invalid credentials")
	}

	// --- 3. Pass the stored secret to the GenerateJWT function ---
	token, err := auth.GenerateJWT(user.ID, s.jwtSecret)
	if err != nil {
		return "", errors.New("could not generate token")
	}

	return token, nil
}