// File: internal/config/config.go
package config

import (
    "errors"
    "fmt"
    "os"
    "strconv"
    "strings"
    
    "github.com/joho/godotenv"
)

// Config holds all configuration for the application.
type Config struct {
    // Server Configuration
    ServerPort  string
    Environment string
    LogLevel    string
    Port        string    // ✅ Added missing Port field

    // Security
    JWTSecretKey   string
    AllowedOrigins []string  // ✅ Single definition (removed duplicate)
    StaticDir      string

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

    // SMS Configuration - ✅ Clean field definitions
    SMSAccessKey  string
    SMSTemplateID string  // Keep as string, convert to int in wire.go
    SMSAPIURL     string

    // Application Settings
    AdminPhoneNumber   string
    TranslationEnabled bool
}

func New() (*Config, error) {
    // Load .env file if it exists. In Docker, it won't, and that's OK.
    // godotenv.Load() is smart and won't overwrite existing environment variables.
    // This makes it safe to use in all environments.
    _ = godotenv.Load()

    env := getEnv("GO_ENV", "development")


    cfg := &Config{
        // Server Configuration
        ServerPort:  getEnv("SERVER_PORT", "8080"),
        Environment: env,
        LogLevel:    getEnv("LOG_LEVEL", "INFO"),
        Port:        getEnv("PORT", ""), // ✅ Added Port field

        // Security
        JWTSecretKey:   os.Getenv("JWT_SECRET_KEY"), // No default - must be provided
        AllowedOrigins: getEnvAsSlice("ALLOWED_ORIGINS", []string{}), // ✅ Single definition
        StaticDir:      getEnv("STATIC_DIR", "web/static"),

        // Database Configuration
        DBHost:     os.Getenv("DB_HOST"),     // No default - must be provided
        DBUser:     os.Getenv("DB_USER"),     // No default - must be provided
        DBPassword: os.Getenv("DB_PASSWORD"), // No default - must be provided
        DBName:     os.Getenv("DB_NAME"),     // No default - must be provided
        DBPort:     getEnv("DB_PORT", "5432"),
        DBSSLMode:  getDBSSLMode(env), // Smart default based on environment

        // AI Services
        AvalaiAPIKeyEmbedding:   os.Getenv("AVALAI_API_KEY_EMBEDDING"), // No default
        AvalaiAPIKeyTranslation: os.Getenv("AVALAI_API_KEY_TRANSLATION"), // Optional
        JabirAPIKey:             os.Getenv("JABIR_API_KEY"),              // No default
        EmbeddingModelName:      getEnv("EMBEDDING_MODEL_NAME", "text-embedding-3-large"),

        // Vector Database
        PineconeAPIKey:    os.Getenv("PINECONE_API_KEY"),    // No default
        PineconeIndexHost: os.Getenv("PINECONE_INDEX_HOST"), // No default
        PineconeNamespace: getEnv("PINECONE_NAMESPACE", "UpToDate"),
        RetrievalTopK:     getEnvAsInt("RAG_TOPK", 5),

        // SMS Service - ✅ Clean field population
        SMSAccessKey:  os.Getenv("SMS_ACCESS_KEY"), // No default
        SMSTemplateID: os.Getenv("SMS_TEMPLATE_ID"), // Keep as string
        SMSAPIURL:     os.Getenv("SMS_API_URL"), // No default

        // Application Settings
        AdminPhoneNumber:   os.Getenv("ADMIN_PHONE_NUMBER"), // No default
        TranslationEnabled: getEnvAsBool("TRANSLATION_ENABLED", true),
    }

    // Smart logic: disable translation if no API key provided
    if cfg.AvalaiAPIKeyTranslation == "" {
        cfg.TranslationEnabled = false
    }

    // Set development-friendly CORS origins if none provided
    if len(cfg.AllowedOrigins) == 0 && cfg.Environment == "development" {
        cfg.AllowedOrigins = []string{
            "http://localhost:8080",
            "http://localhost:8081",
        }
    }

    // Validate the fully populated config struct
    if err := cfg.Validate(); err != nil {
        return nil, err
    }

    return cfg, nil
}

