package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/rizome-dev/go-openrouter/pkg/errors"
	"github.com/rizome-dev/go-openrouter/pkg/models"
	"github.com/rizome-dev/go-openrouter/pkg"
)

func main() {
	// Get API key from environment
	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		log.Fatal("Please set OPENROUTER_API_KEY environment variable")
	}

	// Example 1: Observable client with metrics
	fmt.Println("=== Observable Client Example ===")
	observableExample(apiKey)

	// Example 2: Retry client with circuit breaker
	fmt.Println("\n=== Retry Client Example ===")
	retryExample(apiKey)

	// Example 3: Advanced routing
	fmt.Println("\n=== Advanced Routing Example ===")
	advancedRoutingExample(apiKey)

	// Example 4: Complete workflow
	fmt.Println("\n=== Complete Workflow Example ===")
	completeWorkflowExample(apiKey)
}

func observableExample(apiKey string) {
	// Create logger and metrics collector
	logger := pkg.NewSimpleLogger(pkg.LogLevelInfo)
	metrics := pkg.NewSimpleMetricsCollector()

	// Create observable client
	client := pkg.NewObservableClient(apiKey,
		pkg.ObservabilityOptions{
			Logger:       logger,
			Metrics:      metrics,
			LogRequests:  true,
			LogResponses: true,
			TrackCosts:   true,
		},
		pkg.WithHTTPReferer("https://github.com/rizome-dev/go-openrouter"),
		pkg.WithXTitle("Comprehensive Example"),
	)

	// Add hooks
	client.AddRequestHook(func(ctx context.Context, operation string, request interface{}) context.Context {
		fmt.Printf("[Request Hook] Operation: %s\n", operation)
		return ctx
	})

	client.AddResponseHook(func(ctx context.Context, operation string, request interface{}, response interface{}, err error) {
		if err != nil {
			fmt.Printf("[Response Hook] Operation %s failed: %v\n", operation, err)
		} else {
			fmt.Printf("[Response Hook] Operation %s succeeded\n", operation)
		}
	})

	// Make some requests
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		resp, err := client.CreateChatCompletion(ctx, models.ChatCompletionRequest{
			Model: "openai/gpt-3.5-turbo",
			Messages: []models.Message{
				models.NewTextMessage(models.RoleUser, fmt.Sprintf("What is %d + %d?", i, i+1)),
			},
			MaxTokens:   intPtr(50),
			Temperature: float64Ptr(0),
		})

		if err != nil {
			log.Printf("Request %d failed: %v", i, err)
			continue
		}

		content, _ := resp.Choices[0].Message.GetTextContent()
		fmt.Printf("Response %d: %s\n", i, content)
	}

	// Print metrics summary
	time.Sleep(3 * time.Second) // Wait for cost tracking
	summary := metrics.GetSummary()
	fmt.Printf("\nMetrics Summary:\n")
	summaryJSON, _ := json.MarshalIndent(summary, "", "  ")
	fmt.Println(string(summaryJSON))
}

func retryExample(apiKey string) {
	// Create retry client with custom config
	retryConfig := &pkg.RetryConfig{
		MaxRetries:    5,
		InitialDelay:  500 * time.Millisecond,
		MaxDelay:      10 * time.Second,
		BackoffFactor: 2.0,
		JitterFactor:  0.1,
		RetryableErrors: map[errors.ErrorCode]bool{
			errors.ErrorCodeTimeout:          true,
			errors.ErrorCodeRateLimited:      true,
			errors.ErrorCodeModelDown:        true,
			errors.ErrorCodeNoAvailableModel: true,
		},
	}

	client := pkg.NewRetryClient(apiKey, retryConfig,
		pkg.WithTimeout(5*time.Second),
	)

	// Simulate a request that might fail
	ctx := context.Background()
	resp, err := client.CreateChatCompletion(ctx, models.ChatCompletionRequest{
		Model: "openai/gpt-4",
		Messages: []models.Message{
			models.NewTextMessage(models.RoleUser, "Explain retry patterns in distributed systems"),
		},
	})

	if err != nil {
		log.Printf("Request failed after retries: %v", err)
		return
	}

	content, _ := resp.Choices[0].Message.GetTextContent()
	fmt.Printf("Response: %s\n", content[:min(200, len(content))]+"...")
}

func advancedRoutingExample(apiKey string) {
	client := pkg.NewClient(apiKey)
	ctx := context.Background()

	// Example 1: Specific provider order with fallbacks
	fmt.Println("\n--- Provider Order Example ---")
	resp, err := client.CreateChatCompletion(ctx, models.ChatCompletionRequest{
		Model: "meta-llama/llama-3.1-70b-instruct",
		Messages: []models.Message{
			models.NewTextMessage(models.RoleUser, "Hello!"),
		},
		Provider: models.NewProviderPreferences().
			WithOrder("together", "deepinfra", "anyscale").
			WithFallbacks(true).
			WithSort(models.SortByThroughput),
	})

	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Printf("Model used: %s\n", resp.Model)
	}

	// Example 2: Price constraints
	fmt.Println("\n--- Price Constraint Example ---")
	resp, err = client.CreateChatCompletion(ctx, models.ChatCompletionRequest{
		Model: "openai/gpt-4",
		Messages: []models.Message{
			models.NewTextMessage(models.RoleUser, "What is 2+2?"),
		},
		Provider: models.NewProviderPreferences().
			WithMaxPrice(0.01, 0.02). // Max $0.01/M prompt, $0.02/M completion
			WithSort(models.SortByPrice),
	})

	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		content, _ := resp.Choices[0].Message.GetTextContent()
		fmt.Printf("Response: %s\n", content)
	}

	// Example 3: Quantization filtering
	fmt.Println("\n--- Quantization Example ---")
	resp, err = client.CreateChatCompletion(ctx, models.ChatCompletionRequest{
		Model: "meta-llama/llama-3.1-8b-instruct",
		Messages: []models.Message{
			models.NewTextMessage(models.RoleUser, "Explain quantization"),
		},
		Provider: models.NewProviderPreferences().
			WithQuantizations(models.QuantizationFP16, models.QuantizationBF16).
			WithDataCollection(models.DataCollectionDeny),
		MaxTokens: intPtr(100),
	})

	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		content, _ := resp.Choices[0].Message.GetTextContent()
		fmt.Printf("Response: %s\n", content[:min(200, len(content))]+"...")
	}
}

