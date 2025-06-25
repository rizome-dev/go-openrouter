package errors

import (
	"fmt"
)

// ErrorCode represents OpenRouter API error codes
type ErrorCode int

const (
	// ErrorCodeBadRequest indicates invalid or missing params, CORS
	ErrorCodeBadRequest ErrorCode = 400
	
	// ErrorCodeUnauthorized indicates invalid credentials
	ErrorCodeUnauthorized ErrorCode = 401
	
	// ErrorCodeInsufficientCredits indicates insufficient credits
	ErrorCodeInsufficientCredits ErrorCode = 402
	
	// ErrorCodeForbidden indicates moderation flag
	ErrorCodeForbidden ErrorCode = 403
	
	// ErrorCodeTimeout indicates request timeout
	ErrorCodeTimeout ErrorCode = 408
	
	// ErrorCodeRateLimited indicates rate limiting
	ErrorCodeRateLimited ErrorCode = 429
	
	// ErrorCodeModelDown indicates model is down or invalid response
	ErrorCodeModelDown ErrorCode = 502
	
	// ErrorCodeNoAvailableModel indicates no available model provider
	ErrorCodeNoAvailableModel ErrorCode = 503
)

// ErrorResponse represents an error response from the OpenRouter API
type ErrorResponse struct {
	Error struct {
		Code     int                    `json:"code"`
		Message  string                 `json:"message"`
		Metadata map[string]interface{} `json:"metadata,omitempty"`
	} `json:"error"`
}

// ToError converts the ErrorResponse to a Go error
func (e ErrorResponse) ToError() error {
	return &APIError{
		Code:     ErrorCode(e.Error.Code),
		Message:  e.Error.Message,
		Metadata: e.Error.Metadata,
	}
}

// APIError represents an error from the OpenRouter API
type APIError struct {
	Code     ErrorCode
	Message  string
	Metadata map[string]interface{}
}

// Error implements the error interface
func (e *APIError) Error() string {
	return fmt.Sprintf("openrouter error %d: %s", e.Code, e.Message)
}

// IsModerationError returns true if this is a moderation error
func (e *APIError) IsModerationError() bool {
	return e.Code == ErrorCodeForbidden
}

// GetModerationMetadata returns moderation-specific metadata if available
func (e *APIError) GetModerationMetadata() (*ModerationErrorMetadata, bool) {
	if !e.IsModerationError() || e.Metadata == nil {
		return nil, false
	}
	
	metadata := &ModerationErrorMetadata{}
	
	if reasons, ok := e.Metadata["reasons"].([]interface{}); ok {
		for _, r := range reasons {
			if str, ok := r.(string); ok {
				metadata.Reasons = append(metadata.Reasons, str)
			}
		}
	}
	
	if flagged, ok := e.Metadata["flagged_input"].(string); ok {
		metadata.FlaggedInput = flagged
	}
	
	if provider, ok := e.Metadata["provider_name"].(string); ok {
		metadata.ProviderName = provider
	}
	
	if model, ok := e.Metadata["model_slug"].(string); ok {
		metadata.ModelSlug = model
	}
	
	return metadata, true
}

// GetProviderMetadata returns provider-specific metadata if available
func (e *APIError) GetProviderMetadata() (*ProviderErrorMetadata, bool) {
	if e.Metadata == nil {
		return nil, false
	}
	
	metadata := &ProviderErrorMetadata{}
	
	if provider, ok := e.Metadata["provider_name"].(string); ok {
		metadata.ProviderName = provider
	}
	
	if raw, ok := e.Metadata["raw"]; ok {
		metadata.Raw = raw
	}
	
	if metadata.ProviderName == "" {
		return nil, false
	}
	
	return metadata, true
}

// ModerationErrorMetadata contains moderation-specific error metadata
type ModerationErrorMetadata struct {
	Reasons      []string `json:"reasons"`
	FlaggedInput string   `json:"flagged_input"`
	ProviderName string   `json:"provider_name"`
	ModelSlug    string   `json:"model_slug"`
}

// ProviderErrorMetadata contains provider-specific error metadata
type ProviderErrorMetadata struct {
	ProviderName string      `json:"provider_name"`
	Raw          interface{} `json:"raw"`
}