package openrouter

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rizome-dev/go-openrouter/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToolRegistry(t *testing.T) {
	registry := NewToolRegistry()

	// Test function registration
	called := false
	registry.RegisterFunc("test_tool", func(tc models.ToolCall) (string, error) {
		called = true
		assert.Equal(t, "test_tool", tc.Function.Name)
		return "tool result", nil
	})

	// Execute tool
	result, err := registry.Execute(models.ToolCall{
		ID: "call-123",
		Function: models.FunctionCall{
			Name:      "test_tool",
			Arguments: `{"arg": "value"}`,
		},
	})

	assert.NoError(t, err)
	assert.Equal(t, "tool result", result)
	assert.True(t, called)

	// Test non-existent tool
	_, err = registry.Execute(models.ToolCall{
		Function: models.FunctionCall{
			Name: "non_existent",
		},
	})
	assert.Error(t, err)
}

func TestAgent(t *testing.T) {
	// Create mock server
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++

		if callCount == 1 {
			// First call - return tool call
			resp := models.ChatCompletionResponse{
				ID:    "resp-1",
				Model: "test-model",
				Choices: []models.Choice{
					{
						Message: &models.Message{
							Role: models.RoleAssistant,
							ToolCalls: []models.ToolCall{
								{
									ID:   "call-1",
									Type: "function",
									Function: models.FunctionCall{
										Name:      "get_weather",
										Arguments: `{"location": "Tokyo"}`,
									},
								},
							},
						},
					},
				},
			}
			json.NewEncoder(w).Encode(resp)
		} else {
			// Second call - return final response
			content, _ := json.Marshal("The weather in Tokyo is sunny and 22°C")
			resp := models.ChatCompletionResponse{
				ID:    "resp-2",
				Model: "test-model",
				Choices: []models.Choice{
					{
						Message: &models.Message{
							Role:    models.RoleAssistant,
							Content: content,
						},
					},
				},
			}
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer server.Close()

	// Create client and agent
	client := NewClient("test-key", WithBaseURL(server.URL))
	agent := NewAgent(client, "test-model")

	// Register tool
	weatherTool, _ := models.NewTool("get_weather",
		"Get weather for a location",
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

	agent.RegisterToolFunc(*weatherTool, func(tc models.ToolCall) (string, error) {
		return "Sunny, 22°C", nil
	})

	// Run agent
	messages := []models.Message{
		models.NewTextMessage(models.RoleUser, "What's the weather in Tokyo?"),
	}

	finalMessages, err := agent.Run(context.Background(), messages, RunOptions{
		Tools:         []models.Tool{*weatherTool},
		MaxIterations: 5,
	})

	require.NoError(t, err)
	assert.Equal(t, 2, callCount)
	assert.Greater(t, len(finalMessages), len(messages))

	// Verify tool was called
	hasToolCall := false
	hasToolResult := false
	for _, msg := range finalMessages {
		if len(msg.ToolCalls) > 0 {
			hasToolCall = true
		}
		if msg.Role == models.RoleTool {
			hasToolResult = true
		}
	}
	assert.True(t, hasToolCall)
	assert.True(t, hasToolResult)
}

func TestToolExecutor(t *testing.T) {
	// Test ToolExecutorFunc
	executed := false
	executor := ToolExecutorFunc(func(tc models.ToolCall) (string, error) {
		executed = true
		var args map[string]interface{}
		json.Unmarshal([]byte(tc.Function.Arguments), &args)
		return args["message"].(string), nil
	})

	result, err := executor.Execute(models.ToolCall{
		Function: models.FunctionCall{
			Arguments: `{"message": "hello"}`,
		},
	})

	assert.NoError(t, err)
	assert.Equal(t, "hello", result)
	assert.True(t, executed)
}
