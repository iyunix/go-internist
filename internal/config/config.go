// File: internal/config/config.go

package config

import (
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
	// ADD JABIR API KEY
	JabirAPIKey       string
	PineconeAPIKey        string
	PineconeIndexName     string
	PineconeNamespace     string
	Environment           string
}

// Load loads configuration from environment variables or .env file.
func Load() *Config {
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
		// LOAD JABIR API KEY
		JabirAPIKey:           getEnv("JABIR_API_KEY", ""),
		PineconeAPIKey:        getEnv("PINECONE_API_KEY", ""),
		PineconeIndexName:     getEnv("PINECONE_INDEX_NAME", "medical-articles"),
		PineconeNamespace:     getEnv("PINECONE_NAMESPACE", "UpToDate"),
		Environment:           env,
	}

	if strings.ToLower(env) == "production" {
		missing := []string{}
		if cfg.JWTSecretKey == "" {
			missing = append(missing, "JWT_SECRET_KEY")
		}
		if cfg.AvalaiAPIKeyEmbedding == "" {
			missing = append(missing, "AVALAI_API_KEY_EMBEDDING")
		}
		// VALIDATE JABIR API KEY INSTEAD OF OLD LLM KEY
		if cfg.JabirAPIKey == "" {
			missing = append(missing, "JABIR_API_KEY")
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