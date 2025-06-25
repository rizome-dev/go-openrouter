package models

import (
	"encoding/json"
)

// Role represents the role of a message in a conversation
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

// ContentType represents the type of content in a message
type ContentType string

const (
	ContentTypeText     ContentType = "text"
	ContentTypeImageURL ContentType = "image_url"
	ContentTypeFile     ContentType = "file"
)

// TextContent represents text content in a message
type TextContent struct {
	Type ContentType `json:"type"`
	Text string      `json:"text"`
}

// ImageContent represents image content in a message
type ImageContent struct {
	Type     ContentType `json:"type"`
	ImageURL ImageURL    `json:"image_url"`
}

// ImageURL represents an image URL with optional detail level
type ImageURL struct {
	URL    string `json:"url"`
	Detail string `json:"detail,omitempty"`
}

// FileContent represents file content in a message
type FileContent struct {
	Type ContentType `json:"type"`
	File File        `json:"file"`
}

// File represents a file with its data
type File struct {
	Filename string `json:"filename"`
	FileData string `json:"file_data"` // Base64 encoded data URL
}

// Content represents any type of content in a message
type Content interface {
	contentType() ContentType
}

func (t TextContent) contentType() ContentType  { return ContentTypeText }
func (i ImageContent) contentType() ContentType { return ContentTypeImageURL }
func (f FileContent) contentType() ContentType  { return ContentTypeFile }

// Message represents a message in a conversation
type Message struct {
	Role       Role            `json:"role"`
	Content    json.RawMessage `json:"content"`
	Name       string          `json:"name,omitempty"`
	ToolCallID string          `json:"tool_call_id,omitempty"`
	ToolCalls  []ToolCall      `json:"tool_calls,omitempty"`
	
	// For responses with web search or file annotations
	Annotations []Annotation `json:"annotations,omitempty"`
}

// NewTextMessage creates a new text message
func NewTextMessage(role Role, text string) Message {
	content, _ := json.Marshal(text)
	return Message{
		Role:    role,
		Content: content,
	}
}

// NewMultiContentMessage creates a message with multiple content parts
func NewMultiContentMessage(role Role, contents ...Content) (Message, error) {
	var contentParts []interface{}
	for _, c := range contents {
		contentParts = append(contentParts, c)
	}
	
	content, err := json.Marshal(contentParts)
	if err != nil {
		return Message{}, err
	}
	
	return Message{
		Role:    role,
		Content: content,
	}, nil
}

// NewToolMessage creates a new tool response message
func NewToolMessage(toolCallID, name, content string) Message {
	contentJSON, _ := json.Marshal(content)
	return Message{
		Role:       RoleTool,
		Content:    contentJSON,
		ToolCallID: toolCallID,
		Name:       name,
	}
}

// GetTextContent attempts to get the text content from a message
func (m Message) GetTextContent() (string, error) {
	var text string
	if err := json.Unmarshal(m.Content, &text); err != nil {
		return "", err
	}
	return text, nil
}

// GetMultiContent attempts to get multi-part content from a message
func (m Message) GetMultiContent() ([]Content, error) {
	var rawParts []json.RawMessage
	if err := json.Unmarshal(m.Content, &rawParts); err != nil {
		return nil, err
	}
	
	var contents []Content
	for _, raw := range rawParts {
		var typeCheck struct {
			Type ContentType `json:"type"`
		}
		if err := json.Unmarshal(raw, &typeCheck); err != nil {
			continue
		}
		
		switch typeCheck.Type {
		case ContentTypeText:
			var tc TextContent
			if err := json.Unmarshal(raw, &tc); err == nil {
				contents = append(contents, tc)
			}
		case ContentTypeImageURL:
			var ic ImageContent
			if err := json.Unmarshal(raw, &ic); err == nil {
				contents = append(contents, ic)
			}
		case ContentTypeFile:
			var fc FileContent
			if err := json.Unmarshal(raw, &fc); err == nil {
				contents = append(contents, fc)
			}
		}
	}
	
	return contents, nil
}