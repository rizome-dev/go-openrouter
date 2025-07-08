package openrouter

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rizome-dev/go-openrouter/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProviderRouting(t *testing.T) {
	tests := []struct {
		name             string
		providerPrefs    *models.ProviderPreferences
		expectedProvider string
		verifyRequest    func(t *testing.T, req models.ChatCompletionRequest)
	}{
		{
			name: "Allow specific providers",
			providerPrefs: &models.ProviderPreferences{
				AllowFallbacks:    boolPtr(true),
				RequireParameters: boolPtr(true),
				DataCollection:    models.DataCollectionDeny,
				Order:             []string{"openai", "anthropic"},
			},
			expectedProvider: "openai",
			verifyRequest: func(t *testing.T, req models.ChatCompletionRequest) {
				assert.NotNil(t, req.Provider)
				assert.NotNil(t, req.Provider.AllowFallbacks)
				assert.True(t, *req.Provider.AllowFallbacks)
				assert.NotNil(t, req.Provider.RequireParameters)
				assert.True(t, *req.Provider.RequireParameters)
				assert.Equal(t, models.DataCollectionDeny, req.Provider.DataCollection)
			},
		},
		{
			name: "Block specific providers",
			providerPrefs: &models.ProviderPreferences{
				Ignore:        []string{"anthropic", "google"},
				Quantizations: []models.QuantizationLevel{models.QuantizationBF16},
			},
			expectedProvider: "openai",
			verifyRequest: func(t *testing.T, req models.ChatCompletionRequest) {
				assert.NotNil(t, req.Provider)
				assert.Contains(t, req.Provider.Ignore, "anthropic")
				assert.Contains(t, req.Provider.Quantizations, models.QuantizationBF16)
			},
		},
		{
			name: "Ignore specific providers",
			providerPrefs: &models.ProviderPreferences{
				Ignore: []string{"azure"},
			},
			expectedProvider: "openai",
			verifyRequest: func(t *testing.T, req models.ChatCompletionRequest) {
				assert.NotNil(t, req.Provider)
				assert.Contains(t, req.Provider.Ignore, "azure")
			},
		},
		{
			name: "Cost constraints",
			providerPrefs: &models.ProviderPreferences{
				MaxPrice: &models.MaxPrice{
					Prompt:     0.01,
					Completion: 0.01,
				},
			},
			expectedProvider: "cheap-provider",
			verifyRequest: func(t *testing.T, req models.ChatCompletionRequest) {
				assert.NotNil(t, req.Provider)
				assert.NotNil(t, req.Provider.MaxPrice)
				assert.Equal(t, 0.01, req.Provider.MaxPrice.Prompt)
				assert.Equal(t, 0.01, req.Provider.MaxPrice.Completion)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var req models.ChatCompletionRequest
				err := json.NewDecoder(r.Body).Decode(&req)
				require.NoError(t, err)

				// Verify provider preferences
				tt.verifyRequest(t, req)

				resp := models.ChatCompletionResponse{
					ID:    "resp-123",
					Model: req.Model,
					Choices: []models.Choice{
						{
							Message: &models.Message{
								Role:    models.RoleAssistant,
								Content: json.RawMessage(`"Response from provider"`),
							},
						},
					},
				}
				json.NewEncoder(w).Encode(resp)
			}))
			defer server.Close()

			client := NewClient("test-key", WithBaseURL(server.URL))

			req := models.ChatCompletionRequest{
				Model: "openai/gpt-4",
				Messages: []models.Message{
					models.NewTextMessage(models.RoleUser, "Test"),
				},
				Provider: tt.providerPrefs,
			}

			resp, err := client.CreateChatCompletion(context.Background(), req)
			require.NoError(t, err)
			assert.NotNil(t, resp)
		})
	}
}

