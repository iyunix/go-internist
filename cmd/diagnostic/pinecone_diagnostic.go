package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/iyunix/go-internist/internal/services"
	"github.com/joho/godotenv"
)

func main() {
	log.Println("--- Running Pinecone Performance Test ---")

	// --- Test Parameters ---
	const testRuns = 5
	const topK = 10

	// --- 1. Load Configuration from .env file ---
	if err := godotenv.Load(".env"); err != nil {
		log.Fatalf("FATAL: Error loading .env file. Make sure it exists at the project root. Error: %v", err)
	}

	// --- 2. Get and Validate Environment Variables ---
	pineconeAPIKey := getEnvOrFatal("PINECONE_API_KEY")
	pineconeIndexHost := getEnvOrFatal("PINECONE_INDEX_HOST")
	pineconeNamespace := getEnvOrFatal("PINECONE_NAMESPACE")
	embeddingAPIKey := getEnvOrFatal("AVALAI_API_KEY_EMBEDDING")

	// --- 3. Initialize Services ---
	aiService := services.NewAIService(embeddingAPIKey, "") // LLM key not needed for this test
	pineconeService, err := services.NewPineconeService(pineconeAPIKey, pineconeIndexHost, pineconeNamespace)
	if err != nil {
		log.Fatalf("FATAL: Failed to initialize Pinecone service: %v", err)
	}

	testQuery := "What is the standard dosage for metoprolol?"
	log.Printf("Test Query: \"%s\"\n\n", testQuery)

	// --- 4. Measure Embedding Creation Time (once) ---
	startEmbedding := time.Now()
	embedding, err := aiService.CreateEmbedding(context.Background(), testQuery)
	if err != nil {
		log.Fatalf("FATAL: Failed to create embedding: %v", err)
	}
	durationEmbedding := time.Since(startEmbedding)
	log.Printf("[TIMING] ✅ Embedding creation took: %s\n", durationEmbedding)

	// --- 5. Measure Pinecone Query Time (multiple runs) ---
	var totalQueryDuration time.Duration
	var queryDurations []time.Duration

	log.Printf("[INFO] Running %d queries with topK=%d...\n", testRuns, topK)
	for i := 1; i <= testRuns; i++ {
		startQuery := time.Now()
		results, err := pineconeService.QuerySimilar(context.Background(), embedding, topK)
		if err != nil {
			log.Printf("ERROR: Query run #%d failed: %v", i, err)
			continue
		}
		durationQuery := time.Since(startQuery)
		queryDurations = append(queryDurations, durationQuery)
		totalQueryDuration += durationQuery
		log.Printf("[TIMING] Query run #%d took: %s (found %d matches)", i, durationQuery, len(results))
	}

	// --- 6. Print Summary ---
	avgQueryTime := totalQueryDuration / time.Duration(testRuns)
	log.Printf("\n--- Test Summary ---")
	log.Printf("Average query latency over %d runs: %s", testRuns, avgQueryTime)
	log.Println("--------------------")
}

// getEnvOrFatal retrieves an environment variable or exits if it's not set.
func getEnvOrFatal(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Fatalf("FATAL: Environment variable %s is not set in your .env file.", key)
	}
	return value
}

