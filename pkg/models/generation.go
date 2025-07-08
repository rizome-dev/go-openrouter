package models

// GenerationResponse represents the response from the generation endpoint
type GenerationResponse struct {
	Data Generation `json:"data"`
}

// Generation represents metadata about a specific generation
type Generation struct {
	ID                string                 `json:"id"`
	Model             string                 `json:"model"`
	Object            string                 `json:"object"`
	Created           int64                  `json:"created"`
	Usage             interface{}            `json:"usage"`
	NativeTokenCounts NativeTokenCounts      `json:"native_token_counts"`
	Metrics           GenerationMetrics      `json:"metrics"`
	Provider          string                 `json:"provider"`
	Error             *GenerationError       `json:"error,omitempty"`
	Moderation        *ModerationInfo        `json:"moderation,omitempty"`
	Transforms        []string               `json:"transforms,omitempty"`
	Origin            interface{} `json:"origin,omitempty"`
}

// GenerationUsage represents token usage with costs
type GenerationUsage struct {
	PromptTokens     int     `json:"prompt_tokens"`
	CompletionTokens int     `json:"completion_tokens"`
	TotalTokens      int     `json:"total_tokens"`
	TotalCost        float64 `json:"total_cost"`
}

// NativeTokenCounts represents the actual token counts from the provider
type NativeTokenCounts struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// GenerationMetrics represents performance metrics
type GenerationMetrics struct {
	LatencyMS        int     `json:"latency_ms"`
	TokensPerSecond  float64 `json:"tokens_per_second"`
	TimeToFirstToken int     `json:"time_to_first_token_ms,omitempty"`
	StreamingLatency int     `json:"streaming_latency_ms,omitempty"`
}

// GenerationError represents an error that occurred during generation
type GenerationError struct {
	Code     int    `json:"code"`
	Message  string `json:"message"`
	Provider string `json:"provider,omitempty"`
}

// ModerationInfo represents content moderation information
type ModerationInfo struct {
	Flagged      bool     `json:"flagged"`
	Categories   []string `json:"categories,omitempty"`
	FlaggedInput string   `json:"flagged_input,omitempty"`
}
