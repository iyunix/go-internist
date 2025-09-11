package services

import (
    "encoding/json"
    "fmt"
    "log"
    "os"
    "strings"
    "time"
)

// Logger defines common logging interface for all services
type Logger interface {
    Info(msg string, keysAndValues ...interface{})
    Error(msg string, keysAndValues ...interface{})
    Debug(msg string, keysAndValues ...interface{})
    Warn(msg string, keysAndValues ...interface{})
}

// LogLevel represents different logging levels
type LogLevel int

const (
    LogLevelDebug LogLevel = iota
    LogLevelInfo
    LogLevelWarn
    LogLevelError
)

func (l LogLevel) String() string {
    switch l {
    case LogLevelDebug:
        return "DEBUG"
    case LogLevelInfo:
        return "INFO"
    case LogLevelWarn:
        return "WARN"
    case LogLevelError:
        return "ERROR"
    default:
        return "UNKNOWN"
    }
}

// ProductionLogger is a structured logger for production use
type ProductionLogger struct {
    logger    *log.Logger
    level     LogLevel
    service   string
    structured bool
}

// NewProductionLogger creates a production-ready logger
func NewProductionLogger(service string) *ProductionLogger {
    return &ProductionLogger{
        logger:    log.New(os.Stdout, "", 0), // We'll handle our own formatting
        level:     LogLevelInfo,              // Default to INFO level
        service:   service,
        structured: true, // Enable structured JSON logging
    }
}

// NewProductionLoggerWithLevel creates logger with specific log level
func NewProductionLoggerWithLevel(service string, level LogLevel) *ProductionLogger {
    return &ProductionLogger{
        logger:    log.New(os.Stdout, "", 0),
        level:     level,
        service:   service,
        structured: true,
    }
}

// SetLevel updates the logging level
func (p *ProductionLogger) SetLevel(level LogLevel) {
    p.level = level
}

// SetStructured enables/disables structured JSON logging
func (p *ProductionLogger) SetStructured(structured bool) {
    p.structured = structured
}

// Info logs informational messages
func (p *ProductionLogger) Info(msg string, keysAndValues ...interface{}) {
    if p.level > LogLevelInfo {
        return
    }
    p.log(LogLevelInfo, msg, keysAndValues...)
}

// Error logs error messages
func (p *ProductionLogger) Error(msg string, keysAndValues ...interface{}) {
    if p.level > LogLevelError {
        return
    }
    p.log(LogLevelError, msg, keysAndValues...)
}

// Debug logs debug messages
func (p *ProductionLogger) Debug(msg string, keysAndValues ...interface{}) {
    if p.level > LogLevelDebug {
        return
    }
    p.log(LogLevelDebug, msg, keysAndValues...)
}

// Warn logs warning messages
func (p *ProductionLogger) Warn(msg string, keysAndValues ...interface{}) {
    if p.level > LogLevelWarn {
        return
    }
    p.log(LogLevelWarn, msg, keysAndValues...)
}

// log is the internal logging method
func (p *ProductionLogger) log(level LogLevel, msg string, keysAndValues ...interface{}) {
    timestamp := time.Now().UTC().Format(time.RFC3339)
    
    if p.structured {
        // Structured JSON logging for production
        logEntry := map[string]interface{}{
            "timestamp": timestamp,
            "level":     level.String(),
            "service":   p.service,
            "message":   msg,
        }
        
        // Add key-value pairs
        if len(keysAndValues) > 0 {
            fields := make(map[string]interface{})
            for i := 0; i < len(keysAndValues)-1; i += 2 {
                if key, ok := keysAndValues[i].(string); ok {
                    fields[key] = keysAndValues[i+1]
                }
            }
            if len(fields) > 0 {
                logEntry["fields"] = fields
            }
        }
        
        jsonBytes, _ := json.Marshal(logEntry)
        p.logger.Println(string(jsonBytes))
    } else {
        // Human-readable logging for development
        var kvStr strings.Builder
        if len(keysAndValues) > 0 {
            kvStr.WriteString(" ")
            for i := 0; i < len(keysAndValues)-1; i += 2 {
                if i > 0 {
                    kvStr.WriteString(" ")
                }
                kvStr.WriteString(fmt.Sprintf("%v=%v", keysAndValues[i], keysAndValues[i+1]))
            }
        }
        
        p.logger.Printf("[%s] %s [%s] %s%s", 
            timestamp, level.String(), p.service, msg, kvStr.String())
    }
}

// NoOpLogger is a logger that does nothing (for testing)
type NoOpLogger struct{}

func (n *NoOpLogger) Info(msg string, keysAndValues ...interface{})  {}
func (n *NoOpLogger) Error(msg string, keysAndValues ...interface{}) {}
func (n *NoOpLogger) Debug(msg string, keysAndValues ...interface{}) {}
func (n *NoOpLogger) Warn(msg string, keysAndValues ...interface{})  {}

// Environment-based logger factory
func NewLogger(service string) Logger {
    env := os.Getenv("GO_ENV")
    logLevel := os.Getenv("LOG_LEVEL")
    
    if env == "test" {
        return &NoOpLogger{}
    }
    
    logger := NewProductionLogger(service)
    
    // Set log level from environment
    switch strings.ToUpper(logLevel) {
    case "DEBUG":
        logger.SetLevel(LogLevelDebug)
    case "INFO", "":
        logger.SetLevel(LogLevelInfo)
    case "WARN":
        logger.SetLevel(LogLevelWarn)
    case "ERROR":
        logger.SetLevel(LogLevelError)
    }
    
    // Use structured logging in production
    if env == "production" {
        logger.SetStructured(true)
    } else {
        logger.SetStructured(false) // Human-readable for development
    }
    
    return logger
}
