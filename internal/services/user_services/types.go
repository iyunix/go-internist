package user_services

// Logger interface for all user services
type Logger interface {
    Info(msg string, keysAndValues ...interface{})
    Error(msg string, keysAndValues ...interface{})
    Debug(msg string, keysAndValues ...interface{})
    Warn(msg string, keysAndValues ...interface{})
}

// Helper function for safe string slicing
func min(a, b int) int {
    if a < b {
        return a
    }
    return b
}
