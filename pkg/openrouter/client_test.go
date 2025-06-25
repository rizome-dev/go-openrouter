package openrouter

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rizome-dev/openroutergo/pkg/errors"
	"github.com/rizome-dev/openroutergo/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	client := NewClient("test-api-key")
	assert.NotNil(t, client)
	assert.Equal(t, "test-api-key", client.apiKey)
	assert.Equal(t, DefaultBaseURL, client.baseURL)
}

func TestClientOptions(t *testing.T) {
	client := NewClient("test-api-key",
		WithBaseURL("https://custom.api.com"),
		WithTimeout(5*time.Second),
		WithHTTPReferer("https://myapp.com"),
		WithXTitle("My App"),
		WithUserAgent("MyApp/1.0"),
	)

	assert.Equal(t, "https://custom.api.com", client.baseURL)
	assert.Equal(t, 5*time.Second, client.httpClient.Timeout)
	assert.Equal(t, "https://myapp.com", client.httpReferer)
	assert.Equal(t, "My App", client.xTitle)
	assert.Equal(t, "MyApp/1.0", client.userAgent)
}

func TestCreateChatCompletion(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/api/v1/chat/completions", r.URL.Path)
		assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		// Verify request body
		var req models.ChatCompletionRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)
		assert.Equal(t, "openai/gpt-3.5-turbo", req.Model)
		assert.Equal(t, 1, len(req.Messages))
		assert.False(t, req.Stream)

		// Send response
		resp := models.ChatCompletionResponse{
			ID:      "chatcmpl-123",
			Object:  "chat.completion",
			Created: time.Now().Unix(),
			Model:   "openai/gpt-3.5-turbo",
			Choices: []models.Choice{
				{
					Index: 0,
					Message: &models.Message{
						Role:    models.RoleAssistant,
						Content: json.RawMessage(`"42"`),
					},
					FinishReason: "stop",
				},
			},
			Usage: &models.Usage{
				PromptTokens:     10,
				CompletionTokens: 5,
				TotalTokens:      15,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Create client
	client := NewClient("test-api-key", WithBaseURL(server.URL))

	// Make request
	resp, err := client.CreateChatCompletion(context.Background(), models.ChatCompletionRequest{
		Model: "openai/gpt-3.5-turbo",
		Messages: []models.Message{
			models.NewTextMessage(models.RoleUser, "What is the meaning of life?"),
		},
	})

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "chatcmpl-123", resp.ID)
	assert.Equal(t, 1, len(resp.Choices))
	
	content, err := resp.Choices[0].Message.GetTextContent()
	assert.NoError(t, err)
	assert.Equal(t, "42", content)
}

func TestErrorHandling(t *testing.T) {
	tests := []struct {
		name         string
		statusCode   int
		response     interface{}
		expectedCode errors.ErrorCode
		expectedMsg  string
	}{
		{
			name:       "Bad Request",
			statusCode: 400,
			response: errors.ErrorResponse{
				Error: struct {
					Code     int                    `json:"code"`
					Message  string                 `json:"message"`
					Metadata map[string]interface{} `json:"metadata,omitempty"`
				}{
					Code:    400,
					Message: "Invalid request parameters",
				},
			},
			expectedCode: errors.ErrorCodeBadRequest,
			expectedMsg:  "Invalid request parameters",
		},
		{
			name:       "Rate Limited",
			statusCode: 429,
			response: errors.ErrorResponse{
				Error: struct {
					Code     int                    `json:"code"`
					Message  string                 `json:"message"`
					Metadata map[string]interface{} `json:"metadata,omitempty"`
				}{
					Code:    429,
					Message: "Rate limit exceeded",
				},
			},
			expectedCode: errors.ErrorCodeRateLimited,
			expectedMsg:  "Rate limit exceeded",
		},
		{
			name:       "Insufficient Credits",
			statusCode: 402,
			response: errors.ErrorResponse{
				Error: struct {
					Code     int                    `json:"code"`
					Message  string                 `json:"message"`
					Metadata map[string]interface{} `json:"metadata,omitempty"`
				}{
					Code:    402,
					Message: "Insufficient credits",
				},
			},
			expectedCode: errors.ErrorCodeInsufficientCredits,
			expectedMsg:  "Insufficient credits",
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
			require.True(t, ok)
			assert.Equal(t, tt.expectedCode, apiErr.Code)
			assert.Equal(t, tt.expectedMsg, apiErr.Message)
		})
	}
}

func TestListModels(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/api/v1/models", r.URL.Path)

		resp := models.ModelsResponse{
			Data: []models.Model{
				{
					ID:            "openai/gpt-4",
					Name:          "GPT-4",
					CreatedAt:     time.Now().Unix(),
					ContextLength: 8192,
					Pricing: models.Pricing{
						Prompt:     "0.03",
						Completion: "0.06",
					},
				},
				{
					ID:            "anthropic/claude-3-opus",
					Name:          "Claude 3 Opus",
					CreatedAt:     time.Now().Unix(),
					ContextLength: 200000,
					Pricing: models.Pricing{
						Prompt:     "0.015",
						Completion: "0.075",
					},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))

	resp, err := client.ListModels(context.Background(), nil)
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, 2, len(resp.Data))
	assert.Equal(t, "openai/gpt-4", resp.Data[0].ID)
	assert.Equal(t, "anthropic/claude-3-opus", resp.Data[1].ID)
}

func TestGetGeneration(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/api/v1/generation", r.URL.Path)
		assert.Equal(t, "gen-123", r.URL.Query().Get("id"))

		resp := models.GenerationResponse{
			Data: models.Generation{
				ID:       "gen-123",
				Model:    "openai/gpt-4",
				Provider: "openai",
				Created:  time.Now().Unix(),
				Usage: models.GenerationUsage{
					PromptTokens:     100,
					CompletionTokens: 50,
					TotalTokens:      150,
					TotalCost:        0.0045,
				},
				NativeTokenCounts: models.NativeTokenCounts{
					PromptTokens:     98,
					CompletionTokens: 49,
					TotalTokens:      147,
				},
				Metrics: models.GenerationMetrics{
					LatencyMS:       1234,
					TokensPerSecond: 40.5,
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))

	resp, err := client.GetGeneration(context.Background(), "gen-123")
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "gen-123", resp.Data.ID)
	assert.Equal(t, 0.0045, resp.Data.Usage.TotalCost)
}