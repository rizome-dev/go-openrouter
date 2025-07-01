package openrouter

import (
	"encoding/json"
	"testing"

	"github.com/rizome-dev/go-openrouter/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateSchema(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected map[string]interface{}
	}{
		{
			name: "Simple struct",
			input: struct {
				Name string `json:"name"`
				Age  int    `json:"age"`
			}{},
			expected: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{"type": "string"},
					"age":  map[string]interface{}{"type": "integer"},
				},
				"required":             []string{"name", "age"},
				"additionalProperties": false,
			},
		},
		{
			name: "Struct with omitempty",
			input: struct {
				Required string `json:"required"`
				Optional string `json:"optional,omitempty"`
			}{},
			expected: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"required": map[string]interface{}{"type": "string"},
					"optional": map[string]interface{}{"type": "string"},
				},
				"required":             []string{"required"},
				"additionalProperties": false,
			},
		},
		{
			name: "Nested struct",
			input: struct {
				Name    string `json:"name"`
				Address struct {
					Street string `json:"street"`
					City   string `json:"city"`
				} `json:"address"`
			}{},
			expected: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{"type": "string"},
					"address": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"street": map[string]interface{}{"type": "string"},
							"city":   map[string]interface{}{"type": "string"},
						},
						"required":             []string{"street", "city"},
						"additionalProperties": false,
					},
				},
				"required":             []string{"name", "address"},
				"additionalProperties": false,
			},
		},
		{
			name: "Array field",
			input: struct {
				Tags []string `json:"tags"`
			}{},
			expected: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"tags": map[string]interface{}{
						"type": "array",
						"items": map[string]interface{}{
							"type": "string",
						},
					},
				},
				"required":             []string{"tags"},
				"additionalProperties": false,
			},
		},
		{
			name:  "WeatherInfo struct",
			input: WeatherInfo{},
			expected: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"location": map[string]interface{}{
						"type":        "string",
						"description": "City or location name",
					},
					"temperature": map[string]interface{}{
						"type":        "number",
						"description": "Temperature in Celsius",
					},
					"conditions": map[string]interface{}{
						"type":        "string",
						"description": "Weather conditions description",
					},
					"humidity": map[string]interface{}{
						"type":        "integer",
						"description": "Humidity percentage",
					},
					"wind_speed": map[string]interface{}{
						"type":        "number",
						"description": "Wind speed in km/h",
					},
				},
				"required":             []string{"location", "temperature", "conditions"},
				"additionalProperties": false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema, err := GenerateSchema(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, schema)
		})
	}
}

func TestParseStructuredResponse(t *testing.T) {
	// Create a mock response
	content, _ := json.Marshal(`{
		"location": "Paris",
		"temperature": 22.5,
		"conditions": "Partly cloudy",
		"humidity": 65
	}`)

	resp := &models.ChatCompletionResponse{
		Choices: []models.Choice{
			{
				Message: &models.Message{
					Role:    models.RoleAssistant,
					Content: content,
				},
			},
		},
	}

	var weather WeatherInfo
	err := ParseStructuredResponse(resp, &weather)
	require.NoError(t, err)

	assert.Equal(t, "Paris", weather.Location)
	assert.Equal(t, 22.5, weather.Temperature)
	assert.Equal(t, "Partly cloudy", weather.Conditions)
	assert.Equal(t, 65, weather.Humidity)
}

func TestValidateJSONResponse(t *testing.T) {
	tests := []struct {
		name    string
		content string
		valid   bool
	}{
		{
			name:    "Valid JSON",
			content: `{"key": "value", "number": 42}`,
			valid:   true,
		},
		{
			name:    "Invalid JSON",
			content: `{key: "value"}`,
			valid:   false,
		},
		{
			name:    "Empty object",
			content: `{}`,
			valid:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ValidateJSONResponse(tt.content)
			if tt.valid {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestCreateStructuredPrompt(t *testing.T) {
	prompt := CreateStructuredPrompt(
		"What's the weather like?",
		"A JSON object with location, temperature, and conditions",
	)

	assert.Contains(t, prompt, "What's the weather like?")
	assert.Contains(t, prompt, "JSON object")
	assert.Contains(t, prompt, "location, temperature, and conditions")
}
