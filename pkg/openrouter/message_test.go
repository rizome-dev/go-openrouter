package openrouter

import (
	"encoding/json"
	"testing"

	"github.com/rizome-dev/go-openrouter/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMessageCreation(t *testing.T) {
	tests := []struct {
		name          string
		createMessage func() models.Message
		expectedRole  models.Role
		checkContent  func(t *testing.T, msg models.Message)
	}{
		{
			name: "Simple text message",
			createMessage: func() models.Message {
				return models.NewTextMessage(models.RoleUser, "Hello world")
			},
			expectedRole: models.RoleUser,
			checkContent: func(t *testing.T, msg models.Message) {
				content, err := msg.GetTextContent()
				assert.NoError(t, err)
				assert.Equal(t, "Hello world", content)
			},
		},
		{
			name: "Message with name",
			createMessage: func() models.Message {
				msg := models.NewTextMessage(models.RoleUser, "Hello")
				msg.Name = "Alice"
				return msg
			},
			expectedRole: models.RoleUser,
			checkContent: func(t *testing.T, msg models.Message) {
				assert.Equal(t, "Alice", msg.Name)
				content, _ := msg.GetTextContent()
				assert.Equal(t, "Hello", content)
			},
		},
		{
			name: "Tool message",
			createMessage: func() models.Message {
				return models.Message{
					Role:       models.RoleTool,
					Content:    json.RawMessage(`"Tool result"`),
					ToolCallID: "call-123",
					Name:       "weather_tool",
				}
			},
			expectedRole: models.RoleTool,
			checkContent: func(t *testing.T, msg models.Message) {
				assert.Equal(t, "call-123", msg.ToolCallID)
				assert.Equal(t, "weather_tool", msg.Name)
				content, _ := msg.GetTextContent()
				assert.Equal(t, "Tool result", content)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := tt.createMessage()
			assert.Equal(t, tt.expectedRole, msg.Role)
			tt.checkContent(t, msg)
		})
	}
}

func TestMultiContentMessage(t *testing.T) {
	textContent := models.TextContent{
		Type: models.ContentTypeText,
		Text: "What's in this image?",
	}

	imageContent := models.ImageContent{
		Type: models.ContentTypeImageURL,
		ImageURL: models.ImageURL{
			URL:    "https://example.com/image.jpg",
			Detail: "high",
		},
	}

	msg, err := models.NewMultiContentMessage(models.RoleUser, textContent, imageContent)
	require.NoError(t, err)
	assert.Equal(t, models.RoleUser, msg.Role)

	// Check content is a RawMessage that contains an array
	var contents []interface{}
	err = json.Unmarshal(msg.Content, &contents)
	require.NoError(t, err)
	assert.Len(t, contents, 2)
}

func TestMessageWithToolCalls(t *testing.T) {
	msg := models.Message{
		Role: models.RoleAssistant,
		ToolCalls: []models.ToolCall{
			{
				ID:   "call-1",
				Type: "function",
				Function: models.FunctionCall{
					Name:      "get_weather",
					Arguments: `{"location": "Tokyo"}`,
				},
			},
			{
				ID:   "call-2",
				Type: "function",
				Function: models.FunctionCall{
					Name:      "get_time",
					Arguments: `{"timezone": "JST"}`,
				},
			},
		},
	}

	assert.Len(t, msg.ToolCalls, 2)
	assert.Equal(t, "get_weather", msg.ToolCalls[0].Function.Name)
	assert.Equal(t, "get_time", msg.ToolCalls[1].Function.Name)
}

func TestGetTextContent(t *testing.T) {
	tests := []struct {
		name        string
		content     interface{}
		expected    string
		expectError bool
	}{
		{
			name:     "String content",
			content:  json.RawMessage(`"Hello world"`),
			expected: "Hello world",
		},
		{
			name:     "JSON object as string",
			content:  json.RawMessage(`{"message": "Hello"}`),
			expected: `{"message": "Hello"}`,
		},
		{
			name:        "Array content",
			content:     []interface{}{"text", "parts"},
			expectError: true,
		},
		{
			name:     "Null content",
			content:  json.RawMessage(`null`),
			expected: "",
		},
		{
			name:     "Empty string",
			content:  json.RawMessage(`""`),
			expected: "",
		},
		{
			name:     "Number as content",
			content:  json.RawMessage(`42`),
			expected: "42",
		},
		{
			name:     "Boolean as content",
			content:  json.RawMessage(`true`),
			expected: "true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var content json.RawMessage
			switch v := tt.content.(type) {
			case json.RawMessage:
				content = v
			case []interface{}:
				// For array content, marshal it to json.RawMessage
				data, _ := json.Marshal(v)
				content = json.RawMessage(data)
			default:
				// For other types, marshal to json
				data, _ := json.Marshal(v)
				content = json.RawMessage(data)
			}

			msg := models.Message{
				Role:    models.RoleAssistant,
				Content: content,
			}

			textContent, err := msg.GetTextContent()
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, textContent)
			}
		})
	}
}