func TestModelRouting(t *testing.T) {
	tests := []struct {
		name          string
		models        []string
		route         string
		expectedModel string
	}{
		{
			name:          "Multiple models with fallback",
			models:        []string{"openai/gpt-4", "anthropic/claude-3-opus", "google/gemini-pro"},
			route:         "fallback",
			expectedModel: "openai/gpt-4",
		},
		{
			name:          "Single model",
			models:        []string{"openai/gpt-3.5-turbo"},
			expectedModel: "openai/gpt-3.5-turbo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var req models.ChatCompletionRequest
				err := json.NewDecoder(r.Body).Decode(&req)
				require.NoError(t, err)

				// Verify model routing
				if len(tt.models) > 1 {
					assert.Equal(t, tt.models, req.Models)
					assert.Equal(t, tt.route, req.Route)
				}

				resp := models.ChatCompletionResponse{
					ID:    "resp-123",
					Model: tt.expectedModel,
					Choices: []models.Choice{
						{
							Message: &models.Message{
								Role:    models.RoleAssistant,
								Content: json.RawMessage(`"Response"`),
							},
						},
					},
				}
				json.NewEncoder(w).Encode(resp)
			}))
			defer server.Close()

			client := NewClient("test-key", WithBaseURL(server.URL))

			req := models.ChatCompletionRequest{
				Model: tt.models[0], // Primary model
				Messages: []models.Message{
					models.NewTextMessage(models.RoleUser, "Test"),
				},
			}

			if len(tt.models) > 1 {
				req.Models = tt.models
				req.Route = tt.route
			}

			resp, err := client.CreateChatCompletion(context.Background(), req)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedModel, resp.Model)
		})
	}
}

