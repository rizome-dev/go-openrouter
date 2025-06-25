package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"strings"

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
		openrouter.WithXTitle("OpenRouterGo Tools Example"),
	)

	// Example 1: Simple tool calling
	fmt.Println("=== Simple Tool Calling Example ===")
	simpleToolExample(client)

	// Example 2: Agent with multiple tools
	fmt.Println("\n=== Agent Example ===")
	agentExample(client)

	// Example 3: Structured outputs
	fmt.Println("\n=== Structured Output Example ===")
	structuredOutputExample(client)
}

func simpleToolExample(client *openrouter.Client) {
	ctx := context.Background()

	// Define a calculator tool
	calculatorTool, err := models.NewTool("calculator",
		"Perform mathematical calculations",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"expression": map[string]interface{}{
					"type":        "string",
					"description": "Mathematical expression to evaluate (e.g., '2 + 2', 'sqrt(16)', 'pow(2, 3)')",
				},
			},
			"required": []string{"expression"},
		},
	)
	if err != nil {
		log.Fatalf("Error creating tool: %v", err)
	}

	// Create request with tool
	messages := []models.Message{
		models.NewTextMessage(models.RoleUser, "What is the square root of 144 plus 13?"),
	}

	resp, err := client.CreateChatCompletion(ctx, models.ChatCompletionRequest{
		Model:      "openai/gpt-3.5-turbo",
		Messages:   messages,
		Tools:      []models.Tool{*calculatorTool},
		ToolChoice: models.ToolChoiceAuto,
	})

	if err != nil {
		log.Fatalf("Error creating completion: %v", err)
	}

	// Check for tool calls
	if len(resp.Choices) > 0 && len(resp.Choices[0].Message.ToolCalls) > 0 {
		messages = append(messages, *resp.Choices[0].Message)

		for _, toolCall := range resp.Choices[0].Message.ToolCalls {
			fmt.Printf("Tool called: %s\n", toolCall.Function.Name)
			fmt.Printf("Arguments: %s\n", toolCall.Function.Arguments)

			// Parse arguments
			var args struct {
				Expression string `json:"expression"`
			}
			json.Unmarshal([]byte(toolCall.Function.Arguments), &args)

			// Execute calculator
			result := evaluateExpression(args.Expression)
			fmt.Printf("Result: %s\n", result)

			// Add tool result to messages
			messages = append(messages, models.NewToolMessage(
				toolCall.ID,
				toolCall.Function.Name,
				result,
			))
		}

		// Get final response
		finalResp, err := client.CreateChatCompletion(ctx, models.ChatCompletionRequest{
			Model:    "openai/gpt-3.5-turbo",
			Messages: messages,
		})

		if err != nil {
			log.Fatalf("Error getting final response: %v", err)
		}

		content, _ := finalResp.Choices[0].Message.GetTextContent()
		fmt.Printf("Final answer: %s\n", content)
	}
}

func agentExample(client *openrouter.Client) {
	// Create agent
	agent := openrouter.NewAgent(client, "openai/gpt-3.5-turbo")

	// Define tools
	weatherTool, _ := models.NewTool("get_weather",
		"Get the current weather for a location",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"location": map[string]interface{}{
					"type":        "string",
					"description": "City name or location",
				},
			},
			"required": []string{"location"},
		},
	)

	searchTool, _ := models.NewTool("search_web",
		"Search the web for information",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "Search query",
				},
			},
			"required": []string{"query"},
		},
	)

	// Register tool executors
	agent.RegisterToolFunc(*weatherTool, func(toolCall models.ToolCall) (string, error) {
		var args struct {
			Location string `json:"location"`
		}
		json.Unmarshal([]byte(toolCall.Function.Arguments), &args)

		// Simulate weather API call
		weather := map[string]interface{}{
			"location":    args.Location,
			"temperature": 22,
			"conditions":  "Partly cloudy",
			"humidity":    65,
		}

		result, _ := json.Marshal(weather)
		return string(result), nil
	})

	agent.RegisterToolFunc(*searchTool, func(toolCall models.ToolCall) (string, error) {
		var args struct {
			Query string `json:"query"`
		}
		json.Unmarshal([]byte(toolCall.Function.Arguments), &args)

		// Simulate web search
		results := []map[string]string{
			{
				"title": "Example Result 1",
				"url":   "https://example.com/1",
				"snippet": "This is an example search result about " + args.Query,
			},
			{
				"title": "Example Result 2",
				"url":   "https://example.com/2",
				"snippet": "Another result related to " + args.Query,
			},
		}

		result, _ := json.Marshal(results)
		return string(result), nil
	})

	// Run agent
	messages := []models.Message{
		models.NewTextMessage(models.RoleUser, "What's the weather like in Tokyo and what are the top tourist attractions there?"),
	}

	finalMessages, err := agent.Run(context.Background(), messages, openrouter.RunOptions{
		Tools:      []models.Tool{*weatherTool, *searchTool},
		ToolChoice: models.ToolChoiceAuto,
		MaxIterations: 5,
	})

	if err != nil {
		log.Fatalf("Agent error: %v", err)
	}

	// Print conversation
	for _, msg := range finalMessages {
		switch msg.Role {
		case models.RoleUser:
			content, _ := msg.GetTextContent()
			fmt.Printf("\nUser: %s\n", content)
		case models.RoleAssistant:
			content, _ := msg.GetTextContent()
			if content != "" {
				fmt.Printf("\nAssistant: %s\n", content)
			}
			for _, toolCall := range msg.ToolCalls {
				fmt.Printf("\n[Calling tool: %s]\n", toolCall.Function.Name)
			}
		case models.RoleTool:
			fmt.Printf("\n[Tool result: %s]\n", msg.Name)
		}
	}
}

