package openrouter

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rizome-dev/go-openrouter/pkg/errors"
	"github.com/rizome-dev/go-openrouter/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComprehensiveErrorHandling(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		response       interface{}
		expectedCode   errors.ErrorCode
		expectedMsg    string
		checkMetadata  bool
		metadataChecks func(t *testing.T, metadata map[string]interface{})
	}{
		{
			name:       "400 Bad Request - CORS",
			statusCode: 400,
			response: errors.ErrorResponse{
				Error: struct {
					Code     int                    `json:"code"`
					Message  string                 `json:"message"`
					Metadata map[string]interface{} `json:"metadata,omitempty"`
				}{
					Code:    400,
					Message: "CORS error: Origin not allowed",
				},
			},
			expectedCode: errors.ErrorCodeBadRequest,
			expectedMsg:  "CORS error: Origin not allowed",
		},
		{
			name:       "401 Unauthorized - Invalid API Key",
			statusCode: 401,
			response: errors.ErrorResponse{
				Error: struct {
					Code     int                    `json:"code"`
					Message  string                 `json:"message"`
					Metadata map[string]interface{} `json:"metadata,omitempty"`
				}{
					Code:    401,
					Message: "Invalid API key provided",
				},
			},
			expectedCode: errors.ErrorCodeUnauthorized,
			expectedMsg:  "Invalid API key provided",
		},
		{
			name:       "401 Unauthorized - OAuth Session Expired",
			statusCode: 401,
			response: errors.ErrorResponse{
				Error: struct {
					Code     int                    `json:"code"`
					Message  string                 `json:"message"`
					Metadata map[string]interface{} `json:"metadata,omitempty"`
				}{
					Code:    401,
					Message: "OAuth session expired",
					Metadata: map[string]interface{}{
						"oauth_error": "session_expired",
					},
				},
			},
			expectedCode:  errors.ErrorCodeUnauthorized,
			expectedMsg:   "OAuth session expired",
			checkMetadata: true,
			metadataChecks: func(t *testing.T, metadata map[string]interface{}) {
				assert.Equal(t, "session_expired", metadata["oauth_error"])
			},
		},
		{
			name:       "402 Payment Required - Insufficient Credits",
			statusCode: 402,
			response: errors.ErrorResponse{
				Error: struct {
					Code     int                    `json:"code"`
					Message  string                 `json:"message"`
					Metadata map[string]interface{} `json:"metadata,omitempty"`
				}{
					Code:    402,
					Message: "Your account has insufficient credits",
					Metadata: map[string]interface{}{
						"credits_required": 100.0,
						"credits_balance":  5.0,
					},
				},
			},
			expectedCode:  errors.ErrorCodeInsufficientCredits,
			expectedMsg:   "Your account has insufficient credits",
			checkMetadata: true,
			metadataChecks: func(t *testing.T, metadata map[string]interface{}) {
				assert.Equal(t, 100.0, metadata["credits_required"])
				assert.Equal(t, 5.0, metadata["credits_balance"])
			},
		},
		{
			name:       "403 Forbidden - Moderation",
			statusCode: 403,
			response: errors.ErrorResponse{
				Error: struct {
					Code     int                    `json:"code"`
					Message  string                 `json:"message"`
					Metadata map[string]interface{} `json:"metadata,omitempty"`
				}{
					Code:    403,
					Message: "Input flagged by moderation",
					Metadata: map[string]interface{}{
						"reasons":       []interface{}{"violence", "hate_speech"},
						"flagged_input": "This is the flagged...",
						"provider_name": "openai",
						"model_slug":    "gpt-4",
					},
				},
			},
			expectedCode:  errors.ErrorCodeForbidden,
			expectedMsg:   "Input flagged by moderation",
			checkMetadata: true,
			metadataChecks: func(t *testing.T, metadata map[string]interface{}) {
				reasons := metadata["reasons"].([]interface{})
				assert.Contains(t, reasons, "violence")
				assert.Contains(t, reasons, "hate_speech")
				assert.Equal(t, "openai", metadata["provider_name"])
			},
		},
		{
			name:       "408 Request Timeout",
			statusCode: 408,
			response: errors.ErrorResponse{
				Error: struct {
					Code     int                    `json:"code"`
					Message  string                 `json:"message"`
					Metadata map[string]interface{} `json:"metadata,omitempty"`
				}{
					Code:    408,
					Message: "Request timed out after 60 seconds",
				},
			},
			expectedCode: errors.ErrorCodeTimeout,
			expectedMsg:  "Request timed out after 60 seconds",
		},
		{
			name:       "429 Too Many Requests",
			statusCode: 429,
			response: errors.ErrorResponse{
				Error: struct {
					Code     int                    `json:"code"`
					Message  string                 `json:"message"`
					Metadata map[string]interface{} `json:"metadata,omitempty"`
				}{
					Code:    429,
					Message: "Rate limit exceeded. Please retry after 60 seconds",
					Metadata: map[string]interface{}{
						"retry_after": 60,
						"limit":       100,
						"remaining":   0,
					},
				},
			},
			expectedCode:  errors.ErrorCodeRateLimited,
			expectedMsg:   "Rate limit exceeded. Please retry after 60 seconds",
			checkMetadata: true,
			metadataChecks: func(t *testing.T, metadata map[string]interface{}) {
				assert.Equal(t, float64(60), metadata["retry_after"])
				assert.Equal(t, float64(100), metadata["limit"])
			},
		},
		{
			name:       "502 Bad Gateway - Provider Error",
			statusCode: 502,
			response: errors.ErrorResponse{
				Error: struct {
					Code     int                    `json:"code"`
					Message  string                 `json:"message"`
					Metadata map[string]interface{} `json:"metadata,omitempty"`
				}{
					Code:    502,
					Message: "Provider returned invalid response",
					Metadata: map[string]interface{}{
						"provider_name": "anthropic",
						"raw":           "upstream connect error",
					},
				},
			},
			expectedCode:  errors.ErrorCodeBadGateway,
			expectedMsg:   "Provider returned invalid response",
			checkMetadata: true,
			metadataChecks: func(t *testing.T, metadata map[string]interface{}) {
				assert.Equal(t, "anthropic", metadata["provider_name"])
				assert.Equal(t, "upstream connect error", metadata["raw"])
			},
		},
		{
			name:       "503 Service Unavailable",
			statusCode: 503,
			response: errors.ErrorResponse{
				Error: struct {
					Code     int                    `json:"code"`
					Message  string                 `json:"message"`
					Metadata map[string]interface{} `json:"metadata,omitempty"`
				}{
					Code:    503,
					Message: "No available model provider meets your routing requirements",
					Metadata: map[string]interface{}{
						"requested_model": "gpt-4",
						"requirements": map[string]interface{}{
							"max_cost":    0.01,
							"max_latency": 1000,
						},
					},
				},
			},
			expectedCode:  errors.ErrorCodeServiceUnavailable,
			expectedMsg:   "No available model provider meets your routing requirements",
			checkMetadata: true,
			metadataChecks: func(t *testing.T, metadata map[string]interface{}) {
				assert.Equal(t, "gpt-4", metadata["requested_model"])
				requirements := metadata["requirements"].(map[string]interface{})
				assert.Equal(t, 0.01, requirements["max_cost"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				json.NewEncoder(w).Encode(tt.response)
			}))
			defer server.Close()

			client := NewClient("test-api-key", WithBaseURL(server.URL))

			_, err := client.CreateChatCompletion(context.Background(), models.ChatCompletionRequest{
				Model: "test-model",
				Messages: []models.Message{
					models.NewTextMessage(models.RoleUser, "test"),
				},
			})

			require.Error(t, err)

			apiErr, ok := err.(*errors.APIError)
			require.True(t, ok, "Expected APIError type, got %T", err)
			assert.Equal(t, tt.expectedCode, apiErr.Code)
			assert.Equal(t, tt.expectedMsg, apiErr.Message)

			if tt.checkMetadata && tt.metadataChecks != nil {
				assert.NotNil(t, apiErr.Metadata)
				tt.metadataChecks(t, apiErr.Metadata)
			}
		})
	}
}

