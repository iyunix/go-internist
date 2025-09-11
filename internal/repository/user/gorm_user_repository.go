package user

import (
    "context"
    "errors"
    "log"
    "time"

    "github.com/iyunix/go-internist/internal/domain"
    "gorm.io/gorm"
)

var ErrUserNotFound = errors.New("user not found")

type gormUserRepository struct {
    db *gorm.DB
}

func NewGormUserRepository(db *gorm.DB) UserRepository {
    return &gormUserRepository{db: db}
}

func (r *gormUserRepository) Create(ctx context.Context, user *domain.User) (*domain.User, error) {
    if err := r.db.WithContext(ctx).Create(user).Error; err != nil {
        log.Printf("[UserRepository] Create error for username %s: %v", user.Username, err)
        return nil, errors.New("database error creating user")
    }
    return user, nil
}

func (r *gormUserRepository) Update(ctx context.Context, user *domain.User) error {
    if err := r.db.WithContext(ctx).Save(user).Error; err != nil {
        log.Printf("[UserRepository] Update error for user ID %d: %v", user.ID, err)
        return errors.New("database error updating user")
    }
    return nil
}

func (r *gormUserRepository) FindByUsername(ctx context.Context, username string) (*domain.User, error) {
    var user domain.User
    err := r.db.WithContext(ctx).Where("username = ?", username).First(&user).Error
    return r.handleFindError(err, &user, "FindByUsername", username)
}

func (r *gormUserRepository) FindByID(ctx context.Context, id uint) (*domain.User, error) {
    var user domain.User
    err := r.db.WithContext(ctx).First(&user, id).Error
    return r.handleFindError(err, &user, "FindByID", id)
}

func (r *gormUserRepository) FindByPhone(ctx context.Context, phone string) (*domain.User, error) {
    var user domain.User
    err := r.db.WithContext(ctx).Where("phone_number = ?", phone).First(&user).Error
    return r.handleFindError(err, &user, "FindByPhone", phone)
}

func (r *gormUserRepository) FindByUsernameOrPhone(ctx context.Context, username, phone string) (*domain.User, error) {
    var user domain.User
    err := r.db.WithContext(ctx).Where("username = ? OR phone_number = ?", username, phone).First(&user).Error
    return r.handleFindError(err, &user, "FindByUsernameOrPhone", username)
}

func (r *gormUserRepository) FindByPhoneAndStatus(ctx context.Context, phone string, status domain.UserStatus) (*domain.User, error) {
    var user domain.User
    err := r.db.WithContext(ctx).Where("phone_number = ? AND status = ?", phone, status).First(&user).Error
    return r.handleFindError(err, &user, "FindByPhoneAndStatus", phone)
}

func (r *gormUserRepository) ResetFailedAttempts(ctx context.Context, id uint) error {
    err := r.db.WithContext(ctx).Model(&domain.User{}).Where("id = ?", id).Updates(map[string]interface{}{
        "failed_login_attempts": 0,
        "locked_until":          time.Time{},
    }).Error
    if err != nil {
        log.Printf("[UserRepository] ResetFailedAttempts error for user ID %d: %v", id, err)
        return errors.New("database error resetting failed attempts")
    }
    return nil
}

func (r *gormUserRepository) GetCharacterBalance(ctx context.Context, userID uint) (int, error) {
    var balance int
    err := r.db.WithContext(ctx).Model(&domain.User{}).Where("id = ?", userID).Select("character_balance").Scan(&balance).Error
    if err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return 0, ErrUserNotFound
        }
        log.Printf("[UserRepository] GetCharacterBalance error for user ID %d: %v", userID, err)
        return 0, errors.New("database error getting character balance")
    }
    return balance, nil
}

func (r *gormUserRepository) UpdateCharacterBalance(ctx context.Context, userID uint, newBalance int) error {
    result := r.db.WithContext(ctx).Model(&domain.User{}).Where("id = ?", userID).Update("character_balance", newBalance)
    if result.Error != nil {
        log.Printf("[UserRepository] UpdateCharacterBalance error for user ID %d: %v", userID, result.Error)
        return errors.New("database error updating character balance")
    }
    if result.RowsAffected == 0 {
        return ErrUserNotFound
    }
    return nil
}

func (r *gormUserRepository) FindAll(ctx context.Context) ([]domain.User, error) {
    var users []domain.User
    if err := r.db.WithContext(ctx).Order("id asc").Find(&users).Error; err != nil {
        log.Printf("[UserRepository] FindAll error: %v", err)
        return nil, errors.New("database error retrieving all users")
    }
    return users, nil
}

func (r *gormUserRepository) Delete(ctx context.Context, userID uint) error {
    result := r.db.WithContext(ctx).Delete(&domain.User{}, userID)
    if result.Error != nil {
        log.Printf("[UserRepository] Delete error for user ID %d: %v", userID, result.Error)
        return errors.New("database error deleting user")
    }
    if result.RowsAffected == 0 {
        return ErrUserNotFound
    }
    return nil
}

func (r *gormUserRepository) handleFindError(err error, user *domain.User, methodName string, identifier interface{}) (*domain.User, error) {
    if err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, ErrUserNotFound
        }
        log.Printf("[UserRepository] %s error for %v: %v", methodName, identifier, err)
        return nil, errors.New("database error finding user")
    }
    return user, nil
}
