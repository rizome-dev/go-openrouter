package openrouter

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rizome-dev/go-openrouter/pkg/models"
	"github.com/rizome-dev/go-openrouter/pkg/streaming"
)

// CompletionRequest represents a text completion request
type CompletionRequest struct {
	// Core parameters
	Model  string   `json:"model,omitempty"`
	Prompt string   `json:"prompt"`
	Models []string `json:"models,omitempty"`
	
	// Provider routing
	Provider *models.ProviderPreferences `json:"provider,omitempty"`
	
	// Response configuration
	Stream bool `json:"stream,omitempty"`
	
	// LLM Parameters
	MaxTokens         *int               `json:"max_tokens,omitempty"`
	Temperature       *float64           `json:"temperature,omitempty"`
	TopP              *float64           `json:"top_p,omitempty"`
	TopK              *int               `json:"top_k,omitempty"`
	FrequencyPenalty  *float64           `json:"frequency_penalty,omitempty"`
	PresencePenalty   *float64           `json:"presence_penalty,omitempty"`
	RepetitionPenalty *float64           `json:"repetition_penalty,omitempty"`
	Seed              *int               `json:"seed,omitempty"`
	Stop              []string           `json:"stop,omitempty"`
	LogitBias         map[string]float64 `json:"logit_bias,omitempty"`
	TopLogprobs       *int               `json:"top_logprobs,omitempty"`
	MinP              *float64           `json:"min_p,omitempty"`
	TopA              *float64           `json:"top_a,omitempty"`
	
	// OpenRouter-specific parameters
	Transforms []string `json:"transforms,omitempty"`
	User       string   `json:"user,omitempty"`
}

// CompletionResponse represents a text completion response
type CompletionResponse struct {
	ID      string             `json:"id"`
	Object  string             `json:"object"`
	Created int64              `json:"created"`
	Model   string             `json:"model"`
	Choices []CompletionChoice `json:"choices"`
	Usage   *models.Usage      `json:"usage,omitempty"`
}

// CompletionChoice represents a completion choice
type CompletionChoice struct {
	Index        int                  `json:"index"`
	Text         string               `json:"text,omitempty"`
	FinishReason string               `json:"finish_reason,omitempty"`
	Error        *models.ChoiceError  `json:"error,omitempty"`
}

// CreateCompletion creates a text completion
func (c *Client) CreateCompletion(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	// Ensure streaming is disabled for non-streaming endpoint
	req.Stream = false
	
	resp, err := c.doRequest(ctx, "POST", "/completions", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	var completionResp CompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&completionResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	
	return &completionResp, nil
}

// CreateCompletionStream creates a streaming text completion
func (c *Client) CreateCompletionStream(ctx context.Context, req CompletionRequest) (*streaming.CompletionStreamReader, error) {
	// Ensure streaming is enabled
	req.Stream = true
	
	resp, err := c.doRequest(ctx, "POST", "/completions", req)
	if err != nil {
		return nil, err
	}
	
	return streaming.NewCompletionStreamReader(resp.Body), nil
}