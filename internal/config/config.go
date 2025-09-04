// File: internal/config/config.go
package config

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	ServerPort            string
	JWTSecretKey          string
	AvalaiAPIKeyEmbedding string
	JabirAPIKey           string
	// NEW: The name of the model to use for creating embeddings.
	EmbeddingModelName string
	PineconeAPIKey        string
	PineconeIndexHost     string
	PineconeNamespace     string
	RetrievalTopK         int
	Environment           string
}

// Load reads configuration from environment variables or .env file.
func Load() *Config {
	env := os.Getenv("ENV")
	if strings.ToLower(env) != "production" {
		if err := godotenv.Load(); err != nil {
			log.Println("No .env file found; continuing with environment variables")
		}
	}

	cfg := &Config{
		ServerPort:            getEnv("SERVER_PORT", "8080"),
		JWTSecretKey:          getEnv("JWT_SECRET_KEY", ""),
		AvalaiAPIKeyEmbedding: getEnv("AVALAI_API_KEY_EMBEDDING", ""),
		JabirAPIKey:           getEnv("JABIR_API_KEY", ""),
		// Load the embedding model name, defaulting to a common one.
		// IMPORTANT: This should be set in the .env file to match your index.
		EmbeddingModelName:    getEnv("EMBEDDING_MODEL_NAME", "text-embedding-ada-002"),
		PineconeAPIKey:        getEnv("PINECONE_API_KEY", ""),
		PineconeIndexHost:     getEnv("PINECONE_INDEX_HOST", ""),
		PineconeNamespace:     getEnv("PINECONE_NAMESPACE", "UpToDate"),
		RetrievalTopK:         getEnvAsInt("RAG_TOPK", 8), // Default to 8
		Environment:           env,
	}

	// Validation for production environments
	if strings.ToLower(env) == "production" {
		missing := []string{}
		if cfg.JWTSecretKey == "" {
			missing = append(missing, "JWT_SECRET_KEY")
		}
		if cfg.AvalaiAPIKeyEmbedding == "" {
			missing = append(missing, "AVALAI_API_KEY_EMBEDDING")
		}
		if cfg.JabirAPIKey == "" {
			missing = append(missing, "JABIR_API_KEY")
		}
		if cfg.PineconeAPIKey == "" {
			missing = append(missing, "PINECONE_API_KEY")
		}
		if cfg.PineconeIndexHost == "" {
			missing = append(missing, "PINECONE_INDEX_HOST")
		}
		if len(missing) > 0 {
			log.Fatalf("Missing required production environment variables: %v", missing)
		}
	}

	return cfg
}

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
		log.Printf("Warning: could not parse env var %s as integer. Using default value.", key)
		return defaultValue
	}
	return intValue
}