func TestStreamingErrorHandling(t *testing.T) {
	tests := []struct {
		name         string
		streamChunks []string
		expectError  bool
		errorCode    int
	}{
		{
			name: "Error in stream",
			streamChunks: []string{
				`{"choices":[{"delta":{"content":"Starting..."}}]}`,
				`{"error":{"code":500,"message":"Internal server error"}}`,
			},
			expectError: true,
			errorCode:   500,
		},
		{
			name: "Choice-level error",
			streamChunks: []string{
				`{"choices":[{"delta":{"content":"Some content"}}]}`,
				`{"choices":[{"delta":{},"error":{"code":403,"message":"Content filtered"}}]}`,
			},
			expectError: true,
			errorCode:   403,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/event-stream")
				flusher := w.(http.Flusher)

				for _, chunk := range tt.streamChunks {
					fmt.Fprintf(w, "data: %s\n\n", chunk)
					flusher.Flush()
				}

				fmt.Fprintf(w, "data: [DONE]\n\n")
				flusher.Flush()
			}))
			defer server.Close()

			client := NewClient("test-key", WithBaseURL(server.URL))
			stream, err := client.CreateChatCompletionStream(context.Background(), models.ChatCompletionRequest{
				Model:    "test-model",
				Messages: []models.Message{models.NewTextMessage(models.RoleUser, "Test")},
			})
			require.NoError(t, err)
			defer stream.Close()

			errorOccurred := false
			for {
				chunk, err := stream.Read()
				if err != nil {
					if err.Error() != "EOF" {
						errorOccurred = true
					}
					break
				}

				// Check for choice-level errors
				if len(chunk.Choices) > 0 && chunk.Choices[0].Error != nil {
					errorOccurred = true
					assert.Equal(t, tt.errorCode, chunk.Choices[0].Error.Code)
				}
			}

			if tt.expectError {
				assert.True(t, errorOccurred)
			}
		})
	}
}

