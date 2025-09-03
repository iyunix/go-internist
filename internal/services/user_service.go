// File: internal/services/user_service.go

package services

import (
    "context"
    "errors"
    "log"
    "sync"
    "time"

    "golang.org/x/crypto/bcrypt"

    "github.com/iyunix/go-internist/internal/domain"
    "github.com/iyunix/go-internist/internal/repository"
)

// Lockout settings (example)
const (
    maxFailedAttempts = 5
    lockoutDuration   = 15 * time.Minute
)

type UserService struct {
    userRepo   repository.UserRepository
    jwtSecret  string
    mu         sync.Mutex
    failedLogins map[string]failedLoginInfo // username-based tracking example
}

type failedLoginInfo struct {
    count     int
    lastFailed time.Time
    lockedUntil time.Time
}

func NewUserService(userRepo repository.UserRepository, jwtSecret string) *UserService {
    return &UserService{
        userRepo:     userRepo,
        jwtSecret:    jwtSecret,
        failedLogins: make(map[string]failedLoginInfo),
    }
}

// RegisterUser hashes password and creates user.
func (s *UserService) RegisterUser(ctx context.Context, user *domain.User, password string) (*domain.User, error) {
    if err := user.IsValid(); err != nil {
        return nil, err
    }
    if len(password) < 8 {
        return nil, errors.New("password must be at least 8 characters")
    }

    // Hash password with configurable bcrypt cost (DefaultCost is safe default)
    hashedPwd, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    if err != nil {
        log.Printf("Password hashing error: %v", err)
        return nil, errors.New("internal error occurred")
    }
    user.Password = string(hashedPwd)
    return s.userRepo.Create(ctx, user)
}

// Login authenticates user and returns a JWT token if successful.
func (s *UserService) Login(ctx context.Context, username, password string) (string, error) {
    s.mu.Lock()
    defer s.mu.Unlock()

    if info, found := s.failedLogins[username]; found {
        if time.Now().Before(info.lockedUntil) {
            return "", errors.New("account temporarily locked due to multiple failed login attempts")
        }
    }

    user, err := s.userRepo.FindByUsername(ctx, username)
    if err != nil {
        s.recordFailedLogin(username)
        log.Printf("Login error for user %s: %v", username, err)
        return "", errors.New("invalid username or password")
    }

    if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
        s.recordFailedLogin(username)
        return "", errors.New("invalid username or password")
    }

    // Reset failed login record after successful login
    delete(s.failedLogins, username)

    token, err := auth.GenerateJWT(user.ID, []byte(s.jwtSecret))
    if err != nil {
        log.Printf("JWT generation error for user %s: %v", username, err)
        return "", errors.New("authentication error")
    }

    return token, nil
}

func (s *UserService) recordFailedLogin(username string) {
    info := s.failedLogins[username]
    info.count++
    info.lastFailed = time.Now()
    if info.count >= maxFailedAttempts {
        info.lockedUntil = time.Now().Add(lockoutDuration)
        log.Printf("User %s locked out until %v due to failed logins", username, info.lockedUntil)
    }
    s.failedLogins[username] = info
}
