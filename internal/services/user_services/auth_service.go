// G:\go_internist\internal\services\user_services\auth_service.go
package user_services

import (
    "context"
    "errors"
    "fmt"
    "regexp"  // ← Add this import for validation
    "time"

    "github.com/golang-jwt/jwt/v5"
    "golang.org/x/crypto/bcrypt"
    "github.com/iyunix/go-internist/internal/domain"
    "github.com/iyunix/go-internist/internal/repository/user"
)

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

// Login authenticates a user and returns a JWT token
func (s *AuthService) Login(ctx context.Context, username, password string) (*domain.User, string, error) {
    if username == "" || password == "" {
        s.logger.Warn("login attempt with empty credentials", 
            "has_username", username != "",
            "has_password", password != "")
        return nil, "", errors.New("username and password are required")
    }

    s.logger.Info("user login attempt", 
        "username", username[:min(4, len(username))]+"****",
        "username_length", len(username))

    user, err := s.userRepo.FindByUsername(ctx, username)
    if err != nil {
        s.logger.Warn("login failed - user not found", 
            "username", username[:min(4, len(username))]+"****",
            "error", "user_not_found")
        return nil, "", errors.New("invalid credentials")
    }

    // Verify password
    if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
        s.logger.Warn("login failed - invalid password", 
            "username", username[:min(4, len(username))]+"****",
            "user_id", user.ID,
            "error", "invalid_password")
        return nil, "", errors.New("invalid credentials")
    }

    // Check if user is verified
    if !user.IsVerified {
        s.logger.Warn("login attempt by unverified user", 
            "username", username[:min(4, len(username))]+"****",
            "user_id", user.ID)
        return nil, "", errors.New("account not verified")
    }

    // Generate JWT token
    token, err := s.generateJWTToken(user)
    if err != nil {
        s.logger.Error("JWT token generation failed", 
            "error", err,
            "user_id", user.ID,
            "username", username[:min(4, len(username))]+"****")
        return nil, "", fmt.Errorf("failed to generate token: %w", err)
    }

    s.logger.Info("login successful", 
        "username", username[:min(4, len(username))]+"****",
        "user_id", user.ID,
        "is_admin", user.IsAdmin,
        "subscription_plan", user.SubscriptionPlan)

    return user, token, nil
}

// ✅ FIXED: Register now accepts username as first parameter
func (s *AuthService) Register(ctx context.Context, username, phone, password string) (*domain.User, error) {
    // ✅ Validate all inputs
    if err := s.validateRegistrationInput(username, phone, password); err != nil {
        s.logger.Warn("registration validation failed", 
            "username", username[:min(4, len(username))]+"****",
            "phone", phone[:min(4, len(phone))]+"****",
            "error", err.Error())
        return nil, fmt.Errorf("validation failed: %w", err)
    }

    s.logger.Info("user registration attempt", 
        "username", username[:min(4, len(username))]+"****",
        "phone", phone[:min(4, len(phone))]+"****")

    // ✅ Check if user already exists by phone
    existingUser, err := s.userRepo.FindByPhone(ctx, phone)
    if err == nil && existingUser != nil {
        s.logger.Warn("registration failed - phone already exists", 
            "phone", phone[:min(4, len(phone))]+"****",
            "existing_user_id", existingUser.ID)
        return nil, errors.New("user with this phone number already exists")
    }

    // ✅ Check if username already exists
    existingUser, err = s.userRepo.FindByUsername(ctx, username)
    if err == nil && existingUser != nil {
        s.logger.Warn("registration failed - username already exists", 
            "username", username[:min(4, len(username))]+"****",
            "existing_user_id", existingUser.ID)
        return nil, errors.New("username already taken")
    }

    // ✅ Hash password
    hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    if err != nil {
        s.logger.Error("password hashing failed", 
            "error", err,
            "username", username[:min(4, len(username))]+"****",
            "phone", phone[:min(4, len(phone))]+"****")
        return nil, fmt.Errorf("failed to hash password: %w", err)
    }

    // ✅ Create user with ALL required fields including username
    user := &domain.User{
        Username:              username,                              // ✅ Username included from start
        PhoneNumber:           phone,
        Password:              string(hashedPassword),
        IsAdmin:               phone == s.adminPhone,
        SubscriptionPlan:      domain.PlanBasic,
        CharacterBalance:      domain.PlanCredits[domain.PlanBasic],
        TotalCharacterBalance: domain.PlanCredits[domain.PlanBasic],
        Status:               domain.UserStatusPending,               // Assuming you have this field
    }

    // ✅ Create user in database
    createdUser, err := s.userRepo.Create(ctx, user)
    if err != nil {
        s.logger.Error("user creation failed", 
            "error", err,
            "username", username[:min(4, len(username))]+"****",
            "phone", phone[:min(4, len(phone))]+"****")
        return nil, fmt.Errorf("failed to create user: %w", err)
    }

    s.logger.Info("user registered successfully", 
        "username", username[:min(4, len(username))]+"****",
        "phone", phone[:min(4, len(phone))]+"****",
        "user_id", createdUser.ID,
        "is_admin", createdUser.IsAdmin,
        "initial_balance", createdUser.CharacterBalance)

    return createdUser, nil
}

