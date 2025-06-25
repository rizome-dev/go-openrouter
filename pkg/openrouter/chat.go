package openrouter

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/rizome-dev/openroutergo/pkg/models"
	"github.com/rizome-dev/openroutergo/pkg/streaming"
)

// CreateChatCompletion creates a chat completion
func (c *Client) CreateChatCompletion(ctx context.Context, req models.ChatCompletionRequest) (*models.ChatCompletionResponse, error) {
	// Ensure streaming is disabled for non-streaming endpoint
	req.Stream = false
	
	resp, err := c.doRequest(ctx, "POST", "/chat/completions", req)
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

// CreateChatCompletionStream creates a streaming chat completion
func (c *Client) CreateChatCompletionStream(ctx context.Context, req models.ChatCompletionRequest) (*streaming.ChatCompletionStreamReader, error) {
	// Ensure streaming is enabled
	req.Stream = true
	
	resp, err := c.doRequest(ctx, "POST", "/chat/completions", req)
	if err != nil {
		return nil, err
	}
	
	return streaming.NewChatCompletionStreamReader(resp.Body), nil
}