<img src="https://r2cdn.perplexity.ai/pplx-full-logo-primary-dark%402x.png" style="height:64px;margin-right:32px"/>

# User Services Package

## Overview

The `user_services` package provides comprehensive user management functionality for the Go Internist medical AI application. It handles all aspects of user lifecycle management including authentication, authorization, account security, balance management, and SMS verification.

## Architecture

The package is organized into specialized service layers:

```
user_services/
├── auth_service.go          # Authentication & JWT management
├── balance_service.go       # Credit/balance operations
├── lockout_service.go       # Account security & brute force protection
├── user_service.go          # Core user management
├── verification_service.go  # SMS verification workflows
└── types.go                # Shared interfaces & utilities
```


## Services Overview

### AuthService

Handles user authentication, registration, and JWT token management.

**Key Responsibilities:**

- User login with username/password
- User registration with phone verification
- JWT token generation and validation
- Password hashing and verification


### UserService

Core user management operations and data access.

**Key Responsibilities:**

- User CRUD operations
- User lookup by ID, phone, username
- Admin privilege management
- User statistics for admin dashboard


### BalanceService

Manages user credits and subscription balances.

**Key Responsibilities:**

- Credit deduction for AI queries
- Balance checking and validation
- Credit addition and refresh
- Usage tracking and analytics


### LockoutService

Account security and brute force attack protection.

**Key Responsibilities:**

- Failed login attempt tracking
- Automatic account lockout after excessive failures
- Lockout status checking
- Security event logging


### VerificationService

SMS-based account verification workflows.

**Key Responsibilities:**

- SMS verification code generation
- Code validation and expiry management
- Resend functionality with rate limiting
- Phone number verification


## Features

- **🔐 Secure Authentication**: JWT-based auth with bcrypt password hashing
- **📱 SMS Verification**: Phone number verification with SMS.ir integration
- **💳 Credit Management**: Flexible balance system with multiple subscription tiers
- **🛡️ Account Security**: Brute force protection with automatic lockouts
- **👥 User Management**: Complete CRUD operations with admin controls
- **📊 Analytics**: User statistics and usage tracking
- **🔄 Balance Refresh**: Automatic credit renewal based on subscription plans
- **⚡ Rate Limiting**: Built-in protection against abuse


## Configuration

### Required Dependencies

```go
import (
    "github.com/iyunix/go-internist/internal/domain"
    "github.com/iyunix/go-internist/internal/repository"
    "github.com/golang-jwt/jwt/v5"
    "golang.org/x/crypto/bcrypt"
)
```


### Service Initialization

```go
// Initialize all user services
func InitializeUserServices(
    userRepo repository.UserRepository,
    jwtSecret, adminPhone string,
    logger Logger,
) (*UserService, *AuthService, *BalanceService, *LockoutService, *VerificationService) {
    
    userService := NewUserService(userRepo, jwtSecret, adminPhone, logger)
    authService := NewAuthService(userRepo, jwtSecret, adminPhone, logger)
    balanceService := NewBalanceService(userRepo, logger)
    lockoutService := NewLockoutService(userRepo, logger)
    verificationService := NewVerificationService(userRepo, smsService, logger)
    
    return userService, authService, balanceService, lockoutService, verificationService
}
```


## Usage Examples

### User Authentication

```go
// User login
user, token, err := authService.Login(ctx, "john_doe", "password123")
if err != nil {
    // Handle login error
}

// Validate JWT token
userID, err := authService.ValidateJWTToken(tokenString)
if err != nil {
    // Handle invalid token
}
```


### User Registration \& Verification

```go
// Register new user
user, err := authService.Register(ctx, "+1234567890", "securepassword")
if err != nil {
    // Handle registration error
}

// Send SMS verification
err = verificationService.SendVerificationCode(ctx, user.ID)
if err != nil {
    // Handle SMS error
}

// Verify SMS code
err = verificationService.VerifyCode(ctx, user.ID, "123456")
if err != nil {
    // Handle verification error
}
```


### Balance Management

```go
// Check user balance
canAsk, chargeAmount, err := userService.CanUserAskQuestion(ctx, userID, questionLength)
if err != nil {
    // Handle error
}

if !canAsk {
    // Insufficient balance
}

// Deduct credits
actualCharge, err := userService.DeductCharactersForQuestion(ctx, userID, questionLength)
if err != nil {
    // Handle deduction error
}

// Add credits (admin operation)
err = balanceService.AddCredits(ctx, userID, 1000, "admin_credit")
if err != nil {
    // Handle error
}
```


### Account Security

