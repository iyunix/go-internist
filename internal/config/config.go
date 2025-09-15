// File: internal/config/config.go
package config

import (
    "fmt"
    "log"
    "os"
    "strconv"
    "strings"
    "time"

    "github.com/joho/godotenv"
)

type Config struct {
    // Server Configuration
    ServerPort    string
    Environment   string
    LogLevel      string
    
    // Security
    JWTSecretKey  string
    AllowedOrigins []string
    StaticDir     string
    
    // Database Configuration
    DBHost     string
    DBUser     string
    DBPassword string
    DBName     string
    DBPort     string
    DBSSLMode  string
    
    // AI Services
    AvalaiAPIKeyEmbedding   string
    AvalaiAPIKeyTranslation string
    JabirAPIKey             string
    EmbeddingModelName      string
    
    // Vector Database
    PineconeAPIKey    string
    PineconeIndexHost string
    PineconeNamespace string
    RetrievalTopK     int
    
    // SMS Service
    SMSAccessKey   string
    SMSTemplateID  int
    SMSAPIURL      string
    SMSLineNumber  string
    
    // Application Settings
    AdminPhoneNumber   string
    TranslationEnabled bool
    
    // Server Timeouts
    ReadTimeout  time.Duration
    WriteTimeout time.Duration
    IdleTimeout  time.Duration
}

// Load reads configuration from environment variables or .env file.
func Load() *Config {
    // Use GO_ENV instead of ENV for consistency
    env := os.Getenv("GO_ENV")
    if env == "" {
        env = os.Getenv("ENV") // Fallback for backward compatibility
    }
    if env == "" {
        env = "development" // Default to development
    }
    
    // Load .env file only in non-production environments
    if strings.ToLower(env) != "production" {
        if err := godotenv.Load(); err != nil {
            log.Println("No .env file found; continuing with environment variables")
        }
    }

    cfg := &Config{
        // Server Configuration
        ServerPort:  getEnv("SERVER_PORT", getEnv("PORT", "8080")),
        Environment: env,
        LogLevel:    getEnv("LOG_LEVEL", "INFO"),
        
        // Security
        JWTSecretKey:   getEnv("JWT_SECRET_KEY", ""),
        AllowedOrigins: getEnvAsSlice("ALLOWED_ORIGINS", []string{"http://localhost:8080", "http://localhost:8081"}),
        StaticDir:      getEnv("STATIC_DIR", "web/static"),
        
        // Database Configuration
        DBHost:     getEnv("DB_HOST", "localhost"),
        DBUser:     getEnv("DB_USER", "internist"),
        DBPassword: getEnv("DB_PASSWORD", "medical_ai_2025"),
        DBName:     getEnv("DB_NAME", "go_internist"),
        DBPort:     getEnv("DB_PORT", "5432"),
        DBSSLMode:  getEnv("DB_SSL_MODE", "disable"),
        
        // AI Services  
        AvalaiAPIKeyEmbedding:   getEnv("AVALAI_API_KEY_EMBEDDING", ""),
        AvalaiAPIKeyTranslation: getEnv("AVALAI_API_KEY_TRANSLATION", ""),
        JabirAPIKey:             getEnv("JABIR_API_KEY", ""),
        EmbeddingModelName:      getEnv("EMBEDDING_MODEL_NAME", "text-embedding-3-large"), // Updated default
        
        // Vector Database
        PineconeAPIKey:    getEnv("PINECONE_API_KEY", ""),
        PineconeIndexHost: getEnv("PINECONE_INDEX_HOST", ""),
        PineconeNamespace: getEnv("PINECONE_NAMESPACE", "UpToDate"),
        RetrievalTopK:     getEnvAsInt("RAG_TOPK", 5), // Matches your .env
        
        // SMS Service
        SMSAccessKey:  getEnv("SMS_ACCESS_KEY", ""),
        SMSTemplateID: getEnvAsInt("SMS_TEMPLATE_ID", 0),
        SMSAPIURL:     getEnv("SMS_API_URL", ""),
        SMSLineNumber: getEnv("SMS_LINE_NUMBER", ""),
        
        // Application Settings
        AdminPhoneNumber:   getEnv("ADMIN_PHONE_NUMBER", ""),
        TranslationEnabled: getEnvAsBool("TRANSLATION_ENABLED", true),
        
        // Server Timeouts
        ReadTimeout:  getEnvAsDuration("READ_TIMEOUT", 60*time.Second),
        WriteTimeout: getEnvAsDuration("WRITE_TIMEOUT", 120*time.Second),
        IdleTimeout:  getEnvAsDuration("IDLE_TIMEOUT", 120*time.Second),
    }

    // Set translation enabled based on API key availability
    if cfg.AvalaiAPIKeyTranslation == "" {
        cfg.TranslationEnabled = false
        log.Println("Translation disabled: No AVALAI_API_KEY_TRANSLATION provided")
    }

    // Validate configuration
    if err := cfg.Validate(); err != nil {
        log.Fatalf("Configuration validation failed: %v", err)
    }

    return cfg
}

