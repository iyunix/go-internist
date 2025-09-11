// G:\go_internist\internal\services\chat\types.go
package chat

// Logger defines the logging interface used across chat services
type Logger interface {
    Info(msg string, keysAndValues ...interface{})
    Error(msg string, keysAndValues ...interface{})
    Debug(msg string, keysAndValues ...interface{})
    Warn(msg string, keysAndValues ...interface{})
}
