// File: cmd/diagnostics/jabir_test.go
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
	log.Println("--- Running Jabir LLM Performance Test ---")

	// --- Test Parameters ---
	const testRuns = 3 // Run the test 3 times to get a stable average

	// --- 1. Load Configuration from .env file ---
	if err := godotenv.Load(".env"); err != nil {
		log.Fatalf("FATAL: Error loading .env file. Make sure it exists at the project root. Error: %v", err)
	}

	// --- 2. Get and Validate Jabir Environment Variable ---
	jabirAPIKey := getEnvOrFatal("JABIR_API_KEY")

	// --- 3. Initialize AI Service for Jabir ---
	// We provide the specific BaseURL for the Jabir project.
	aiService := services.NewAIService(
		"", // No embedding key needed
		jabirAPIKey,
		"", // No embedding URL needed
		"https://openai.jabirproject.org/v1", // Jabir LLM Base URL
		"", // No embedding model name needed
	)

	testPrompt := "Explain what a beta-blocker is in simple terms, in about 100 words."
	log.Printf("Test Prompt: \"%s\"\n\n", testPrompt)
	log.Printf("[INFO] Running %d completion tests...\n", testRuns)

	// --- 4. Measure LLM Completion Time (multiple runs) ---
	var totalCompletionDuration time.Duration

	for i := 1; i <= testRuns; i++ {
		startCompletion := time.Now()
		// Call the GetCompletion function with the specific model name "jabir-400b"
		reply, err := aiService.GetCompletion(context.Background(), "jabir-400b", testPrompt)
		if err != nil {
			log.Printf("ERROR: Completion run #%d failed: %v", i, err)
			continue
		}
		durationCompletion := time.Since(startCompletion)
		totalCompletionDuration += durationCompletion
		log.Printf("[TIMING] Completion run #%d took: %s (response length: %d)", i, durationCompletion, len(reply))
	}

	// --- 5. Print Summary ---
	if testRuns > 0 {
		avgCompletionTime := totalCompletionDuration / time.Duration(testRuns)
		log.Printf("\n--- Test Summary ---")
		log.Printf("Average completion latency over %d runs: %s", testRuns, avgCompletionTime)
		log.Println("--------------------")
	}
}

// getEnvOrFatal retrieves an environment variable or exits if it's not set.
func getEnvOrFatal(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Fatalf("FATAL: Environment variable %s is not set in your .env file.", key)
	}
	return value
}

