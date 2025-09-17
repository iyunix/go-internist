// G:\go_internist\internal\services\sms_service.go
package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/iyunix/go-internist/internal/services/sms"
)

// SMSService handles SMS delivery with rate limiting and retry logic
type SMSService struct {
	provider   sms.Provider
	logger     Logger
	limiter    map[string]*rateEntry // phone -> attempts/hour
	mu         sync.Mutex
	maxPerHour int
}

// rateEntry tracks attempts and reset time per phone number
type rateEntry struct {
	Attempts  int
	LastReset time.Time
}

// NewSMSService creates a new SMS service with rate limiting (default: 6/hour per phone)
func NewSMSService(provider sms.Provider, logger Logger) *SMSService {
	return &SMSService{
		provider:   provider,
		logger:     logger,
		limiter:    make(map[string]*rateEntry),
		maxPerHour: 6, // 6 SMS per phone per hour — adjust as needed
	}
}

// allowSend checks if sending an SMS to the given phone is within rate limits
func (s *SMSService) allowSend(phone string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, exists := s.limiter[phone]
	now := time.Now()

	if !exists || now.Sub(entry.LastReset) > time.Hour {
		// Reset counter for new hour or new phone
		s.limiter[phone] = &rateEntry{
			Attempts:  1,
			LastReset: now,
		}
		return true
	}

	if entry.Attempts < s.maxPerHour {
		entry.Attempts++
		return true
	}

	return false
}

// SendVerificationCode sends a verification code via SMS with retry logic and rate limiting
func (s *SMSService) SendVerificationCode(ctx context.Context, phone, code string) error {
	if !s.allowSend(phone) {
		s.logger.Warn("SMS rate limit reached", "phone", maskPhone(phone))
		return fmt.Errorf("too many verification codes sent. Please wait before trying again")
	}

	s.logger.Info("sending SMS verification",
		"phone", maskPhone(phone),
		"code_length", len(code))

	// Apply retry with exponential backoff
	retryConfig := sms.DefaultRetryConfig()
	err := sms.RetryWithBackoff(ctx, retryConfig, func(retryCtx context.Context) error {
		return s.provider.SendVerificationCode(retryCtx, phone, code)
	})

	if err != nil {
		s.logger.Error("SMS send failed after retries", "error", err, "phone", maskPhone(phone))
		return err
	}

	s.logger.Info("SMS sent successfully", "phone", maskPhone(phone))
	return nil
}

// GetProviderStatus checks the health of the underlying SMS provider
func (s *SMSService) GetProviderStatus(ctx context.Context) sms.ProviderStatus {
	err := s.provider.HealthCheck(ctx)
	if err != nil {
		return sms.ProviderStatus{
			IsHealthy: false,
			Message:   fmt.Sprintf("SMS provider unhealthy: %v", err),
		}
	}
	return sms.ProviderStatus{
		IsHealthy: true,
		Message:   "SMS provider healthy",
	}
}

// maskPhone hides part of the phone number for logging (e.g., +1234****5678)
func maskPhone(phone string) string {
	if len(phone) <= 8 {
		return "****" // fallback for very short numbers
	}
	return phone[:4] + "****" + phone[len(phone)-4:]
}