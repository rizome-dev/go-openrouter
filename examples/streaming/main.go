package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"time"

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
		openrouter.WithXTitle("OpenRouterGo Streaming Example"),
	)

	// Example 1: Basic streaming
	fmt.Println("=== Basic Streaming Example ===")
	basicStreaming(client)

	// Example 2: Streaming with cancellation
	fmt.Println("\n=== Streaming with Cancellation Example ===")
	streamingWithCancellation(client)

	// Example 3: Concurrent streaming
	fmt.Println("\n=== Concurrent Streaming Example ===")
	concurrentStreaming(client)
}

func basicStreaming(client *openrouter.Client) {
	ctx := context.Background()

	// Create streaming request
	stream, err := client.CreateChatCompletionStream(ctx, models.ChatCompletionRequest{
		Model: "openai/gpt-3.5-turbo",
		Messages: []models.Message{
			models.NewTextMessage(models.RoleUser, "Write a haiku about programming"),
		},
		Temperature: float64Ptr(0.7),
	})

	if err != nil {
		log.Fatalf("Error creating stream: %v", err)
	}
	defer stream.Close()

	fmt.Print("Response: ")
	// Read stream
	for {
		chunk, err := stream.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("Error reading stream: %v", err)
			break
		}

		// Print content from delta
		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta != nil {
			content, _ := chunk.Choices[0].Delta.GetTextContent()
			fmt.Print(content)
		}
	}
	fmt.Println()
}

func streamingWithCancellation(client *openrouter.Client) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create streaming request
	stream, err := client.CreateChatCompletionStream(ctx, models.ChatCompletionRequest{
		Model: "openai/gpt-3.5-turbo",
		Messages: []models.Message{
			models.NewTextMessage(models.RoleUser, "Count from 1 to 100 slowly"),
		},
	})

	if err != nil {
		log.Fatalf("Error creating stream: %v", err)
	}
	defer stream.Close()

	fmt.Print("Response (will be cancelled after 5 seconds): ")

	// Read stream until cancelled
	for {
		chunk, err := stream.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			if err == context.DeadlineExceeded {
				fmt.Print("\n[Stream cancelled due to timeout]")
			} else {
				log.Printf("\nError reading stream: %v", err)
			}
			break
		}

		// Print content
		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta != nil {
			content, _ := chunk.Choices[0].Delta.GetTextContent()
			fmt.Print(content)
		}
	}
	fmt.Println()
}

func concurrentStreaming(client *openrouter.Client) {
	// Create concurrent client
	concurrentClient := openrouter.NewConcurrentClient(
		os.Getenv("OPENROUTER_API_KEY"),
		3, // Max 3 concurrent requests
		openrouter.WithHTTPReferer("https://github.com/rizome-dev/go-openrouter"),
		openrouter.WithXTitle("OpenRouterGo Concurrent Example"),
	)

	// Create multiple requests
	requests := []models.ChatCompletionRequest{
		{
			Model: "openai/gpt-3.5-turbo",
			Messages: []models.Message{
				models.NewTextMessage(models.RoleUser, "What is 2+2?"),
			},
		},
		{
			Model: "openai/gpt-3.5-turbo",
			Messages: []models.Message{
				models.NewTextMessage(models.RoleUser, "What is the capital of France?"),
			},
		},
		{
			Model: "openai/gpt-3.5-turbo",
			Messages: []models.Message{
				models.NewTextMessage(models.RoleUser, "What color is the sky?"),
			},
		},
	}

	// Execute concurrently
	ctx := context.Background()
	results := concurrentClient.CreateChatCompletionsConcurrent(ctx, requests)

	// Print results
	for _, result := range results {
		if result.Error != nil {
			fmt.Printf("Request %d failed: %v\n", result.Index, result.Error)
		} else {
			content, _ := result.Response.Choices[0].Message.GetTextContent()
			fmt.Printf("Request %d: %s\n", result.Index, content)
		}
	}
}

func float64Ptr(f float64) *float64 {
	return &f
}
