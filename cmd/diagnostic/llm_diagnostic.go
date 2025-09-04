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
	log.Println("--- Running LLM Performance Test ---")

	// --- Test Parameters ---
	const testRuns = 3 // Run the test 3 times to get a stable average

	// --- 1. Load Configuration from .env file ---
	if err := godotenv.Load(".env"); err != nil {
		log.Fatalf("FATAL: Error loading .env file. Make sure it exists at the project root. Error: %v", err)
	}

	// --- 2. Get and Validate Environment Variable ---
	llmAPIKey := getEnvOrFatal("AVALAI_API_KEY_LLM")

	// --- 3. Initialize AI Service ---
	aiService := services.NewAIService("", llmAPIKey) // Embedding key is not needed

	testPrompt := "Explain what a beta-blocker is in simple terms, in about 100 words."
	log.Printf("Test Prompt: \"%s\"\n\n", testPrompt)
	log.Printf("[INFO] Running %d completion tests...\n", testRuns)

	// --- 4. Measure LLM Completion Time (multiple runs) ---
	var totalCompletionDuration time.Duration
	var completionDurations []time.Duration

	for i := 1; i <= testRuns; i++ {
		startCompletion := time.Now()
		reply, err := aiService.GetCompletion(context.Background(), testPrompt)
		if err != nil {
			log.Printf("ERROR: Completion run #%d failed: %v", i, err)
			continue
		}
		durationCompletion := time.Since(startCompletion)
		completionDurations = append(completionDurations, durationCompletion)
		totalCompletionDuration += durationCompletion
		log.Printf("[TIMING] Completion run #%d took: %s (response length: %d)", i, durationCompletion, len(reply))
	}

	// --- 5. Print Summary ---
	avgCompletionTime := totalCompletionDuration / time.Duration(testRuns)
	log.Printf("\n--- Test Summary ---")
	log.Printf("Average completion latency over %d runs: %s", testRuns, avgCompletionTime)
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

