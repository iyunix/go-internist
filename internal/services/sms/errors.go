// G:\go_internist\internal\services\sms\errors.go
package sms

import "fmt"

type ErrorType string

const (
    ErrTypeConfig      ErrorType = "CONFIG"
    ErrTypeNetwork     ErrorType = "NETWORK" 
    ErrTypeProvider    ErrorType = "PROVIDER"
    ErrTypeRateLimit   ErrorType = "RATE_LIMIT"
    ErrTypeValidation  ErrorType = "VALIDATION"
)

type SMSError struct {
    Type    ErrorType
    Code    int
    Message string
    Cause   error
}

func (e *SMSError) Error() string {
    if e.Cause != nil {
        return fmt.Sprintf("SMS %s error: %s (caused by: %v)", e.Type, e.Message, e.Cause)
    }
    return fmt.Sprintf("SMS %s error: %s", e.Type, e.Message)
}
