// File: internal/services/user_services/auth_service.go
package user_services

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/iyunix/go-internist/internal/domain"
	"github.com/iyunix/go-internist/internal/repository/user"
)

// NOTE: The duplicate 'Logger' interface has been removed from this file.
// It should be defined in another file in this package, like types.go.

type AuthService struct {
	userRepo     user.UserRepository
	jwtSecretKey string
	adminPhone   string
	logger       Logger
}

func NewAuthService(userRepo user.UserRepository, jwtSecretKey, adminPhone string, logger Logger) *AuthService {
	return &AuthService{
		userRepo:     userRepo,
		jwtSecretKey: jwtSecretKey,
		adminPhone:   adminPhone,
		logger:       logger,
	}
}

// Login authenticates a user by username OR phone number and returns a JWT token.
func (s *AuthService) Login(ctx context.Context, identifier, password string) (*domain.User, string, error) {
	if identifier == "" || password == "" {
		s.logger.Warn("login attempt with empty credentials")
		return nil, "", errors.New("username or phone number, and password are required")
	}

	s.logger.Info("user login attempt", "identifier", identifier)

	phoneRegex := regexp.MustCompile(`^09\d{9}$`)
	var userEntity *domain.User
	var err error

	if phoneRegex.MatchString(identifier) {
		s.logger.Debug("identifier detected as phone number", "phone", identifier)
		userEntity, err = s.userRepo.FindByPhone(ctx, identifier)
	} else {
		s.logger.Debug("identifier detected as username", "username", identifier)
		userEntity, err = s.userRepo.FindByUsername(ctx, identifier)
	}

	if err != nil || userEntity == nil {
		s.logger.Warn("login failed - user not found with identifier", "identifier", identifier)
		return nil, "", errors.New("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(userEntity.Password), []byte(password)); err != nil {
		s.logger.Warn("login failed - invalid password", "identifier", identifier, "user_id", userEntity.ID)
		return nil, "", errors.New("invalid credentials")
	}

	if !userEntity.IsVerified {
		s.logger.Warn("login attempt by unverified user", "identifier", identifier, "user_id", userEntity.ID)
		return nil, "", errors.New("account not verified")
	}

	token, err := s.generateJWTToken(userEntity)
	if err != nil {
		s.logger.Error("JWT token generation failed", "error", err, "user_id", userEntity.ID)
		return nil, "", fmt.Errorf("failed to generate token: %w", err)
	}

	s.logger.Info("login successful", "identifier", identifier, "user_id", userEntity.ID)
	return userEntity, token, nil
}


func (s *AuthService) Register(ctx context.Context, username, phone, hashedPassword string) (*domain.User, error) {
	if err := s.validateRegistrationInput(username, phone, hashedPassword); err != nil {
		s.logger.Warn("registration validation failed", "error", err.Error())
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	s.logger.Info("user registration attempt", "username", username, "phone", phone)

	existingUser, _ := s.userRepo.FindByPhone(ctx, phone)
	if existingUser != nil {
		s.logger.Warn("registration failed - phone already exists", "phone", phone)
		return nil, errors.New("user with this phone number already exists")
	}
	existingUser, _ = s.userRepo.FindByUsername(ctx, username)
	if existingUser != nil {
		s.logger.Warn("registration failed - username already exists", "username", username)
		return nil, errors.New("username already taken")
	}

	// --- CORRECTED LOGIC ---
	// 1. Use the domain constructor. Note that it expects a hashed password.
	userEntity := domain.NewUser(username, phone, hashedPassword) // Pass the hash directly

	// 2. Set fields specific to the registration context from your handler.
	now := time.Now()
	userEntity.IsAdmin = (phone == s.adminPhone)
	userEntity.Status = domain.UserStatusActive // Set to active as per your logic
	userEntity.IsVerified = true                // Set to verified
	userEntity.VerifiedAt = &now                 // Mark verification time
	// --- END CORRECTION ---

	createdUser, err := s.userRepo.Create(ctx, userEntity)
	if err != nil {
		s.logger.Error("user creation failed", "error", err)
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	s.logger.Info("user registered successfully", "user_id", createdUser.ID)
	return createdUser, nil
}


// Helper for validation (adjust password check since it's now a hash)
func (s *AuthService) validateRegistrationInput(username, phone, passwordHash string) error {
	usernameRegex := regexp.MustCompile(`^[a-zA-Z0-9_]{3,20}$`)
	if !usernameRegex.MatchString(username) {
		return errors.New("username must be 3-20 characters, alphanumeric or underscore")
	}
	phoneRegex := regexp.MustCompile(`^09\d{9}$`)
	if !phoneRegex.MatchString(phone) {
		return errors.New("invalid phone number format")
	}
    // We can't check the length of the original password, but we can check the hash isn't empty
	if passwordHash == "" {
		return errors.New("password hash cannot be empty")
	}
	return nil
}


// Validate JWT token
func (s *AuthService) ValidateJWTToken(tokenString string) (uint, error) {
	if tokenString == "" {
		return 0, errors.New("empty token")
	}
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.jwtSecretKey), nil
	})
	if err != nil {
		return 0, fmt.Errorf("invalid token: %w", err)
	}
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		userID, ok := claims["user_id"].(float64)
		if !ok {
			return 0, errors.New("invalid token claims: missing user_id")
		}
		return uint(userID), nil
	}
	return 0, errors.New("invalid token")
}

// Generate JWT token
func (s *AuthService) generateJWTToken(user *domain.User) (string, error) {
	claims := jwt.MapClaims{
		"user_id":  user.ID,
		"username": user.Username,
		"is_admin": user.IsAdmin,
		"exp":      time.Now().Add(time.Hour * 24 * 7).Unix(), // 7 days
		"iat":      time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecretKey))
}

// NOTE: The duplicate 'min' function has been removed from this file.
// It should be defined in another file in this package, like types.go.