func TestMessageAnnotations(t *testing.T) {
	msg := models.Message{
		Role:    models.RoleAssistant,
		Content: json.RawMessage(`"Here are the search results"`),
		Annotations: []models.Annotation{
			{
				Type: models.AnnotationTypeURLCitation,
				URLCitation: &models.URLCitation{
					URL:        "https://example.com/article",
					Title:      "Example Article",
					Content:    "Article content snippet",
					StartIndex: 0,
					EndIndex:   27,
				},
			},
			{
				Type: models.AnnotationTypeFile,
				File: &models.FileAnnotation{
					Filename: "document.pdf",
					FileData: map[string]interface{}{
						"id": "file-123",
					},
				},
			},
		},
	}

	assert.Len(t, msg.Annotations, 2)
	assert.Equal(t, models.AnnotationTypeURLCitation, msg.Annotations[0].Type)
	assert.Equal(t, models.AnnotationTypeFile, msg.Annotations[1].Type)

	// Verify URL citation
	urlCitation := msg.Annotations[0].URLCitation
	assert.NotNil(t, urlCitation)
	assert.Equal(t, "https://example.com/article", urlCitation.URL)
	assert.Equal(t, "Example Article", urlCitation.Title)

	// Verify file annotation
	fileAnnotation := msg.Annotations[1].File
	assert.NotNil(t, fileAnnotation)
	assert.Equal(t, "document.pdf", fileAnnotation.Filename)
	assert.Equal(t, "file-123", fileAnnotation.FileData["id"])
}

func TestContentTypeSerialization(t *testing.T) {
	tests := []struct {
		name    string
		content models.Content
		check   func(t *testing.T, data []byte)
	}{
		{
			name: "Text content",
			content: models.TextContent{
				Type: models.ContentTypeText,
				Text: "Hello",
			},
			check: func(t *testing.T, data []byte) {
				var result map[string]interface{}
				err := json.Unmarshal(data, &result)
				require.NoError(t, err)
				assert.Equal(t, "text", result["type"])
				assert.Equal(t, "Hello", result["text"])
			},
		},
		{
			name: "Image content",
			content: models.ImageContent{
				Type: models.ContentTypeImageURL,
				ImageURL: models.ImageURL{
					URL:    "https://example.com/img.jpg",
					Detail: "auto",
				},
			},
			check: func(t *testing.T, data []byte) {
				var result map[string]interface{}
				err := json.Unmarshal(data, &result)
				require.NoError(t, err)
				assert.Equal(t, "image_url", result["type"])
				imageURL := result["image_url"].(map[string]interface{})
				assert.Equal(t, "https://example.com/img.jpg", imageURL["url"])
				assert.Equal(t, "auto", imageURL["detail"])
			},
		},
		{
			name: "File content",
			content: models.FileContent{
				Type: models.ContentTypeFile,
				File: models.File{
					Filename: "doc.pdf",
					FileData: "data:application/pdf;base64,abc123",
				},
			},
			check: func(t *testing.T, data []byte) {
				var result map[string]interface{}
				err := json.Unmarshal(data, &result)
				require.NoError(t, err)
				assert.Equal(t, "file", result["type"])
				file := result["file"].(map[string]interface{})
				assert.Equal(t, "doc.pdf", file["filename"])
				assert.Contains(t, file["file_data"], "base64")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.content)
			require.NoError(t, err)
			tt.check(t, data)
		})
	}
}

func TestComplexMessageSerialization(t *testing.T) {
	// Test a complex message with all fields
	msg := models.Message{
		Role:    models.RoleAssistant,
		Content: json.RawMessage(`"This is the response"`),
		Name:    "Assistant",
		ToolCalls: []models.ToolCall{
			{
				ID:   "call-1",
				Type: "function",
				Function: models.FunctionCall{
					Name:      "search",
					Arguments: `{"query": "test"}`,
				},
			},
		},
		Annotations: []models.Annotation{
			{
				Type: models.AnnotationTypeURLCitation,
				URLCitation: &models.URLCitation{
					URL: "https://example.com",
				},
			},
		},
	}

	// Serialize and deserialize
	data, err := json.Marshal(msg)
	require.NoError(t, err)

	var decoded models.Message
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	// Verify all fields
	assert.Equal(t, msg.Role, decoded.Role)
	assert.Equal(t, msg.Name, decoded.Name)
	assert.Len(t, decoded.ToolCalls, 1)
	assert.Len(t, decoded.Annotations, 1)
}

func TestMessageValidation(t *testing.T) {
	tests := []struct {
		name        string
		msg         models.Message
		expectValid bool
		errorMsg    string
	}{
		{
			name: "Valid user message",
			msg: models.Message{
				Role:    models.RoleUser,
				Content: json.RawMessage(`"Hello"`),
			},
			expectValid: true,
		},
		{
			name: "Valid assistant message",
			msg: models.Message{
				Role:    models.RoleAssistant,
				Content: json.RawMessage(`"Response"`),
			},
			expectValid: true,
		},
		{
			name: "Valid system message",
			msg: models.Message{
				Role:    models.RoleSystem,
				Content: json.RawMessage(`"You are helpful"`),
			},
			expectValid: true,
		},
		{
			name: "Tool message without tool_call_id",
			msg: models.Message{
				Role:    models.RoleTool,
				Content: json.RawMessage(`"Result"`),
			},
			expectValid: false,
			errorMsg:    "tool_call_id required",
		},
		{
			name: "Valid tool message",
			msg: models.Message{
				Role:       models.RoleTool,
				Content:    json.RawMessage(`"Result"`),
				ToolCallID: "call-123",
			},
			expectValid: true,
		},
		{
			name: "Assistant with tool calls",
			msg: models.Message{
				Role: models.RoleAssistant,
				ToolCalls: []models.ToolCall{
					{
						ID:   "call-1",
						Type: "function",
						Function: models.FunctionCall{
							Name: "test",
						},
					},
				},
			},
			expectValid: true,
		},
		{
			name:        "Empty message",
			msg:         models.Message{},
			expectValid: false,
			errorMsg:    "role required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: In real implementation, you would have a Validate() method
			// For now, we're testing the expected structure
			if tt.expectValid {
				// Valid messages should serialize without issues
				_, err := json.Marshal(tt.msg)
				assert.NoError(t, err)
			}
		})
	}
}