func completeWorkflowExample(apiKey string) {
	// Create a comprehensive client setup
	logger := pkg.NewSimpleLogger(pkg.LogLevelInfo)
	metrics := pkg.NewSimpleMetricsCollector()

	// Base client with observability
	baseClient := pkg.NewObservableClient(apiKey,
		pkg.ObservabilityOptions{
			Logger:       logger,
			Metrics:      metrics,
			LogRequests:  true,
			LogResponses: true,
			TrackCosts:   true,
		},
	)

	// Wrap with retry logic
	retryClient := &pkg.RetryClient{
		Client: baseClient.Client,
	}

	ctx := context.Background()

	// Step 1: Research with web search
	fmt.Println("\n--- Step 1: Research ---")
	webHelper := pkg.NewWebSearchHelper(retryClient.Client)

	researchResp, err := webHelper.CreateWithWebSearch(ctx,
		"Latest advances in quantum computing 2024",
		"openai/gpt-4",
		&pkg.SearchOptions{
			MaxResults: 5,
		},
	)

	if err != nil {
		log.Printf("Research failed: %v", err)
		return
	}

	researchContent, _ := researchResp.Choices[0].Message.GetTextContent()
	fmt.Printf("Research findings: %s\n", researchContent[:min(300, len(researchContent))]+"...")

	// Extract citations
	citations := pkg.ExtractCitations(researchResp)
	fmt.Printf("\nFound %d citations\n", len(citations))

	// Step 2: Create structured summary
	fmt.Println("\n--- Step 2: Structured Summary ---")
	structured := pkg.NewStructuredOutput(retryClient.Client)

	type ResearchSummary struct {
		Topic         string   `json:"topic" description:"Main research topic"`
		KeyFindings   []string `json:"key_findings" description:"List of key findings"`
		Breakthroughs []struct {
			Title       string `json:"title" description:"Breakthrough title"`
			Description string `json:"description" description:"Brief description"`
			Impact      string `json:"impact" description:"Potential impact"`
		} `json:"breakthroughs" description:"Major breakthroughs"`
		FutureOutlook string `json:"future_outlook" description:"Future outlook and predictions"`
	}

	summaryResp, err := structured.CreateWithSchema(ctx,
		models.ChatCompletionRequest{
			Model: "openai/gpt-4",
			Messages: []models.Message{
				models.NewTextMessage(models.RoleUser,
					fmt.Sprintf("Based on this research about quantum computing, create a structured summary:\n\n%s", researchContent)),
			},
		},
		"research_summary",
		ResearchSummary{},
	)

	if err != nil {
		log.Printf("Structured summary failed: %v", err)
		return
	}

	var summary ResearchSummary
	if err := pkg.ParseStructuredResponse(summaryResp, &summary); err != nil {
		log.Printf("Failed to parse summary: %v", err)
		return
	}

	fmt.Printf("\nStructured Summary:\n")
	summaryJSON, _ := json.MarshalIndent(summary, "", "  ")
	fmt.Println(string(summaryJSON))

	// Step 3: Generate visual representation prompt
	fmt.Println("\n--- Step 3: Visual Representation ---")
	imagePromptResp, err := retryClient.CreateChatCompletion(ctx, models.ChatCompletionRequest{
		Model: "openai/gpt-4",
		Messages: []models.Message{
			models.NewTextMessage(models.RoleSystem, "You are an expert at creating prompts for image generation."),
			models.NewTextMessage(models.RoleUser,
				fmt.Sprintf("Create a detailed prompt for an image that visualizes this quantum computing breakthrough: %s",
					summary.Breakthroughs[0].Title)),
		},
		MaxTokens: intPtr(150),
	})

	if err != nil {
		log.Printf("Image prompt generation failed: %v", err)
		return
	}

	imagePrompt, _ := imagePromptResp.Choices[0].Message.GetTextContent()
	fmt.Printf("\nImage generation prompt: %s\n", imagePrompt)

	// Print final metrics
	fmt.Println("\n--- Final Metrics ---")
	finalSummary := metrics.GetSummary()
	finalJSON, _ := json.MarshalIndent(finalSummary, "", "  ")
	fmt.Println(string(finalJSON))
}

func intPtr(i int) *int {
	return &i
}

func float64Ptr(f float64) *float64 {
	return &f
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
