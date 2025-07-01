package streaming

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/rizome-dev/go-openrouter/pkg/models"
)

var (
	// ErrStreamClosed is returned when trying to read from a closed stream
	ErrStreamClosed = errors.New("stream is closed")

	// ErrInvalidSSE is returned when SSE data is malformed
	ErrInvalidSSE = errors.New("invalid SSE format")
)

// SSEParser parses Server-Sent Events
type SSEParser struct {
	reader *bufio.Reader
	closed bool
}

// NewSSEParser creates a new SSE parser
func NewSSEParser(reader io.Reader) *SSEParser {
	return &SSEParser{
		reader: bufio.NewReader(reader),
		closed: false,
	}
}

// ParseNext parses the next SSE event
func (p *SSEParser) ParseNext() (*SSEEvent, error) {
	if p.closed {
		return nil, ErrStreamClosed
	}

	var event SSEEvent
	var dataBuffer bytes.Buffer

	for {
		line, err := p.reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				p.closed = true
				if dataBuffer.Len() > 0 {
					event.Data = dataBuffer.String()
					return &event, nil
				}
				return nil, io.EOF
			}
			return nil, fmt.Errorf("error reading stream: %w", err)
		}

		line = strings.TrimSpace(line)

		// Empty line signals end of event
		if line == "" && dataBuffer.Len() > 0 {
			event.Data = dataBuffer.String()
			return &event, nil
		}

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}

		// Parse field
		colonIndex := strings.Index(line, ":")
		if colonIndex == -1 {
			continue
		}

		field := line[:colonIndex]
		value := strings.TrimSpace(line[colonIndex+1:])

		switch field {
		case "event":
			event.Event = value
		case "data":
			if dataBuffer.Len() > 0 {
				dataBuffer.WriteString("\n")
			}
			dataBuffer.WriteString(value)
		case "id":
			event.ID = value
		case "retry":
			// Ignore retry field for now
		}
	}
}

// SSEEvent represents a Server-Sent Event
type SSEEvent struct {
	Event string
	Data  string
	ID    string
}

// ChatCompletionStreamReader reads streaming chat completions
type ChatCompletionStreamReader struct {
	parser *SSEParser
	closer io.Closer
}

// NewChatCompletionStreamReader creates a new stream reader
func NewChatCompletionStreamReader(reader io.ReadCloser) *ChatCompletionStreamReader {
	return &ChatCompletionStreamReader{
		parser: NewSSEParser(reader),
		closer: reader,
	}
}

// Read reads the next chunk from the stream
func (r *ChatCompletionStreamReader) Read() (*models.ChatCompletionResponse, error) {
	event, err := r.parser.ParseNext()
	if err != nil {
		return nil, err
	}

	// Skip comments
	if strings.HasPrefix(event.Data, ": ") {
		return r.Read() // Recursively read next event
	}

	// Check for end of stream
	if event.Data == "[DONE]" {
		return nil, io.EOF
	}

	// Check for top-level error first
	var errorCheck struct {
		Error *models.ChoiceError `json:"error,omitempty"`
	}
	if err := json.Unmarshal([]byte(event.Data), &errorCheck); err == nil && errorCheck.Error != nil {
		return nil, fmt.Errorf("openrouter error %d: %s", errorCheck.Error.Code, errorCheck.Error.Message)
	}

	// Parse JSON response
	var response models.ChatCompletionResponse
	if err := json.Unmarshal([]byte(event.Data), &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// Close closes the stream
func (r *ChatCompletionStreamReader) Close() error {
	if r.closer != nil {
		return r.closer.Close()
	}
	return nil
}

// CompletionStreamReader reads streaming text completions
type CompletionStreamReader struct {
	parser *SSEParser
	closer io.Closer
}

// NewCompletionStreamReader creates a new completion stream reader
func NewCompletionStreamReader(reader io.ReadCloser) *CompletionStreamReader {
	return &CompletionStreamReader{
		parser: NewSSEParser(reader),
		closer: reader,
	}
}

// Read reads the next chunk from the stream
func (r *CompletionStreamReader) Read() (*CompletionResponse, error) {
	event, err := r.parser.ParseNext()
	if err != nil {
		return nil, err
	}

	// Skip comments
	if strings.HasPrefix(event.Data, ": ") {
		return r.Read() // Recursively read next event
	}

	// Check for end of stream
	if event.Data == "[DONE]" {
		return nil, io.EOF
	}

	// Parse JSON response
	var response CompletionResponse
	if err := json.Unmarshal([]byte(event.Data), &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// Close closes the stream
func (r *CompletionStreamReader) Close() error {
	if r.closer != nil {
		return r.closer.Close()
	}
	return nil
}

// CompletionResponse represents a streaming completion response chunk
type CompletionResponse struct {
	ID      string                   `json:"id"`
	Object  string                   `json:"object"`
	Created int64                    `json:"created"`
	Model   string                   `json:"model"`
	Choices []CompletionStreamChoice `json:"choices"`
	Usage   *models.Usage            `json:"usage,omitempty"`
}

// CompletionStreamChoice represents a streaming choice
type CompletionStreamChoice struct {
	Index        int                 `json:"index"`
	Text         string              `json:"text,omitempty"`
	FinishReason string              `json:"finish_reason,omitempty"`
	Error        *models.ChoiceError `json:"error,omitempty"`
}
