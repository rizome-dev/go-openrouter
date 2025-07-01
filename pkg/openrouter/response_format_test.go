package openrouter

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rizome-dev/go-openrouter/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResponseFormat(t *testing.T) {
	tests := []struct {
		name           string
		responseFormat *models.ResponseFormat
		expectedResp   interface{}
		verifyRequest  func(t *testing.T, req models.ChatCompletionRequest)
	}{
		{
			name: "JSON object response format",
			responseFormat: &models.ResponseFormat{
				Type: "json_object",
			},
			expectedResp: map[string]interface{}{
				"name": "John Doe",
				"age":  float64(30),
			},
			verifyRequest: func(t *testing.T, req models.ChatCompletionRequest) {
				assert.NotNil(t, req.ResponseFormat)
				assert.Equal(t, "json_object", req.ResponseFormat.Type)
			},
		},
		{
			name: "JSON schema response format",
			responseFormat: &models.ResponseFormat{
				Type: "json_schema",
				JSONSchema: &models.JSONSchema{
					Name:   "weather",
					Strict: true,
					Schema: json.RawMessage(`{
						"type": "object",
						"properties": {
							"location": {
								"type": "string",
								"description": "City name"
							},
							"temperature": {
								"type": "number",
								"description": "Temperature in Celsius"
							},
							"conditions": {
								"type": "string",
								"description": "Weather conditions"
							}
						},
						"required": ["location", "temperature", "conditions"],
						"additionalProperties": false
					}`),
				},
			},
			expectedResp: map[string]interface{}{
				"location":    "Tokyo",
				"temperature": 22.5,
				"conditions":  "Partly cloudy",
			},
			verifyRequest: func(t *testing.T, req models.ChatCompletionRequest) {
				assert.NotNil(t, req.ResponseFormat)
				assert.Equal(t, "json_schema", req.ResponseFormat.Type)
				assert.NotNil(t, req.ResponseFormat.JSONSchema)
				assert.Equal(t, "weather", req.ResponseFormat.JSONSchema.Name)
				assert.True(t, req.ResponseFormat.JSONSchema.Strict)
				// Schema should be present
				assert.NotEmpty(t, req.ResponseFormat.JSONSchema.Schema)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var req models.ChatCompletionRequest
				err := json.NewDecoder(r.Body).Decode(&req)
				require.NoError(t, err)

				// Verify request
				tt.verifyRequest(t, req)

				// Return structured response
				respJSON, _ := json.Marshal(tt.expectedResp)
				resp := models.ChatCompletionResponse{
					ID:    "resp-123",
					Model: "openai/gpt-4",
					Choices: []models.Choice{
						{
							Message: &models.Message{
								Role:    models.RoleAssistant,
								Content: respJSON,
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
					models.NewTextMessage(models.RoleUser, "Generate data"),
				},
				ResponseFormat: tt.responseFormat,
			}

			resp, err := client.CreateChatCompletion(context.Background(), req)
			require.NoError(t, err)
			assert.NotNil(t, resp)

			// Verify structured response
			var result map[string]interface{}
			content, _ := resp.Choices[0].Message.GetTextContent()
			err = json.Unmarshal([]byte(content), &result)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedResp, result)
		})
	}
}

func TestStructuredOutputWithStreaming(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req models.ChatCompletionRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)

		// Verify streaming and response format
		assert.True(t, req.Stream)
		assert.NotNil(t, req.ResponseFormat)

		w.Header().Set("Content-Type", "text/event-stream")
		flusher := w.(http.Flusher)

		// Stream partial JSON that builds up to valid JSON
		chunks := []string{
			`{"choices":[{"delta":{"content":"{"}}]}`,
			`{"choices":[{"delta":{"content":"\"name\":"}}]}`,
			`{"choices":[{"delta":{"content":"\"Alice\","}}]}`,
			`{"choices":[{"delta":{"content":"\"age\":"}}]}`,
			`{"choices":[{"delta":{"content":"25"}}]}`,
			`{"choices":[{"delta":{"content":"}"}}]}`,
			`{"choices":[{"delta":{},"finish_reason":"stop"}]}`,
		}

		for _, chunk := range chunks {
			fmt.Fprintf(w, "data: %s\n\n", chunk)
			flusher.Flush()
		}

		fmt.Fprintf(w, "data: [DONE]\n\n")
		flusher.Flush()
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))

	req := models.ChatCompletionRequest{
		Model: "openai/gpt-4",
		Messages: []models.Message{
			models.NewTextMessage(models.RoleUser, "Generate user data"),
		},
		ResponseFormat: &models.ResponseFormat{
			Type: "json_object",
		},
	}

	stream, err := client.CreateChatCompletionStream(context.Background(), req)
	require.NoError(t, err)
	defer stream.Close()

	// Collect streamed content
	var fullContent string
	for {
		chunk, err := stream.Read()
		if err != nil {
			break
		}
		if len(chunk.Choices) > 0 && len(chunk.Choices[0].Delta.Content) > 0 {
			var deltaContent string
			err := json.Unmarshal(chunk.Choices[0].Delta.Content, &deltaContent)
			if err == nil {
				fullContent += deltaContent
			}
		}
	}

	// Verify final JSON is valid
	var result map[string]interface{}
	err = json.Unmarshal([]byte(fullContent), &result)
	require.NoError(t, err)
	assert.Equal(t, "Alice", result["name"])
	assert.Equal(t, float64(25), result["age"])
}

