package models

import (
	"encoding/json"
)

// ChatCompletionRequest represents a request to the chat completions endpoint
type ChatCompletionRequest struct {
	// Either messages or prompt is required
	Messages []Message `json:"messages,omitempty"`
	Prompt   string    `json:"prompt,omitempty"`
	
	// Model selection
	Model  string   `json:"model,omitempty"`
	Models []string `json:"models,omitempty"`
	
	// Provider routing
	Provider *ProviderPreferences `json:"provider,omitempty"`
	
	// Response configuration
	ResponseFormat *ResponseFormat `json:"response_format,omitempty"`
	Stream         bool            `json:"stream,omitempty"`
	
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
	
	// Tool calling
	Tools      []Tool     `json:"tools,omitempty"`
	ToolChoice ToolChoice `json:"tool_choice,omitempty"`
	
	// Predicted outputs for latency optimization
	Prediction *Prediction `json:"prediction,omitempty"`
	
	// OpenRouter-specific parameters
	Transforms []string `json:"transforms,omitempty"`
	Route      string   `json:"route,omitempty"`
	User       string   `json:"user,omitempty"`
	
	// Plugins
	Plugins []Plugin `json:"plugins,omitempty"`
	
	// Web search options (for native web search models)
	WebSearchOptions *WebSearchOptions `json:"web_search_options,omitempty"`
}

// ResponseFormat represents the desired response format
type ResponseFormat struct {
	Type       string      `json:"type"`
	JSONSchema *JSONSchema `json:"json_schema,omitempty"`
}

// JSONSchema represents a JSON schema for structured outputs
type JSONSchema struct {
	Name   string          `json:"name"`
	Strict bool            `json:"strict"`
	Schema json.RawMessage `json:"schema"`
}

// Prediction represents predicted output for latency optimization
type Prediction struct {
	Type    string `json:"type"`
	Content string `json:"content"`
}

// WebSearchOptions represents options for native web search
type WebSearchOptions struct {
	SearchContextSize string `json:"search_context_size,omitempty"` // "low", "medium", "high"
}

// ChatCompletionResponse represents a response from the chat completions endpoint
type ChatCompletionResponse struct {
	ID                string       `json:"id"`
	Object            string       `json:"object"`
	Created           int64        `json:"created"`
	Model             string       `json:"model"`
	Choices           []Choice     `json:"choices"`
	Usage             *Usage       `json:"usage,omitempty"`
	SystemFingerprint string       `json:"system_fingerprint,omitempty"`
}

// Choice represents a completion choice
type Choice struct {
	Index int `json:"index"`
	
	// For non-streaming responses
	Message *Message `json:"message,omitempty"`
	
	// For streaming responses
	Delta *Message `json:"delta,omitempty"`
	
	// Finish reasons
	FinishReason       string `json:"finish_reason,omitempty"`
	NativeFinishReason string `json:"native_finish_reason,omitempty"`
	
	// Error if any
	Error *ChoiceError `json:"error,omitempty"`
	
	// Log probabilities
	Logprobs *LogProbs `json:"logprobs,omitempty"`
}

// ChoiceError represents an error in a choice
type ChoiceError struct {
	Code     int                    `json:"code"`
	Message  string                 `json:"message"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// Usage represents token usage information
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// LogProbs represents log probability information
type LogProbs struct {
	Content []LogProbContent `json:"content"`
}

// LogProbContent represents log probability for a token
type LogProbContent struct {
	Token       string              `json:"token"`
	Logprob     float64             `json:"logprob"`
	Bytes       []int               `json:"bytes,omitempty"`
	TopLogprobs []TopLogProbContent `json:"top_logprobs,omitempty"`
}

// TopLogProbContent represents top log probabilities for alternative tokens
type TopLogProbContent struct {
	Token   string  `json:"token"`
	Logprob float64 `json:"logprob"`
	Bytes   []int   `json:"bytes,omitempty"`
}