```go
// Check if account is locked
isLocked, status, err := lockoutService.IsAccountLocked(ctx, phoneNumber)
if err != nil {
    // Handle error
}

if isLocked {
    // Account is locked, show lockout info
    fmt.Printf("Account locked until: %v", status.LockedUntil)
}

// Record failed login attempt
err = lockoutService.RecordFailedAttempt(ctx, phoneNumber, clientIP)
if err != nil {
    // Handle error
}
```


### Admin Operations

```go
// Get user statistics
stats, err := userService.GetUserStats(ctx, adminUserID)
if err != nil {
    // Handle error
}

fmt.Printf("Total users: %d, Verified: %d", stats.TotalUsers, stats.VerifiedUsers)

// Delete user (admin only)
err = userService.DeleteUser(ctx, targetUserID, adminUserID)
if err != nil {
    // Handle error
}
```


## Security Features

### Account Lockout Protection

- **Maximum attempts**: 5 failed login attempts
- **Lockout duration**: 15 minutes
- **Automatic unlock**: After lockout period expires
- **IP tracking**: Failed attempts are logged with source IP


### Password Security

- **Hashing**: bcrypt with default cost
- **Minimum length**: 6 characters (configurable)
- **Validation**: Server-side password strength checks


### JWT Security

- **Algorithm**: HMAC-SHA256
- **Expiry**: 7 days (configurable)
- **Claims**: User ID, phone, admin status
- **Validation**: Signature and expiry verification


## Error Handling

### Custom Errors

```go
// Insufficient balance error
type InsufficientBalanceError struct {
    UserID          uint
    CurrentBalance  int
    RequestedAmount int
    Operation       string
}

// Lockout status information
type LockoutStatus struct {
    UserID         uint
    IsLocked       bool
    FailedAttempts int
    LockedUntil    time.Time
    TimeRemaining  time.Duration
}
```


### Common Error Patterns

- **Validation errors**: Invalid input parameters
- **Not found errors**: User/resource not found
- **Authorization errors**: Insufficient permissions
- **Business logic errors**: Insufficient balance, account locked
- **Repository errors**: Database operation failures


## Logging

All services implement comprehensive logging with structured fields:

```go
type Logger interface {
    Info(msg string, keysAndValues ...interface{})
    Error(msg string, keysAndValues ...interface{})
    Debug(msg string, keysAndValues ...interface{})
    Warn(msg string, keysAndValues ...interface{})
}
```


### Log Levels

- **INFO**: Successful operations, state changes
- **WARN**: Failed attempts, validation errors
- **ERROR**: System errors, database failures
- **DEBUG**: Detailed operation tracing


## Subscription Plans

The system supports multiple subscription tiers:

```go
const (
    PlanBasic    = "basic"     // 10,000 credits
    PlanPremium  = "premium"   // 50,000 credits
    PlanPro      = "pro"       // 100,000 credits
)
```


## Best Practices

### Service Usage

1. **Always check errors**: Every service method returns errors that should be handled
2. **Use context**: All methods accept context for timeouts and cancellation
3. **Validate inputs**: Check user inputs before calling service methods
4. **Log operations**: Use structured logging for audit trails

### Security

1. **Rate limiting**: Implement additional rate limiting at the handler level
2. **Input validation**: Sanitize all user inputs
3. **Admin verification**: Always verify admin privileges for sensitive operations
4. **Token expiry**: Regularly refresh JWT tokens

### Performance

1. **Connection pooling**: Use database connection pooling
2. **Caching**: Consider caching user data for frequent lookups
3. **Batch operations**: Use batch operations for bulk user management

## Dependencies

### Internal Dependencies

- `internal/domain`: User and domain models
- `internal/repository`: Data access interfaces


### External Dependencies

- `github.com/golang-jwt/jwt/v5`: JWT token handling
- `golang.org/x/crypto/bcrypt`: Password hashing
- `gorm.io/gorm`: Database ORM (via repository)


## Testing

### Unit Testing

Each service includes comprehensive unit tests covering:

- Happy path scenarios
- Error conditions
- Edge cases
- Security validations


### Integration Testing

- Database integration tests
- SMS service integration
- End-to-end authentication flows


## Monitoring

### Metrics to Monitor

- Login success/failure rates
- Account lockout frequency
- Balance depletion rates
- SMS verification success rates
- JWT token validation errors


### Health Checks

- Database connectivity
- SMS service availability
- JWT signing key validation
<span style="display:none">[^1][^2][^3][^4][^5]</span>

<div style="text-align: center">⁂</div>

[^1]: user_service.go

[^2]: types.go

[^3]: balance_service.go

[^4]: lockout_service.go

[^5]: auth_service.go

