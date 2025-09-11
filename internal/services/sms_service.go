// G:\go_internist\internal\services\sms_service.go
package services

import (
    "context"
    "github.com/iyunix/go-internist/internal/services/sms"
)

type SMSService struct {
    provider sms.Provider
    logger   Logger // Inject logger interface
}

func NewSMSService(provider sms.Provider, logger Logger) *SMSService {
    return &SMSService{
        provider: provider,
        logger:   logger,
    }
}

func (s *SMSService) SendVerificationCode(ctx context.Context, phone, code string) error {
    s.logger.Info("sending SMS verification", 
        "phone", phone[:4]+"****", // Mask phone for privacy
        "code_length", len(code))
    
    err := s.provider.SendVerificationCode(ctx, phone, code)
    if err != nil {
        s.logger.Error("SMS send failed", "error", err)
        return err
    }
    
    s.logger.Info("SMS sent successfully")
    return nil
}
