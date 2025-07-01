package e2e

import (
	"context"
	"encoding/json"

	"github.com/rizome-dev/go-openrouter/pkg/models"
	"github.com/rizome-dev/go-openrouter/pkg/openrouter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (suite *E2ETestSuite) TestBasicToolCalling() {
	ctx := context.Background()

	// Define a weather tool
	weatherTool, err := models.NewTool("get_weather",
		"Get the current weather for a location",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"location": map[string]interface{}{
					"type":        "string",
					"description": "The city and state, e.g. San Francisco, CA",
				},
				"unit": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"celsius", "fahrenheit"},
					"description": "The temperature unit",
				},
			},
			"required": []string{"location"},
		},
	)
	require.NoError(suite.T(), err)

	// First request with tool
	req := models.ChatCompletionRequest{
		Model: "openai/gpt-4o-mini",
		Messages: []models.Message{
			models.NewTextMessage(models.RoleUser, "What's the weather like in Tokyo?"),
		},
		Tools:      []models.Tool{*weatherTool},
		ToolChoice: models.ToolChoiceAuto,
	}

	resp, err := suite.client.CreateChatCompletion(ctx, req)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), resp)

	// Should have tool calls
	assert.Greater(suite.T(), len(resp.Choices), 0)
	assert.NotNil(suite.T(), resp.Choices[0].Message)
	assert.Greater(suite.T(), len(resp.Choices[0].Message.ToolCalls), 0)

	toolCall := resp.Choices[0].Message.ToolCalls[0]
	assert.Equal(suite.T(), "get_weather", toolCall.Function.Name)
	assert.NotEmpty(suite.T(), toolCall.ID)

	// Parse arguments
	var args map[string]interface{}
	err = json.Unmarshal([]byte(toolCall.Function.Arguments), &args)
	require.NoError(suite.T(), err)
	assert.Contains(suite.T(), args, "location")
	assert.Contains(suite.T(), args["location"], "Tokyo")
}

func (suite *E2ETestSuite) TestToolCallingWithResponse() {
	ctx := context.Background()

	// Define calculator tool
	calcTool, err := models.NewTool("calculate",
		"Perform basic arithmetic calculations",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"operation": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"add", "subtract", "multiply", "divide"},
					"description": "The arithmetic operation to perform",
				},
				"a": map[string]interface{}{
					"type":        "number",
					"description": "First number",
				},
				"b": map[string]interface{}{
					"type":        "number",
					"description": "Second number",
				},
			},
			"required": []string{"operation", "a", "b"},
		},
	)
	require.NoError(suite.T(), err)

	// Initial messages
	messages := []models.Message{
		models.NewTextMessage(models.RoleUser, "What is 25 multiplied by 4?"),
	}

	// First request
	resp1, err := suite.client.CreateChatCompletion(ctx, models.ChatCompletionRequest{
		Model:      "openai/gpt-4o-mini",
		Messages:   messages,
		Tools:      []models.Tool{*calcTool},
		ToolChoice: models.ToolChoiceAuto,
	})
	require.NoError(suite.T(), err)

	// Should have tool call
	require.Greater(suite.T(), len(resp1.Choices[0].Message.ToolCalls), 0)
	toolCall := resp1.Choices[0].Message.ToolCalls[0]

	// Parse arguments and execute tool
	var args struct {
		Operation string  `json:"operation"`
		A         float64 `json:"a"`
		B         float64 `json:"b"`
	}
	err = json.Unmarshal([]byte(toolCall.Function.Arguments), &args)
	require.NoError(suite.T(), err)

	// Execute the calculation
	var result float64
	switch args.Operation {
	case "multiply":
		result = args.A * args.B
	}
	assert.Equal(suite.T(), float64(100), result)

	// Add assistant message with tool call and tool response
	messages = append(messages, *resp1.Choices[0].Message)
	messages = append(messages, models.NewToolMessage(
		toolCall.ID,
		toolCall.Function.Name,
		"100",
	))

	// Second request to get final response
	resp2, err := suite.client.CreateChatCompletion(ctx, models.ChatCompletionRequest{
		Model:    "openai/gpt-4o-mini",
		Messages: messages,
	})
	require.NoError(suite.T(), err)

	// Should have final answer
	content, err := resp2.Choices[0].Message.GetTextContent()
	assert.NoError(suite.T(), err)
	assert.Contains(suite.T(), content, "100")
}