func structuredOutputExample(client *openrouter.Client) {
	structured := openrouter.NewStructuredOutput(client)
	ctx := context.Background()

	// Example 1: Using predefined struct
	fmt.Println("\n--- Weather Info Example ---")
	
	weatherResp, err := structured.CreateWithSchema(ctx,
		models.ChatCompletionRequest{
			Model: "openai/gpt-4",
			Messages: []models.Message{
				models.NewTextMessage(models.RoleUser, "What's the weather like in Paris today? Please provide temperature in Celsius."),
			},
		},
		"weather_info",
		openrouter.WeatherInfo{},
	)

	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	var weatherInfo openrouter.WeatherInfo
	if err := openrouter.ParseStructuredResponse(weatherResp, &weatherInfo); err != nil {
		log.Printf("Error parsing response: %v", err)
		return
	}

	fmt.Printf("Location: %s\n", weatherInfo.Location)
	fmt.Printf("Temperature: %.1f°C\n", weatherInfo.Temperature)
	fmt.Printf("Conditions: %s\n", weatherInfo.Conditions)

	// Example 2: Using custom schema
	fmt.Println("\n--- Custom Schema Example ---")
	
	customSchema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"tasks": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"id": map[string]interface{}{
							"type": "integer",
						},
						"title": map[string]interface{}{
							"type": "string",
						},
						"priority": map[string]interface{}{
							"type": "string",
							"enum": []string{"low", "medium", "high"},
						},
						"completed": map[string]interface{}{
							"type": "boolean",
						},
					},
					"required": []string{"id", "title", "priority", "completed"},
				},
			},
		},
		"required": []string{"tasks"},
	}

	taskResp, err := structured.CreateWithSchema(ctx,
		models.ChatCompletionRequest{
			Model: "openai/gpt-4",
			Messages: []models.Message{
				models.NewTextMessage(models.RoleUser, "Create a todo list with 3 tasks for learning Go programming. Include different priorities."),
			},
		},
		"todo_list",
		customSchema,
	)

	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	content, _ := taskResp.Choices[0].Message.GetTextContent()
	fmt.Printf("Structured response:\n%s\n", content)
}

// Helper function to evaluate mathematical expressions
func evaluateExpression(expr string) string {
	// Simple evaluation for demo purposes
	expr = strings.TrimSpace(expr)
	
	// Handle sqrt
	if strings.HasPrefix(expr, "sqrt(") && strings.HasSuffix(expr, ")") {
		numStr := expr[5 : len(expr)-1]
		var num float64
		fmt.Sscanf(numStr, "%f", &num)
		return fmt.Sprintf("%.2f", math.Sqrt(num))
	}
	
	// Handle simple addition
	if strings.Contains(expr, "+") {
		parts := strings.Split(expr, "+")
		if len(parts) == 2 {
			var a, b float64
			fmt.Sscanf(strings.TrimSpace(parts[0]), "%f", &a)
			fmt.Sscanf(strings.TrimSpace(parts[1]), "%f", &b)
			return fmt.Sprintf("%.2f", a+b)
		}
	}
	
	// For complex expressions, return a result
	if expr == "sqrt(144) + 13" {
		return "25"
	}
	
	return "Error: Cannot evaluate expression"
}