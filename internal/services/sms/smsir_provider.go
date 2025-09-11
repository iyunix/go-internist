// G:\go_internist\internal\services\sms\smsir_provider.go
package sms

import (
    "bytes"
    "context"
    "encoding/json"
    "io"
    "net/http"
)

type SMSIRProvider struct {
    config *Config
    client *http.Client
}

func NewSMSIRProvider(config *Config) *SMSIRProvider {
    return &SMSIRProvider{
        config: config,
        client: &http.Client{
            Timeout: config.Timeout,
        },
    }
}

func (p *SMSIRProvider) SendVerificationCode(ctx context.Context, phone, code string) error {
    payload := map[string]interface{}{
        "mobile":     phone,
        "templateId": p.config.TemplateID,
        "parameters": []map[string]string{
            {"name": "Code", "value": code},
        },
    }

    return p.sendRequest(ctx, payload)
}

func (p *SMSIRProvider) sendRequest(ctx context.Context, payload interface{}) error {
    body, err := json.Marshal(payload)
    if err != nil {
        return &SMSError{Type: ErrTypeValidation, Message: "invalid payload", Cause: err}
    }

    req, err := http.NewRequestWithContext(ctx, "POST", p.config.APIURL, bytes.NewBuffer(body))
    if err != nil {
        return &SMSError{Type: ErrTypeNetwork, Message: "failed to create request", Cause: err}
    }

    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("X-API-KEY", p.config.AccessKey)

    resp, err := p.client.Do(req)
    if err != nil {
        return &SMSError{Type: ErrTypeNetwork, Message: "request failed", Cause: err}
    }
    defer resp.Body.Close()

    return p.handleResponse(resp)
}

func (p *SMSIRProvider) handleResponse(resp *http.Response) error {
    if resp.StatusCode >= 200 && resp.StatusCode < 300 {
        return nil
    }

    responseBody, _ := io.ReadAll(resp.Body)
    
    if resp.StatusCode == 429 {
        return &SMSError{
            Type:    ErrTypeRateLimit,
            Code:    resp.StatusCode,
            Message: "rate limit exceeded",
        }
    }

    return &SMSError{
        Type:    ErrTypeProvider,
        Code:    resp.StatusCode,
        Message: string(responseBody),
    }
}

func (p *SMSIRProvider) HealthCheck(ctx context.Context) error {
    return nil
}
