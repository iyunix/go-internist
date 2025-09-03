// File: internal/config/config.go

package config

import (
    "fmt"
    "log"
    "os"
    "strings"

    "github.com/joho/godotenv"
)

type Config struct {
    ServerPort            string
    JWTSecretKey          string
    AvalaiAPIKeyEmbedding string
    AvalaiAPIKeyLLM       string
    PineconeAPIKey        string
    PineconeIndexName     string
    PineconeNamespace     string
    Environment           string // New field: environment profile (development, production etc.)
}

// Load loads configuration from environment variables or .env file.
func Load() *Config {
    // Load .env only if environment variable ENV != "production"
    env := os.Getenv("ENV")
    if strings.ToLower(env) != "production" {
        if err := godotenv.Load(); err != nil {
            log.Println("No .env file found or error loading it; continuing with environment variables")
        }
    }

    cfg := &Config{
        ServerPort:            getEnv("SERVER_PORT", "8080"),
        JWTSecretKey:          getEnv("JWT_SECRET_KEY", ""),
        AvalaiAPIKeyEmbedding: getEnv("AVALAI_API_KEY_EMBEDDING", ""),
        AvalaiAPIKeyLLM:       getEnv("AVALAI_API_KEY_LLM", ""),
        PineconeAPIKey:        getEnv("PINECONE_API_KEY", ""),
        PineconeIndexName:     getEnv("PINECONE_INDEX_NAME", "medical-articles"),
        PineconeNamespace:     getEnv("PINECONE_NAMESPACE", "UpToDate"),
        Environment:           env,
    }

    // Validate required secrets in production environment
    if strings.ToLower(env) == "production" {
        missing := []string{}
        if cfg.JWTSecretKey == "" {
            missing = append(missing, "JWT_SECRET_KEY")
        }
        if cfg.AvalaiAPIKeyEmbedding == "" {
            missing = append(missing, "AVALAI_API_KEY_EMBEDDING")
        }
        if cfg.AvalaiAPIKeyLLM == "" {
            missing = append(missing, "AVALAI_API_KEY_LLM")
        }
        if cfg.PineconeAPIKey == "" {
            missing = append(missing, "PINECONE_API_KEY")
        }

        if len(missing) > 0 {
            log.Fatalf("Missing required environment variables for production: %v", missing)
        }
    }

    return cfg
}

// getEnv returns the value of the environment variable or the default if unset.
func getEnv(key, defaultValue string) string {
    if value, exists := os.LookupEnv(key); exists {
        return value
    }
    return defaultValue
}
