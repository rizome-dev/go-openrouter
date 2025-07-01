package models

// ModelsResponse represents the response from the models endpoint
type ModelsResponse struct {
	Data []Model `json:"data"`
}

// Model represents an available model
type Model struct {
	ID                  string       `json:"id"`
	Name                string       `json:"name"`
	CreatedAt           int64        `json:"created"`
	Description         string       `json:"description,omitempty"`
	ContextLength       int          `json:"context_length"`
	Pricing             Pricing      `json:"pricing"`
	TopProvider         TopProvider  `json:"top_provider,omitempty"`
	SupportedParams     []string     `json:"supported_parameters,omitempty"`
	MaxCompletionTokens int          `json:"max_completion_tokens,omitempty"`
	Architecture        Architecture `json:"architecture,omitempty"`
}

// Pricing represents the pricing information for a model
type Pricing struct {
	Prompt     string `json:"prompt"`            // Price per million prompt tokens
	Completion string `json:"completion"`        // Price per million completion tokens
	Request    string `json:"request,omitempty"` // Price per request
	Image      string `json:"image,omitempty"`   // Price per image
}

// TopProvider represents the top provider for a model
type TopProvider struct {
	MaxCompletionTokens int    `json:"max_completion_tokens,omitempty"`
	IsModerated         bool   `json:"is_moderated,omitempty"`
	Name                string `json:"name,omitempty"`
}

// Architecture represents model architecture details
type Architecture struct {
	Modality     string `json:"modality,omitempty"`
	Tokenizer    string `json:"tokenizer,omitempty"`
	InstructType string `json:"instruct_type,omitempty"`
}
