// G:\go_internist\internal\services\sms\interface.go
package sms

import "context"

// ProviderStatus represents the health status of SMS provider
type ProviderStatus struct {
    IsHealthy bool
    Message   string
}

type Provider interface {
    SendVerificationCode(ctx context.Context, phone, code string) error
    HealthCheck(ctx context.Context) error
}

type Service interface {
    SendCode(ctx context.Context, phone, code string) error
    GetProviderStatus() ProviderStatus
}

