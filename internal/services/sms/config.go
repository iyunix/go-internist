// G:\go_internist\internal\services\sms\config.go
package sms

import (
    "fmt"
    "time"
)

type Config struct {
    AccessKey     string
    TemplateID    int
    APIURL        string
    Timeout       time.Duration
    MaxRetries    int
    RetryDelay    time.Duration
}

func (c *Config) Validate() error {
    if c.AccessKey == "" {
        return fmt.Errorf("SMS_ACCESS_KEY is required")
    }
    if c.APIURL == "" {
        return fmt.Errorf("SMS_API_URL is required") 
    }
    if c.TemplateID == 0 {
        return fmt.Errorf("SMS_TEMPLATE_ID is required")
    }
    return nil
}
