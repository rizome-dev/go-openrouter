package e2e

import (
	"context"
	"io"
	"strings"
	"time"

	"github.com/rizome-dev/go-openrouter/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (suite *E2ETestSuite) TestBasicStreaming() {
	ctx := context.Background()

	req := models.ChatCompletionRequest{
		Model: "meta-llama/llama-3.2-1b-instruct:free",
		Messages: []models.Message{
			models.NewTextMessage(models.RoleUser, "Count from 1 to 5"),
		},
		MaxTokens:   intPtr(50),
		Temperature: float64Ptr(0.0),
		Stream:      true,
	}

	stream, err := suite.client.CreateChatCompletionStream(ctx, req)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), stream)
	defer stream.Close()

	var fullContent strings.Builder
	chunkCount := 0

	for {
		chunk, err := stream.Read()
		if err == io.EOF {
			break
		}
		require.NoError(suite.T(), err)

		chunkCount++

		if chunk.Choices != nil && len(chunk.Choices) > 0 && chunk.Choices[0].Delta != nil {
			content, _ := chunk.Choices[0].Delta.GetTextContent()
			fullContent.WriteString(content)
		}
	}

	// Should have received multiple chunks
	assert.Greater(suite.T(), chunkCount, 1)

	// Should have received content
	finalContent := fullContent.String()
	assert.NotEmpty(suite.T(), finalContent)

	// Content should contain numbers
	assert.Contains(suite.T(), finalContent, "1")
	assert.Contains(suite.T(), finalContent, "2")
}

func (suite *E2ETestSuite) TestStreamingWithCancel() {
	ctx, cancel := context.WithCancel(context.Background())

	req := models.ChatCompletionRequest{
		Model: "meta-llama/llama-3.2-1b-instruct:free",
		Messages: []models.Message{
			models.NewTextMessage(models.RoleUser, "Write a very long story about space exploration"),
		},
		MaxTokens: intPtr(500),
		Stream:    true,
	}

	stream, err := suite.client.CreateChatCompletionStream(ctx, req)
	require.NoError(suite.T(), err)
	defer stream.Close()

	chunkCount := 0

	// Read a few chunks then cancel
	for i := 0; i < 3; i++ {
		chunk, err := stream.Read()
		if err == io.EOF {
			break
		}
		require.NoError(suite.T(), err)
		chunkCount++

		if chunk.Choices != nil && len(chunk.Choices) > 0 {
			// Got a chunk
		}
	}

	// Cancel the context
	cancel()

	// Next read should return context canceled error
	_, err = stream.Read()
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "context canceled")
}

func (suite *E2ETestSuite) TestStreamingToolCalls() {
	ctx := context.Background()

	// Define a simple tool
	tool, err := models.NewTool("get_current_time",
		"Get the current time",
		map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
			"required":   []string{},
		},
	)
	require.NoError(suite.T(), err)

	req := models.ChatCompletionRequest{
		Model: "openai/gpt-4o-mini",
		Messages: []models.Message{
			models.NewTextMessage(models.RoleUser, "What time is it?"),
		},
		Tools:      []models.Tool{*tool},
		ToolChoice: models.ToolChoiceAuto,
		Stream:     true,
	}

	stream, err := suite.client.CreateChatCompletionStream(ctx, req)
	require.NoError(suite.T(), err)
	defer stream.Close()

	var toolCalls []models.ToolCall
	hasToolCall := false

	for {
		chunk, err := stream.Read()
		if err == io.EOF {
			break
		}
		require.NoError(suite.T(), err)

		if chunk.Choices != nil && len(chunk.Choices) > 0 {
			if chunk.Choices[0].Delta != nil && chunk.Choices[0].Delta.ToolCalls != nil {
				hasToolCall = true
				// Accumulate tool call chunks
				for _, tc := range chunk.Choices[0].Delta.ToolCalls {
					if tc.ID != "" {
						toolCalls = append(toolCalls, tc)
					} else if len(toolCalls) > 0 {
						// Append to existing tool call
						toolCalls[len(toolCalls)-1].Function.Arguments += tc.Function.Arguments
					}
				}
			}
		}
	}

	// Should have detected tool call
	assert.True(suite.T(), hasToolCall)
}

func (suite *E2ETestSuite) TestStreamingTimeout() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req := models.ChatCompletionRequest{
		Model: "meta-llama/llama-3.2-1b-instruct:free",
		Messages: []models.Message{
			models.NewTextMessage(models.RoleUser, "Write an extremely detailed essay about the history of computing"),
		},
		MaxTokens: intPtr(1000),
		Stream:    true,
	}

	stream, err := suite.client.CreateChatCompletionStream(ctx, req)
	require.NoError(suite.T(), err)
	defer stream.Close()

	startTime := time.Now()

	for {
		_, err := stream.Read()
		if err != nil {
			if err == io.EOF {
				// Completed before timeout
				break
			}
			// Should timeout
			assert.Contains(suite.T(), err.Error(), "deadline exceeded")
			break
		}

		// Ensure we're not stuck forever
		if time.Since(startTime) > 5*time.Second {
			suite.T().Fatal("Stream did not timeout as expected")
		}
	}
}
