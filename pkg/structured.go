package pkg

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/rizome-dev/go-openrouter/pkg/models"
)

// StructuredOutput provides helper methods for structured outputs
type StructuredOutput struct {
	client *Client
}

// NewStructuredOutput creates a new structured output helper
func NewStructuredOutput(client *Client) *StructuredOutput {
	return &StructuredOutput{client: client}
}

// GenerateSchema generates a JSON schema from a Go struct
func GenerateSchema(v interface{}) (map[string]interface{}, error) {
	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("input must be a struct or pointer to struct")
	}

	return generateSchemaFromType(t), nil
}

func generateSchemaFromType(t reflect.Type) map[string]interface{} {
	schema := map[string]interface{}{
		"type":                 "object",
		"properties":           make(map[string]interface{}),
		"required":             []string{},
		"additionalProperties": false,
	}

	properties := schema["properties"].(map[string]interface{})
	required := []string{}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// Skip unexported fields
		if field.PkgPath != "" {
			continue
		}

		// Get JSON tag
		jsonTag := field.Tag.Get("json")
		if jsonTag == "-" {
			continue
		}

		fieldName := field.Name
		if jsonTag != "" {
			fieldName = jsonTag
			// Handle omitempty
			if idx := len(fieldName); idx > 10 && fieldName[idx-10:] == ",omitempty" {
				fieldName = fieldName[:idx-10]
			} else {
				required = append(required, fieldName)
			}
		} else {
			required = append(required, fieldName)
		}

		// Get description from tag
		description := field.Tag.Get("description")

		// Generate schema for field
		fieldSchema := generateFieldSchema(field.Type)
		if description != "" {
			fieldSchema["description"] = description
		}

		properties[fieldName] = fieldSchema
	}

	if len(required) > 0 {
		schema["required"] = required
	}

	return schema
}

func generateFieldSchema(t reflect.Type) map[string]interface{} {
	switch t.Kind() {
	case reflect.String:
		return map[string]interface{}{"type": "string"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return map[string]interface{}{"type": "integer"}
	case reflect.Float32, reflect.Float64:
		return map[string]interface{}{"type": "number"}
	case reflect.Bool:
		return map[string]interface{}{"type": "boolean"}
	case reflect.Slice, reflect.Array:
		return map[string]interface{}{
			"type":  "array",
			"items": generateFieldSchema(t.Elem()),
		}
	case reflect.Struct:
		return generateSchemaFromType(t)
	case reflect.Ptr:
		// For pointers, generate schema for the element type
		return generateFieldSchema(t.Elem())
	default:
		return map[string]interface{}{"type": "object"}
	}
}

// CreateWithSchema creates a completion with a structured output schema
func (s *StructuredOutput) CreateWithSchema(ctx context.Context, req models.ChatCompletionRequest, schemaName string, schema interface{}) (*models.ChatCompletionResponse, error) {
	// Generate schema if it's a struct
	var jsonSchema map[string]interface{}

	switch v := schema.(type) {
	case map[string]interface{}:
		jsonSchema = v
	default:
		var err error
		jsonSchema, err = GenerateSchema(schema)
		if err != nil {
			return nil, fmt.Errorf("failed to generate schema: %w", err)
		}
	}

	// Marshal schema
	schemaBytes, err := json.Marshal(jsonSchema)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal schema: %w", err)
	}

	// Set response format
	req.ResponseFormat = &models.ResponseFormat{
		Type: "json_schema",
		JSONSchema: &models.JSONSchema{
			Name:   schemaName,
			Strict: true,
			Schema: schemaBytes,
		},
	}

	return s.client.CreateChatCompletion(ctx, req)
}

// ParseStructuredResponse parses a structured response into a Go struct
func ParseStructuredResponse(resp *models.ChatCompletionResponse, target interface{}) error {
	if len(resp.Choices) == 0 || resp.Choices[0].Message == nil {
		return fmt.Errorf("no message in response")
	}

	content, err := resp.Choices[0].Message.GetTextContent()
	if err != nil {
		return fmt.Errorf("failed to get text content: %w", err)
	}

	if err := json.Unmarshal([]byte(content), target); err != nil {
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return nil
}

// Example structured output types

// WeatherInfo represents weather information
type WeatherInfo struct {
	Location    string  `json:"location" description:"City or location name"`
	Temperature float64 `json:"temperature" description:"Temperature in Celsius"`
	Conditions  string  `json:"conditions" description:"Weather conditions description"`
	Humidity    int     `json:"humidity,omitempty" description:"Humidity percentage"`
	WindSpeed   float64 `json:"wind_speed,omitempty" description:"Wind speed in km/h"`
}

// ExtractedData represents extracted information
type ExtractedData struct {
	Title    string   `json:"title" description:"Main title or subject"`
	Summary  string   `json:"summary" description:"Brief summary"`
	Keywords []string `json:"keywords" description:"Key terms or topics"`
	Entities []Entity `json:"entities" description:"Named entities found"`
}

// Entity represents a named entity
type Entity struct {
	Name string `json:"name" description:"Entity name"`
	Type string `json:"type" description:"Entity type (person, place, organization, etc.)"`
}

// CreateStructuredPrompt creates a prompt that encourages JSON output
func CreateStructuredPrompt(userPrompt string, schemaDescription string) string {
	return fmt.Sprintf(`%s

Please respond with a valid JSON object that follows this structure: %s

Ensure your response is valid JSON with no additional text.`, userPrompt, schemaDescription)
}

// ValidateJSONResponse validates that a response contains valid JSON
func ValidateJSONResponse(content string) (map[string]interface{}, error) {
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return nil, fmt.Errorf("invalid JSON response: %w", err)
	}
	return result, nil
}