// Validate checks configuration for required fields and security requirements
func (c *Config) Validate() error {
    var errors []string
    
    // Always validate critical security settings
    if len(c.JWTSecretKey) < 32 {
        errors = append(errors, "JWT_SECRET_KEY must be at least 32 characters long for security")
    }
    
    // Database validation
    if c.DBHost == "" {
        errors = append(errors, "DB_HOST is required")
    }
    if c.DBName == "" {
        errors = append(errors, "DB_NAME is required")
    }
    if c.DBUser == "" {
        errors = append(errors, "DB_USER is required")
    }
    if c.DBPassword == "" {
        errors = append(errors, "DB_PASSWORD is required")
    }
    
    // Production-specific validation
    if strings.ToLower(c.Environment) == "production" {
        productionRequired := map[string]string{
            "AVALAI_API_KEY_EMBEDDING": c.AvalaiAPIKeyEmbedding,
            "JABIR_API_KEY":           c.JabirAPIKey,
            "PINECONE_API_KEY":        c.PineconeAPIKey,
            "PINECONE_INDEX_HOST":     c.PineconeIndexHost,
            "ADMIN_PHONE_NUMBER":      c.AdminPhoneNumber,
        }
        
        for key, value := range productionRequired {
            if value == "" {
                errors = append(errors, fmt.Sprintf("%s is required in production", key))
            }
        }
        
        // Production security checks
        if strings.Contains(strings.ToLower(c.JWTSecretKey), "test") || 
           strings.Contains(strings.ToLower(c.JWTSecretKey), "dev") {
            errors = append(errors, "JWT_SECRET_KEY appears to contain test/dev values - use production secret")
        }
    }
    
    if len(errors) > 0 {
        return fmt.Errorf("configuration errors: %s", strings.Join(errors, "; "))
    }
    
    return nil
}

// GetDatabaseDSN builds PostgreSQL connection string
func (c *Config) GetDatabaseDSN() string {
    return fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=UTC",
        c.DBHost, c.DBUser, c.DBPassword, c.DBName, c.DBPort, c.DBSSLMode)
}

// IsProduction returns true if running in production environment
func (c *Config) IsProduction() bool {
    return strings.ToLower(c.Environment) == "production"
}

// IsDevelopment returns true if running in development environment
func (c *Config) IsDevelopment() bool {
    return strings.ToLower(c.Environment) == "development"
}

// IsTranslationEnabled returns true if translation is available and enabled
func (c *Config) IsTranslationEnabled() bool {
    return c.TranslationEnabled && c.AvalaiAPIKeyTranslation != ""
}

// Helper Functions

// getEnv returns the value of an environment variable or a default.
func getEnv(key, defaultValue string) string {
    if value, exists := os.LookupEnv(key); exists {
        return value
    }
    return defaultValue
}

// getEnvAsInt gets an env var as an integer, with a fallback.
func getEnvAsInt(key string, defaultValue int) int {
    strValue := getEnv(key, "")
    if strValue == "" {
        return defaultValue
    }
    intValue, err := strconv.Atoi(strValue)
    if err != nil {
        log.Printf("Warning: could not parse env var %s as integer. Using default value %d", key, defaultValue)
        return defaultValue
    }
    return intValue
}

// getEnvAsBool gets an env var as a boolean, with a fallback.
func getEnvAsBool(key string, defaultValue bool) bool {
    strValue := getEnv(key, "")
    if strValue == "" {
        return defaultValue
    }
    boolValue, err := strconv.ParseBool(strValue)
    if err != nil {
        log.Printf("Warning: could not parse env var %s as boolean. Using default value %v", key, defaultValue)
        return defaultValue
    }
    return boolValue
}

// getEnvAsSlice gets an env var as a string slice (comma-separated), with a fallback.
func getEnvAsSlice(key string, defaultValue []string) []string {
    strValue := getEnv(key, "")
    if strValue == "" {
        return defaultValue
    }
    
    values := strings.Split(strValue, ",")
    for i, v := range values {
        values[i] = strings.TrimSpace(v)
    }
    return values
}

// getEnvAsDuration gets an env var as a duration, with a fallback.
func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
    strValue := getEnv(key, "")
    if strValue == "" {
        return defaultValue
    }
    duration, err := time.ParseDuration(strValue)
    if err != nil {
        log.Printf("Warning: could not parse env var %s as duration. Using default value %v", key, defaultValue)
        return defaultValue
    }
    return duration
}