func TestMalformedErrorResponse(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		response   string
	}{
		{
			name:       "Invalid JSON",
			statusCode: 500,
			response:   "Internal Server Error",
		},
		{
			name:       "Empty response",
			statusCode: 502,
			response:   "",
		},
		{
			name:       "HTML error page",
			statusCode: 503,
			response:   "<html><body>503 Service Unavailable</body></html>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				fmt.Fprint(w, tt.response)
			}))
			defer server.Close()

			client := NewClient("test-key", WithBaseURL(server.URL))

			_, err := client.CreateChatCompletion(context.Background(), models.ChatCompletionRequest{
				Model:    "test-model",
				Messages: []models.Message{models.NewTextMessage(models.RoleUser, "test")},
			})

			require.Error(t, err)
			// Should handle malformed error gracefully
			assert.Contains(t, err.Error(), "failed to parse error response")
		})
	}
}

func TestNoContentGenerated(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return 200 OK but with empty content
		resp := models.ChatCompletionResponse{
			ID:    "resp-123",
			Model: "test-model",
			Choices: []models.Choice{
				{
					Message: &models.Message{
						Role:    models.RoleAssistant,
						Content: json.RawMessage(`""`),
					},
					FinishReason: "stop",
				},
			},
			Usage: &models.Usage{
				PromptTokens:     100,
				CompletionTokens: 0,
				TotalTokens:      100,
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	resp, err := client.CreateChatCompletion(context.Background(), models.ChatCompletionRequest{
		Model:    "test-model",
		Messages: []models.Message{models.NewTextMessage(models.RoleUser, "Generate something")},
	})

	require.NoError(t, err)
	assert.NotNil(t, resp)

	// Check that content is empty
	content, err := resp.Choices[0].Message.GetTextContent()
	assert.NoError(t, err)
	assert.Empty(t, content)

	// But usage shows prompt was processed
	assert.Equal(t, 100, resp.Usage.PromptTokens)
	assert.Equal(t, 0, resp.Usage.CompletionTokens)
}

func TestErrorResponseHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set custom headers that might be useful for debugging
		w.Header().Set("X-Request-ID", "req-123")
		w.Header().Set("X-RateLimit-Limit", "100")
		w.Header().Set("X-RateLimit-Remaining", "0")
		w.Header().Set("X-RateLimit-Reset", "1640995200")

		w.WriteHeader(429)
		json.NewEncoder(w).Encode(errors.ErrorResponse{
			Error: struct {
				Code     int                    `json:"code"`
				Message  string                 `json:"message"`
				Metadata map[string]interface{} `json:"metadata,omitempty"`
			}{
				Code:    429,
				Message: "Rate limit exceeded",
			},
		})
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	_, err := client.CreateChatCompletion(context.Background(), models.ChatCompletionRequest{
		Model:    "test-model",
		Messages: []models.Message{models.NewTextMessage(models.RoleUser, "test")},
	})

	require.Error(t, err)
	apiErr, ok := err.(*errors.APIError)
	require.True(t, ok)
	assert.Equal(t, errors.ErrorCodeRateLimited, apiErr.Code)
}
