// File: internal/services/user_services/balance_service.go
package user_services

import (
	"context"
	"errors"
	"fmt"

	"github.com/iyunix/go-internist/internal/domain"
	"github.com/iyunix/go-internist/internal/repository"
)

type BalanceService struct {
	userRepo repository.UserRepository
}

func NewBalanceService(userRepo repository.UserRepository) *BalanceService {
	return &BalanceService{
		userRepo: userRepo,
	}
}

// GetCharacterBalance retrieves a user's current character balance.
func (s *BalanceService) GetCharacterBalance(ctx context.Context, userID uint) (int, error) {
	balance, err := s.userRepo.GetCharacterBalance(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return 0, errors.New("user not found")
		}
		return 0, fmt.Errorf("failed to get character balance: %w", err)
	}
	return balance, nil
}

// --- THIS IS THE NEW FUNCTION WE ARE ADDING ---
// GetUserBalanceInfo retrieves both current and total balance for a user.
func (s *BalanceService) GetUserBalanceInfo(ctx context.Context, userID uint) (current int, total int, err error) {
	user, err := s.findUserByID(ctx, userID)
	if err != nil {
		return 0, 0, err
	}
	// Return both the current balance and the new total balance from the user object.
	return user.CharacterBalance, user.TotalCharacterBalance, nil
}


// CanUserAskQuestion checks if user has enough balance for a question.
func (s *BalanceService) CanUserAskQuestion(ctx context.Context, userID uint, questionLength int) (bool, int, error) {
	user, err := s.findUserByID(ctx, userID)
	if err != nil {
		return false, 0, err
	}

	chargeAmount := user.CalculateChargeForQuestion(questionLength)
	canAsk := user.CharacterBalance >= chargeAmount
	
	return canAsk, chargeAmount, nil
}

// DeductCharactersForQuestion deducts characters when user asks a question.
func (s *BalanceService) DeductCharactersForQuestion(ctx context.Context, userID uint, questionLength int) (int, error) {
	user, err := s.findUserByID(ctx, userID)
	if err != nil {
		return 0, err
	}

	chargeAmount := user.CalculateChargeForQuestion(questionLength)
	if err := s.validateBalance(user, chargeAmount); err != nil {
		return 0, err
	}

	if err := user.DeductCharacters(questionLength); err != nil {
		return 0, err
	}

	if err := s.userRepo.UpdateCharacterBalance(ctx, userID, user.CharacterBalance); err != nil {
		return 0, fmt.Errorf("failed to update character balance: %w", err)
	}

	return chargeAmount, nil
}

// CalculateChargePreview calculates how much will be charged without deducting.
func (s *BalanceService) CalculateChargePreview(questionLength int) int {
	if questionLength < domain.MinCharacterCharge {
		return domain.MinCharacterCharge
	}
	return questionLength
}

// AddCharacters adds characters to user's balance (for admin functionality).
func (s *BalanceService) AddCharacters(ctx context.Context, userID uint, amount int) error {
	if amount <= 0 {
		return errors.New("amount must be positive")
	}

	user, err := s.findUserByID(ctx, userID)
	if err != nil {
		return err
	}

	user.AddCharacters(amount)
	
	if err := s.userRepo.UpdateCharacterBalance(ctx, userID, user.CharacterBalance); err != nil {
		return fmt.Errorf("failed to update character balance: %w", err)
	}

	return nil
}

// Private helper methods
func (s *BalanceService) findUserByID(ctx context.Context, userID uint) (*domain.User, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, fmt.Errorf("failed to find user: %w", err)
	}
	return user, nil
}

func (s *BalanceService) validateBalance(user *domain.User, chargeAmount int) error {
	if !user.CanAskQuestion() {
		return errors.New("insufficient character balance")
	}
	
	if user.CharacterBalance < chargeAmount {
		return errors.New("insufficient character balance")
	}

	return nil
}