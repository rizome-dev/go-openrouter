package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/rizome-dev/go-openrouter/pkg/models"
	"github.com/rizome-dev/go-openrouter/pkg/openrouter"
)

func main() {
	// Get API key from environment
	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		log.Fatal("Please set OPENROUTER_API_KEY environment variable")
	}

	// Create client
	client := openrouter.NewClient(apiKey,
		openrouter.WithHTTPReferer("https://github.com/rizome-dev/go-openrouter"),
		openrouter.WithXTitle("OpenRouterGo Example"),
	)

	// Create a simple chat completion
	fmt.Println("Sending chat completion request...")
	resp, err := client.CreateChatCompletion(context.Background(), models.ChatCompletionRequest{
		Model: "google/gemini-2.5-pro", // You can also try "anthropic/claude-3.5-sonnet"
		Messages: []models.Message{
			models.NewTextMessage(models.RoleSystem, "You are a helpful assistant."),
			models.NewTextMessage(models.RoleUser, "What is the capital of France?"),
		},
		Temperature: float64Ptr(0.7),
		MaxTokens:   intPtr(150), // Be careful with low limits on models that include reasoning
	})

	if err != nil {
		log.Fatalf("Error creating completion: %v", err)
	}

	// Print response
	fmt.Printf("\nModel used: %s\n", resp.Model)
	if len(resp.Choices) > 0 && resp.Choices[0].Message != nil {
		msg := resp.Choices[0].Message
		content := getMessageContent(msg)
		fmt.Printf("Response: %s\n", content)

		// Check if reasoning is present (some models like Gemini include this)
		// Note: When using low max_tokens with models that include reasoning,
		// the actual content might be truncated in favor of the reasoning field
		if msg.Reasoning != "" {
			fmt.Printf("\nReasoning: %s\n", msg.Reasoning)
		}

		// Print finish reason
		if resp.Choices[0].FinishReason != "" {
			fmt.Printf("\nFinish reason: %s\n", resp.Choices[0].FinishReason)
		}
	}

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
