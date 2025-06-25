package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/rizome-dev/openroutergo/pkg/models"
	"github.com/rizome-dev/openroutergo/pkg/openrouter"
)

func main() {
	// Get API key from environment
	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		log.Fatal("Please set OPENROUTER_API_KEY environment variable")
	}

	// Create client
	client := openrouter.NewClient(apiKey,
		openrouter.WithHTTPReferer("https://github.com/rizome-dev/openroutergo"),
		openrouter.WithXTitle("OpenRouterGo Example"),
	)

	// Create a simple chat completion
	fmt.Println("Sending chat completion request...")
	resp, err := client.CreateChatCompletion(context.Background(), models.ChatCompletionRequest{
		Model: "openai/gpt-3.5-turbo",
		Messages: []models.Message{
			models.NewTextMessage(models.RoleSystem, "You are a helpful assistant."),
			models.NewTextMessage(models.RoleUser, "What is the capital of France?"),
		},
		Temperature: float64Ptr(0.7),
		MaxTokens:   intPtr(100),
	})

	if err != nil {
		log.Fatalf("Error creating completion: %v", err)
	}

	// Print response
	fmt.Printf("\nModel used: %s\n", resp.Model)
	fmt.Printf("Response: %s\n", getMessageContent(resp.Choices[0].Message))

	// Print usage if available
	if resp.Usage != nil {
		fmt.Printf("\nToken Usage:\n")
		fmt.Printf("  Prompt: %d\n", resp.Usage.PromptTokens)
		fmt.Printf("  Completion: %d\n", resp.Usage.CompletionTokens)
		fmt.Printf("  Total: %d\n", resp.Usage.TotalTokens)
	}
}

func getMessageContent(msg *models.Message) string {
	if msg == nil {
		return ""
	}
	content, _ := msg.GetTextContent()
	return content
}

func float64Ptr(f float64) *float64 {
	return &f
}

func intPtr(i int) *int {
	return &i
}