func (suite *E2ETestSuite) TestMultipleTools() {
	ctx := context.Background()

	// Define multiple tools
	tools := []models.Tool{}

	// Search tool
	searchTool, _ := models.NewTool("search_web",
		"Search the web for information",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "The search query",
				},
			},
			"required": []string{"query"},
		},
	)
	tools = append(tools, *searchTool)

	// Email tool
	emailTool, _ := models.NewTool("send_email",
		"Send an email",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"to": map[string]interface{}{
					"type":        "string",
					"description": "Recipient email address",
				},
				"subject": map[string]interface{}{
					"type":        "string",
					"description": "Email subject",
				},
				"body": map[string]interface{}{
					"type":        "string",
					"description": "Email body",
				},
			},
			"required": []string{"to", "subject", "body"},
		},
	)
	tools = append(tools, *emailTool)

	req := models.ChatCompletionRequest{
		Model: "openai/gpt-4o-mini",
		Messages: []models.Message{
			models.NewTextMessage(models.RoleUser, "Search for the latest AI news"),
		},
		Tools:      tools,
		ToolChoice: models.ToolChoiceAuto,
	}

	resp, err := suite.client.CreateChatCompletion(ctx, req)
	require.NoError(suite.T(), err)

	// Should call the search tool, not email
	if len(resp.Choices[0].Message.ToolCalls) > 0 {
		assert.Equal(suite.T(), "search_web", resp.Choices[0].Message.ToolCalls[0].Function.Name)
	}
}

func (suite *E2ETestSuite) TestToolAgent() {
	ctx := context.Background()

	// Create agent
	agent := openrouter.NewAgent(suite.client, "openai/gpt-4o-mini")

	// Define weather tool
	weatherTool, _ := models.NewTool("get_weather",
		"Get current weather for a location",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"location": map[string]interface{}{
					"type": "string",
				},
			},
			"required": []string{"location"},
		},
	)

	// Register tool implementation
	agent.RegisterToolFunc(*weatherTool, func(tc models.ToolCall) (string, error) {
		var args struct {
			Location string `json:"location"`
		}
		json.Unmarshal([]byte(tc.Function.Arguments), &args)

		// Mock weather response
		weather := map[string]interface{}{
			"location":    args.Location,
			"temperature": 22,
			"conditions":  "Sunny",
			"humidity":    65,
		}

		result, _ := json.Marshal(weather)
		return string(result), nil
	})

	// Run agent
	messages := []models.Message{
		models.NewTextMessage(models.RoleUser, "What's the weather in Paris?"),
	}

	finalMessages, err := agent.Run(ctx, messages, openrouter.RunOptions{
		Tools:         []models.Tool{*weatherTool},
		MaxIterations: 3,
	})

	require.NoError(suite.T(), err)
	assert.Greater(suite.T(), len(finalMessages), len(messages))

	// Should have tool call and response in messages
	hasToolCall := false
	hasToolResponse := false
	for _, msg := range finalMessages {
		if len(msg.ToolCalls) > 0 {
			hasToolCall = true
		}
		if msg.Role == models.RoleTool {
			hasToolResponse = true
		}
	}
	assert.True(suite.T(), hasToolCall)
	assert.True(suite.T(), hasToolResponse)

	// Final message should contain weather info
	lastMessage := finalMessages[len(finalMessages)-1]
	content, _ := lastMessage.GetTextContent()
	assert.Contains(suite.T(), content, "Paris")
	assert.Contains(suite.T(), content, "22")
}

func (suite *E2ETestSuite) TestToolChoiceSpecific() {
	ctx := context.Background()

	// Define tools
	calcTool, _ := models.NewTool("calculate", "Calculate", map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"expression": map[string]interface{}{"type": "string"},
		},
		"required": []string{"expression"},
	})

	weatherTool, _ := models.NewTool("get_weather", "Get weather", map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"location": map[string]interface{}{"type": "string"},
		},
		"required": []string{"location"},
	})

	// Force specific tool
	req := models.ChatCompletionRequest{
		Model: "openai/gpt-4o-mini",
		Messages: []models.Message{
			models.NewTextMessage(models.RoleUser, "Hello, how are you?"),
		},
		Tools:      []models.Tool{*calcTool, *weatherTool},
		ToolChoice: models.NewFunctionToolChoice("get_weather"),
	}

	resp, err := suite.client.CreateChatCompletion(ctx, req)
	require.NoError(suite.T(), err)

	// Should call the specified tool even though the message doesn't ask for weather
	if len(resp.Choices[0].Message.ToolCalls) > 0 {
		assert.Equal(suite.T(), "get_weather", resp.Choices[0].Message.ToolCalls[0].Function.Name)
	}
}
