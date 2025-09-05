// File: internal/services/sms_service.go
package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv" // NEW: To convert template ID to number
	"time"
)

// SMSService handles sending SMS messages via the Sms.ir provider.
type SMSService struct {
	AccessKey  string
	TemplateID int // CHANGED: TemplateID is a number for Sms.ir
	APIURL     string
	Client     *http.Client
}

// NewSMSService creates and configures a new SMSService for Sms.ir.
func NewSMSService() *SMSService {
	templateID, _ := strconv.Atoi(os.Getenv("SMS_TEMPLATE_ID"))

	return &SMSService{
		AccessKey:  os.Getenv("SMS_ACCESS_KEY"),
		TemplateID: templateID,
		APIURL:     os.Getenv("SMS_API_URL"),
		Client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// SendVerificationCode sends a verification code using the Sms.ir "verify" endpoint.
func (s *SMSService) SendVerificationCode(ctx context.Context, phone, code string) error {
	if s.APIURL == "" || s.AccessKey == "" || s.TemplateID == 0 {
		return fmt.Errorf("SMS service is not configured. Check environment variables SMS_API_URL, SMS_ACCESS_KEY, and SMS_TEMPLATE_ID")
	}

	// CHANGED: This is the specific payload structure required by Sms.ir
	payload := map[string]interface{}{
		"mobile":     phone,
		"templateId": s.TemplateID,
		"parameters": []map[string]string{
			{
				"name":  "Code",
				"value": code,
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal sms payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", s.APIURL, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create sms request: %w", err)
	}
	
	// CHANGED: Set headers required by Sms.ir
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-KEY", s.AccessKey) // Sms.ir uses a header for the key

	resp, err := s.Client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request to sms provider: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		responseBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("sms provider returned non-success status code %d: %s", resp.StatusCode, string(responseBody))
	}

	return nil
}