// ✅ NEW: Validation helper method
func (s *AuthService) validateRegistrationInput(username, phone, password string) error {
    // Validate username (same regex as handler)
    usernameRegex := regexp.MustCompile(`^[a-zA-Z0-9_]{3,20}$`)
    if !usernameRegex.MatchString(username) {
        return fmt.Errorf("username validation: username must be 3-20 characters, alphanumeric or underscore")
    }

    // Validate phone
    phoneRegex := regexp.MustCompile(`^\+?[0-9]{7,15}$`)
    if !phoneRegex.MatchString(phone) {
        return fmt.Errorf("phone validation: invalid phone number format")
    }

    // Validate password (updated to match handler requirement)
    if len(password) < 8 {
        return fmt.Errorf("password validation: password must be at least 8 characters")
    }

    return nil
}

// ValidateJWTToken validates a JWT token and returns the user ID
func (s *AuthService) ValidateJWTToken(tokenString string) (uint, error) {
    if tokenString == "" {
        s.logger.Warn("JWT validation attempted with empty token")
        return 0, errors.New("empty token")
    }

    token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
        if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
            s.logger.Warn("JWT token with invalid signing method", 
                "method", token.Header["alg"])
            return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
        }
        return []byte(s.jwtSecretKey), nil
    })

    if err != nil {
        s.logger.Warn("JWT token validation failed", "error", err)
        return 0, fmt.Errorf("invalid token: %w", err)
    }

    if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
        userID, ok := claims["user_id"].(float64)
        if !ok {
            s.logger.Warn("JWT token missing user_id claim")
            return 0, errors.New("invalid token claims")
        }

        s.logger.Debug("JWT token validated successfully", "user_id", uint(userID))
        return uint(userID), nil
    }

    s.logger.Warn("JWT token validation failed - invalid claims")
    return 0, errors.New("invalid token")
}

// generateJWTToken creates a JWT token for the user
func (s *AuthService) generateJWTToken(user *domain.User) (string, error) {
    claims := jwt.MapClaims{
        "user_id":  user.ID,
        "username": user.Username,  // ✅ Include username in JWT
        "phone":    user.PhoneNumber,
        "is_admin": user.IsAdmin,
        "exp":      time.Now().Add(time.Hour * 24 * 7).Unix(), // 7 days
        "iat":      time.Now().Unix(),
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    tokenString, err := token.SignedString([]byte(s.jwtSecretKey))
    if err != nil {
        return "", err
    }

    s.logger.Debug("JWT token generated", 
        "user_id", user.ID,
        "username", user.Username[:min(4, len(user.Username))]+"****",
        "expires_in", "7 days")

    return tokenString, nil
}
