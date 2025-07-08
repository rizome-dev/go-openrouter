package pkg

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rizome-dev/go-openrouter/pkg/models"
	"github.com/rizome-dev/go-openrouter/pkg/streaming"
)

// CreateCompletion creates a text completion using the legacy completions endpoint
func (c *Client) CreateCompletion(ctx context.Context, req models.ChatCompletionRequest) (*models.ChatCompletionResponse, error) {
	// Ensure streaming is disabled for non-streaming endpoint
	req.Stream = false

	// For legacy completions endpoint, use the prompt field instead of messages
	resp, err := c.doRequest(ctx, "POST", "/completions", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var completionResp models.ChatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&completionResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &completionResp, nil
}

// CreateCompletionStream creates a streaming text completion
func (c *Client) CreateCompletionStream(ctx context.Context, req models.ChatCompletionRequest) (*streaming.ChatCompletionStreamReader, error) {
	// Ensure streaming is enabled
	req.Stream = true

	resp, err := c.doRequest(ctx, "POST", "/completions", req)
	if err != nil {
		return nil, err
	}

	return streaming.NewChatCompletionStreamReader(resp.Body), nil
}