func TestTransforms(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req models.ChatCompletionRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)

		// Verify transforms
		assert.Contains(t, req.Transforms, "middle-out")

		resp := models.ChatCompletionResponse{
			ID:    "resp-123",
			Model: req.Model,
			Choices: []models.Choice{
				{
					Message: &models.Message{
						Role:    models.RoleAssistant,
						Content: json.RawMessage(`"Transformed response"`),
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))

	req := models.ChatCompletionRequest{
		Model: "openai/gpt-4",
		Messages: []models.Message{
			models.NewTextMessage(models.RoleUser, "Apply transforms"),
		},
		Transforms: []string{"middle-out"},
	}

	resp, err := client.CreateChatCompletion(context.Background(), req)
	require.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestAssistantPrefill(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req models.ChatCompletionRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)

		// Verify assistant prefill
		assert.Equal(t, 2, len(req.Messages))
		lastMsg := req.Messages[len(req.Messages)-1]
		assert.Equal(t, models.RoleAssistant, lastMsg.Role)

		content, _ := lastMsg.GetTextContent()
		assert.Equal(t, "I think the answer is", content)

		// Complete the response
		resp := models.ChatCompletionResponse{
			ID:    "resp-123",
			Model: req.Model,
			Choices: []models.Choice{
				{
					Message: &models.Message{
						Role:    models.RoleAssistant,
						Content: json.RawMessage(`"I think the answer is 42, based on my calculations."`),
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))

	messages := []models.Message{
		models.NewTextMessage(models.RoleUser, "What is the meaning of life?"),
		models.NewTextMessage(models.RoleAssistant, "I think the answer is"),
	}

	resp, err := client.CreateChatCompletion(context.Background(), models.ChatCompletionRequest{
		Model:    "openai/gpt-4",
		Messages: messages,
	})

	require.NoError(t, err)
	content, _ := resp.Choices[0].Message.GetTextContent()
	assert.Contains(t, content, "42")
}

func TestPredictedOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req models.ChatCompletionRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)

		// Verify prediction
		assert.NotNil(t, req.Prediction)
		assert.Equal(t, "content", req.Prediction.Type)
		assert.Equal(t, "The weather today is sunny with", req.Prediction.Content)

		resp := models.ChatCompletionResponse{
			ID:    "resp-123",
			Model: req.Model,
			Choices: []models.Choice{
				{
					Message: &models.Message{
						Role:    models.RoleAssistant,
						Content: json.RawMessage(`"The weather today is sunny with clear skies and a temperature of 22°C."`),
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))

	req := models.ChatCompletionRequest{
		Model: "openai/gpt-4",
		Messages: []models.Message{
			models.NewTextMessage(models.RoleUser, "What's the weather like?"),
		},
		Prediction: &models.Prediction{
			Type:    "content",
			Content: "The weather today is sunny with",
		},
	}

	resp, err := client.CreateChatCompletion(context.Background(), req)
	require.NoError(t, err)

	content, _ := resp.Choices[0].Message.GetTextContent()
	assert.Contains(t, content, "sunny")
}

func TestAdvancedParameters(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req models.ChatCompletionRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)

		// Verify advanced parameters
		assert.Equal(t, 100, *req.MaxTokens)
		assert.Equal(t, 0.7, *req.Temperature)
		assert.Equal(t, 0.9, *req.TopP)
		assert.Equal(t, 40, *req.TopK)
		assert.Equal(t, 0.5, *req.FrequencyPenalty)
		assert.Equal(t, 0.3, *req.PresencePenalty)
		assert.Equal(t, 1.1, *req.RepetitionPenalty)
		assert.Equal(t, 12345, *req.Seed)
		assert.Equal(t, 5, *req.TopLogprobs)
		assert.Equal(t, 0.05, *req.MinP)
		assert.Equal(t, 0.1, *req.TopA)
		assert.NotNil(t, req.LogitBias)
		assert.Equal(t, 2.0, req.LogitBias["123"])

		resp := models.ChatCompletionResponse{
			ID:    "resp-123",
			Model: req.Model,
			Choices: []models.Choice{
				{
					Message: &models.Message{
						Role:    models.RoleAssistant,
						Content: json.RawMessage(`"Response with parameters"`),
					},
					Logprobs: &models.LogProbs{
						Content: []models.LogProbContent{
							{
								Token:   "Response",
								Logprob: -0.5,
								TopLogprobs: []models.TopLogProbContent{
									{Token: "Response", Logprob: -0.5},
									{Token: "Reply", Logprob: -1.2},
								},
							},
						},
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))

	maxTokens := 100
	temperature := 0.7
	topP := 0.9
	topK := 40
	frequencyPenalty := 0.5
	presencePenalty := 0.3
	repetitionPenalty := 1.1
	seed := 12345
	topLogprobs := 5
	minP := 0.05
	topA := 0.1

	req := models.ChatCompletionRequest{
		Model: "openai/gpt-4",
		Messages: []models.Message{
			models.NewTextMessage(models.RoleUser, "Test with parameters"),
		},
		MaxTokens:         &maxTokens,
		Temperature:       &temperature,
		TopP:              &topP,
		TopK:              &topK,
		FrequencyPenalty:  &frequencyPenalty,
		PresencePenalty:   &presencePenalty,
		RepetitionPenalty: &repetitionPenalty,
		Seed:              &seed,
		TopLogprobs:       &topLogprobs,
		MinP:              &minP,
		TopA:              &topA,
		LogitBias:         map[string]float64{"123": 2.0},
	}

	resp, err := client.CreateChatCompletion(context.Background(), req)
	require.NoError(t, err)
	assert.NotNil(t, resp)

	// Check logprobs in response
	if len(resp.Choices) > 0 && resp.Choices[0].Logprobs != nil {
		assert.NotEmpty(t, resp.Choices[0].Logprobs.Content)
	}
}

func TestStopSequences(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req models.ChatCompletionRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)

		// Verify stop sequences
		assert.Equal(t, []string{"\n\n", "END", "---"}, req.Stop)

		resp := models.ChatCompletionResponse{
			ID:    "resp-123",
			Model: req.Model,
			Choices: []models.Choice{
				{
					Message: &models.Message{
						Role:    models.RoleAssistant,
						Content: json.RawMessage(`"This is the response"`),
					},
					FinishReason: "stop",
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))

	req := models.ChatCompletionRequest{
		Model: "openai/gpt-4",
		Messages: []models.Message{
			models.NewTextMessage(models.RoleUser, "Generate text"),
		},
		Stop: []string{"\n\n", "END", "---"},
	}

	resp, err := client.CreateChatCompletion(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, "stop", resp.Choices[0].FinishReason)
}

// Helper functions
func boolPtr(b bool) *bool {
	return &b
}

func intPtr(i int) *int {
	return &i
}

func float64Ptr(f float64) *float64 {
	return &f
}

func TestUserIdentification(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req models.ChatCompletionRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)

		// Verify user identification
		assert.Equal(t, "user-12345", req.User)

		resp := models.ChatCompletionResponse{
			ID:    "resp-123",
			Model: req.Model,
			Choices: []models.Choice{
				{
					Message: &models.Message{
						Role:    models.RoleAssistant,
						Content: json.RawMessage(`"Response for user"`),
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))

	req := models.ChatCompletionRequest{
		Model: "openai/gpt-4",
		Messages: []models.Message{
			models.NewTextMessage(models.RoleUser, "Hello"),
		},
		User: "user-12345",
	}

	resp, err := client.CreateChatCompletion(context.Background(), req)
	require.NoError(t, err)
	assert.NotNil(t, resp)
}