// Validate checks configuration for required fields and security best practices.
func (c *Config) Validate() error {
    // Required secrets validation
    required := map[string]string{
        "JWT_SECRET_KEY":           c.JWTSecretKey,
        "DB_HOST":                  c.DBHost,
        "DB_USER":                  c.DBUser,
        "DB_PASSWORD":              c.DBPassword,
        "DB_NAME":                  c.DBName,
        "AVALAI_API_KEY_EMBEDDING": c.AvalaiAPIKeyEmbedding,
        "JABIR_API_KEY":            c.JabirAPIKey,
        "PINECONE_API_KEY":         c.PineconeAPIKey,
        "PINECONE_INDEX_HOST":      c.PineconeIndexHost,
        "SMS_ACCESS_KEY":           c.SMSAccessKey,
        "SMS_API_URL":              c.SMSAPIURL,
        "ADMIN_PHONE_NUMBER":       c.AdminPhoneNumber,
    }

    var missing []string
    for key, value := range required {
        if value == "" {
            missing = append(missing, key)
        }
    }

    if len(missing) > 0 {
        return fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
    }

    // Security validations
    if len(c.JWTSecretKey) < 32 {
        return errors.New("JWT_SECRET_KEY must be at least 32 characters long for security")
    }

    if c.SMSTemplateID == "" {
        return errors.New("SMS_TEMPLATE_ID is required")
    }

    // Production-specific security checks
    if c.IsProduction() {
        if c.DBSSLMode == "disable" {
            return errors.New("DB_SSL_MODE cannot be 'disable' in production - use 'require' or 'verify-full'")
        }
        
        if len(c.AllowedOrigins) == 0 {
            return errors.New("ALLOWED_ORIGINS must be set in production")
        }
        
        // Check for localhost origins in production
        for _, origin := range c.AllowedOrigins {
            if strings.Contains(origin, "localhost") {
                return errors.New("localhost origins not allowed in production")
            }
        }
    }

    return nil
}

// GetDatabaseDSN builds PostgreSQL connection string.
func (c *Config) GetDatabaseDSN() string {
    return fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=UTC",
        c.DBHost, c.DBUser, c.DBPassword, c.DBName, c.DBPort, c.DBSSLMode)
}

// IsProduction returns true if running in production environment.
func (c *Config) IsProduction() bool {
    return strings.ToLower(c.Environment) == "production"
}

// IsTranslationEnabled returns whether translation is enabled
func (c *Config) IsTranslationEnabled() bool {
    return c.TranslationEnabled
}

// Load is kept for backward compatibility but deprecated
// Use New() instead
func Load() *Config {
    cfg, err := New()
    if err != nil {
        panic(fmt.Sprintf("Configuration error: %v", err))
    }
    return cfg
}

// --- Helper Functions ---

func getEnv(key, defaultValue string) string {
    if value, exists := os.LookupEnv(key); exists {
        return value
    }
    return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
    strValue, exists := os.LookupEnv(key)
    if !exists {
        return defaultValue
    }
    intValue, err := strconv.Atoi(strValue)
    if err != nil {
        return defaultValue
    }
    return intValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
    strValue, exists := os.LookupEnv(key)
    if !exists {
        return defaultValue
    }
    boolValue, err := strconv.ParseBool(strValue)
    if err != nil {
        return defaultValue
    }
    return boolValue
}

func getEnvAsSlice(key string, defaultValue []string) []string {
    strValue, exists := os.LookupEnv(key)
    if !exists {
        return defaultValue
    }
    values := strings.Split(strValue, ",")
    // Trim whitespace from each value
    for i := range values {
        values[i] = strings.TrimSpace(values[i])
    }
    return values
}

// getDBSSLMode handles the critical security setting for database SSL
func getDBSSLMode(env string) string {
    mode := os.Getenv("DB_SSL_MODE")
    
    // Force secure defaults based on environment
    if env == "production" && (mode == "" || mode == "disable") {
        return "require" // Force SSL in production
    }
    
    if mode == "" {
        return "disable" // Safe for local development
    }
    
    return mode
}


