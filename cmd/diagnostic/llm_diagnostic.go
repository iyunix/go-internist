// File: cmd/diagnostic/llm_diagnostic.go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    openai "github.com/sashabaranov/go-openai"
    "github.com/joho/godotenv"
)

func main() {
    fmt.Println("🚀 Testing Jabir with Go OpenAI SDK...")

    // Load environment variables
    if err := godotenv.Load(`G:\go_internist\.env`); err != nil {
        log.Printf("Warning: Could not load .env file: %v", err)
    }

    apiKey := os.Getenv("JABIR_API_KEY")
    if apiKey == "" {
        log.Fatal("JABIR_API_KEY not set in environment")
    }

    fmt.Printf("✅ API Key: %s\n", apiKey)

    // Initialize client with custom API key and base URL (exactly like Python)
    config := openai.DefaultConfig(apiKey)
    config.BaseURL = "https://openai.jabirproject.org/v1"
    client := openai.NewClientWithConfig(config)

    // Create a chat completion (exactly like Python)
    resp, err := client.CreateChatCompletion(
        context.Background(),
        openai.ChatCompletionRequest{
            Model: "jabir-400b",
            Messages: []openai.ChatCompletionMessage{
                {
                    Role:    openai.ChatMessageRoleUser,
                    Content: "What is the answer to life, universe and everything?",
                },
            },
        },
    )

    if err != nil {
        log.Fatalf("❌ Chat completion failed: %v", err)
    }

    // Print the response (exactly like Python)
    fmt.Printf("✅ Response: %s\n", resp.Choices[0].Message.Content)
}