func TestComplexJSONSchema(t *testing.T) {
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"user": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"id": map[string]interface{}{
						"type":        "integer",
						"description": "User ID",
					},
					"name": map[string]interface{}{
						"type":        "string",
						"description": "User name",
					},
					"email": map[string]interface{}{
						"type":        "string",
						"format":      "email",
						"description": "User email",
					},
					"roles": map[string]interface{}{
						"type": "array",
						"items": map[string]interface{}{
							"type": "string",
							"enum": []interface{}{"admin", "user", "guest"},
						},
						"description": "User roles",
					},
				},
				"required": []interface{}{"id", "name", "email"},
			},
			"settings": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"theme": map[string]interface{}{
						"type":    "string",
						"enum":    []interface{}{"light", "dark", "auto"},
						"default": "auto",
					},
					"notifications": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"email": map[string]interface{}{
								"type": "boolean",
							},
							"push": map[string]interface{}{
								"type": "boolean",
							},
						},
					},
				},
			},
		},
		"required":             []interface{}{"user"},
		"additionalProperties": false,
	}

	expectedResponse := map[string]interface{}{
		"user": map[string]interface{}{
			"id":    123,
			"name":  "John Doe",
			"email": "john@example.com",
			"roles": []string{"admin", "user"},
		},
		"settings": map[string]interface{}{
			"theme": "dark",
			"notifications": map[string]interface{}{
				"email": true,
				"push":  false,
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req models.ChatCompletionRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)

		// Verify complex schema
		assert.NotNil(t, req.ResponseFormat)
		assert.NotNil(t, req.ResponseFormat.JSONSchema)
		// Schema should be marshaled to JSON
		var parsedSchema map[string]interface{}
		err = json.Unmarshal(req.ResponseFormat.JSONSchema.Schema, &parsedSchema)
		require.NoError(t, err)
		assert.Equal(t, schema, parsedSchema)

		respJSON, _ := json.Marshal(expectedResponse)
		resp := models.ChatCompletionResponse{
			ID:    "resp-123",
			Model: "openai/gpt-4",
			Choices: []models.Choice{
				{
					Message: &models.Message{
						Role:    models.RoleAssistant,
						Content: respJSON,
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))

	// Marshal schema to JSON
	schemaJSON, err := json.Marshal(schema)
	require.NoError(t, err)

	req := models.ChatCompletionRequest{
		Model: "openai/gpt-4",
		Messages: []models.Message{
			models.NewTextMessage(models.RoleUser, "Generate user profile"),
		},
		ResponseFormat: &models.ResponseFormat{
			Type: "json_schema",
			JSONSchema: &models.JSONSchema{
				Name:   "user_profile",
				Strict: true,
				Schema: json.RawMessage(schemaJSON),
			},
		},
	}

	resp, err := client.CreateChatCompletion(context.Background(), req)
	require.NoError(t, err)

	var result map[string]interface{}
	content, _ := resp.Choices[0].Message.GetTextContent()
	err = json.Unmarshal([]byte(content), &result)
	require.NoError(t, err)

	// Verify nested structure
	user := result["user"].(map[string]interface{})
	assert.Equal(t, float64(123), user["id"])
	assert.Equal(t, "John Doe", user["name"])
	assert.Equal(t, "john@example.com", user["email"])
}

func TestResponseFormatWithProviderRouting(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req models.ChatCompletionRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)

		// Verify provider preferences for structured output support
		assert.NotNil(t, req.Provider)
		assert.NotNil(t, req.Provider.RequireParameters)
		assert.True(t, *req.Provider.RequireParameters)

		resp := models.ChatCompletionResponse{
			ID:    "resp-123",
			Model: "openai/gpt-4",
			Choices: []models.Choice{
				{
					Message: &models.Message{
						Role:    models.RoleAssistant,
						Content: json.RawMessage(`{"result": "success"}`),
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
			models.NewTextMessage(models.RoleUser, "Generate response"),
		},
		ResponseFormat: &models.ResponseFormat{
			Type: "json_object",
		},
		Provider: &models.ProviderPreferences{
			RequireParameters: boolPtr(true),
		},
	}

	resp, err := client.CreateChatCompletion(context.Background(), req)
	require.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestInvalidJSONSchemaHandling(t *testing.T) {
	tests := []struct {
		name          string
		schema        *models.JSONSchema
		errorResponse bool
		errorMessage  string
	}{
		{
			name: "Invalid schema format",
			schema: &models.JSONSchema{
				Name: "invalid",
				Schema: json.RawMessage(`{
					"type": "invalid_type"
				}`),
			},
			errorResponse: true,
			errorMessage:  "Invalid JSON schema",
		},
		{
			name: "Missing required name",
			schema: &models.JSONSchema{
				Schema: json.RawMessage(`{
					"type": "object"
				}`),
			},
			errorResponse: true,
			errorMessage:  "JSON schema name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.errorResponse {
					w.WriteHeader(400)
					json.NewEncoder(w).Encode(map[string]interface{}{
						"error": map[string]interface{}{
							"code":    400,
							"message": tt.errorMessage,
						},
					})
				} else {
					resp := models.ChatCompletionResponse{
						ID:    "resp-123",
						Model: "test-model",
						Choices: []models.Choice{
							{
								Message: &models.Message{
									Role:    models.RoleAssistant,
									Content: json.RawMessage(`{"valid": "response"}`),
								},
							},
						},
					}
					json.NewEncoder(w).Encode(resp)
				}
			}))
			defer server.Close()

			client := NewClient("test-key", WithBaseURL(server.URL))

			req := models.ChatCompletionRequest{
				Model: "openai/gpt-4",
				Messages: []models.Message{
					models.NewTextMessage(models.RoleUser, "Test"),
				},
				ResponseFormat: &models.ResponseFormat{
					Type:       "json_schema",
					JSONSchema: tt.schema,
				},
			}

			_, err := client.CreateChatCompletion(context.Background(), req)

			if tt.errorResponse {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMessage)
			} else {
				require.NoError(t, err)
			}
		})
	}
}