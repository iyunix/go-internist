// File: internal/config/config.go
package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	ServerPort             string
	JWTSecretKey           string
	AvalaiAPIKeyEmbedding  string
	AvalaiAPIKeyLLM        string
	PineconeAPIKey         string
	PineconeIndexName      string
	PineconeNamespace      string
}

func Load() *Config {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, using environment variables")
	}

	return &Config{
		ServerPort:             getEnv("SERVER_PORT", "8080"),
		JWTSecretKey:           getEnv("JWT_SECRET_KEY", "default-secret"),
		AvalaiAPIKeyEmbedding:  getEnv("AVALAI_API_KEY_EMBEDDING", ""),
		AvalaiAPIKeyLLM:        getEnv("AVALAI_API_KEY_LLM", ""),
		PineconeAPIKey:         getEnv("PINECONE_API_KEY", ""),
		PineconeIndexName:      getEnv("PINECONE_INDEX_NAME", "medical-articles"),
		PineconeNamespace:      getEnv("PINECONE_NAMESPACE", "UpToDate"),
	}
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}