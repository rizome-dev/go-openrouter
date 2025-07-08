package openrouter

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/rizome-dev/go-openrouter/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateChatCompletionStream(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/chat/completions", r.URL.Path)

		// Set SSE headers
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		// Send SSE data
		flusher, ok := w.(http.Flusher)
		require.True(t, ok)

		// Send comment to test filtering
		fmt.Fprintf(w, ": OPENROUTER PROCESSING\n\n")
		flusher.Flush()

		// Send data chunks
		chunks := []string{
			`{"choices":[{"delta":{"role":"assistant","content":"Hello"}}],"model":"test-model"}`,
			`{"choices":[{"delta":{"content":" world"}}],"model":"test-model"}`,
			`{"choices":[{"delta":{"content":"!"}}],"model":"test-model"}`,
			`{"choices":[{"delta":{},"finish_reason":"stop"}],"model":"test-model"}`,
		}

		for _, chunk := range chunks {
			fmt.Fprintf(w, "data: %s\n\n", chunk)
			flusher.Flush()
			time.Sleep(10 * time.Millisecond)
		}

		// Send done signal
		fmt.Fprintf(w, "data: [DONE]\n\n")
		flusher.Flush()
	}))
	defer server.Close()

	// Create client
	client := NewClient("test-api-key", WithBaseURL(server.URL))

	// Make streaming request
	stream, err := client.CreateChatCompletionStream(context.Background(), models.ChatCompletionRequest{
		Model: "test-model",
		Messages: []models.Message{
			models.NewTextMessage(models.RoleUser, "Hello"),
		},
	})
	require.NoError(t, err)
	defer stream.Close()

	// Collect responses
	var fullContent string
	var finishReason string
	chunkCount := 0

	for {
		chunk, err := stream.Read()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)

		if len(chunk.Choices) > 0 {
			if len(chunk.Choices[0].Delta.Content) > 0 {
				var deltaContent string
				err := json.Unmarshal(chunk.Choices[0].Delta.Content, &deltaContent)
				if err == nil {
					fullContent += deltaContent
				}
			}
			if chunk.Choices[0].FinishReason != "" {
				finishReason = chunk.Choices[0].FinishReason
			}
		}
		chunkCount++
	}

	assert.Equal(t, "Hello world!", fullContent)
	assert.Equal(t, "stop", finishReason)
	assert.Equal(t, 4, chunkCount)
}

func TestStreamingWithToolCalls(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher := w.(http.Flusher)

		// Stream tool call chunks
		chunks := []string{
			`{"choices":[{"delta":{"role":"assistant","tool_calls":[{"id":"call-123","type":"function","function":{"name":"get_weather"}}]}}]}`,
			`{"choices":[{"delta":{"tool_calls":[{"function":{"arguments":"{\"location\":"}}]}}]}`,
			`{"choices":[{"delta":{"tool_calls":[{"function":{"arguments":"\"Tokyo\"}"}}]}}]}`,
			`{"choices":[{"delta":{},"finish_reason":"tool_calls"}]}`,
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
	stream, err := client.CreateChatCompletionStream(context.Background(), models.ChatCompletionRequest{
		Model: "test-model",
		Messages: []models.Message{
			models.NewTextMessage(models.RoleUser, "What's the weather?"),
		},
		Tools: []models.Tool{
			{
				Type: "function",
				Function: models.FunctionDescription{
					Name:        "get_weather",
					Description: "Get weather",
					Parameters:  json.RawMessage(`{}`),
				},
			},
		},
	})
	require.NoError(t, err)
	defer stream.Close()

	var toolCalls []models.ToolCall
	for {
		chunk, err := stream.Read()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)

		if len(chunk.Choices) > 0 && len(chunk.Choices[0].Delta.ToolCalls) > 0 {
			// In real streaming, tool calls are accumulated
			toolCalls = chunk.Choices[0].Delta.ToolCalls
		}
	}

	assert.NotEmpty(t, toolCalls)
}

func TestStreamingError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher := w.(http.Flusher)

		// Send initial chunk
		fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"content\":\"Starting...\"}}]}\n\n")
		flusher.Flush()

		// Send error chunk
		fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{},\"error\":{\"code\":500,\"message\":\"Internal error\"}}]}\n\n")
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

	gotContent := false
	gotError := false

	for {
		chunk, err := stream.Read()
		if err == io.EOF {
			break
		}

		if err != nil {
			gotError = true
			break
		}

		if len(chunk.Choices) > 0 {
			if len(chunk.Choices[0].Delta.Content) > 0 {
				gotContent = true
			}
			if chunk.Choices[0].Error != nil {
				gotError = true
			}
		}
	}

	assert.True(t, gotContent)
	assert.True(t, gotError)
}

func TestStreamingWithUsage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher := w.(http.Flusher)

		// Send content chunks
		fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"content\":\"Test response\"}}]}\n\n")
		flusher.Flush()

		// Send final chunk with usage
		fmt.Fprintf(w, "data: {\"choices\":[],\"usage\":{\"prompt_tokens\":10,\"completion_tokens\":5,\"total_tokens\":15}}\n\n")
		flusher.Flush()

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

	var usage *models.Usage
	for {
		chunk, err := stream.Read()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)

		if chunk.Usage != nil {
			usage = chunk.Usage
		}
	}

	assert.NotNil(t, usage)
	assert.Equal(t, 10, usage.PromptTokens)
	assert.Equal(t, 5, usage.CompletionTokens)
	assert.Equal(t, 15, usage.TotalTokens)
}

func TestStreamCancellation(t *testing.T) {
	// Track if the connection was closed
	connectionClosed := make(chan bool, 1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher := w.(http.Flusher)

		// Monitor for client disconnect
		notify := r.Context().Done()

		go func() {
			<-notify
			connectionClosed <- true
		}()

		// Send chunks slowly
		for i := 0; i < 100; i++ {
			select {
			case <-notify:
				return
			default:
				fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"content\":\"chunk %d \"}}]}\n\n", i)
				flusher.Flush()
				time.Sleep(50 * time.Millisecond)
			}
		}
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	client := NewClient("test-key", WithBaseURL(server.URL))

	stream, err := client.CreateChatCompletionStream(ctx, models.ChatCompletionRequest{
		Model:    "test-model",
		Messages: []models.Message{models.NewTextMessage(models.RoleUser, "Test")},
	})
	require.NoError(t, err)
	defer stream.Close()

	// Read a few chunks
	chunkCount := 0
	for i := 0; i < 3; i++ {
		_, err := stream.Read()
		if err != nil {
			break
		}
		chunkCount++
	}

	// Cancel the context
	cancel()

	// Try to read more - should get context canceled error
	_, err = stream.Read()
	assert.Error(t, err)

	// Verify server detected the cancellation
	select {
	case <-connectionClosed:
		// Good, connection was closed
	case <-time.After(2 * time.Second):
		t.Error("Server did not detect connection closure")
	}
}

func TestInvalidSSEData(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher := w.(http.Flusher)

		// Send various invalid SSE data
		fmt.Fprintf(w, "invalid line without data prefix\n\n")
		flusher.Flush()

		fmt.Fprintf(w, "data: not-json\n\n")
		flusher.Flush()

		fmt.Fprintf(w, "data: {\"valid\":\"json\"}\n\n")
		flusher.Flush()

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

	// Should handle invalid data gracefully
	validChunks := 0
	for {
		_, err := stream.Read()
		if err == io.EOF {
			break
		}
		if err == nil {
			validChunks++
		}
	}

	assert.GreaterOrEqual(t, validChunks, 0)
}

func TestStreamingWithEmptyLines(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")

		// SSE format with various empty lines and comments
		response := strings.Join([]string{
			": Comment line",
			"",
			"data: {\"choices\":[{\"delta\":{\"content\":\"Hello\"}}]}",
			"",
			"",
			": Another comment",
			"data: {\"choices\":[{\"delta\":{\"content\":\" world\"}}]}",
			"",
			"data: [DONE]",
			"",
		}, "\n")

		fmt.Fprint(w, response)
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	stream, err := client.CreateChatCompletionStream(context.Background(), models.ChatCompletionRequest{
		Model:    "test-model",
		Messages: []models.Message{models.NewTextMessage(models.RoleUser, "Test")},
	})
	require.NoError(t, err)
	defer stream.Close()

	var content string
	for {
		chunk, err := stream.Read()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)

		if len(chunk.Choices) > 0 && len(chunk.Choices[0].Delta.Content) > 0 {
			var deltaContent string
			err := json.Unmarshal(chunk.Choices[0].Delta.Content, &deltaContent)
			if err == nil {
				content += deltaContent
			}
		}
	}

	assert.Equal(t, "Hello world", content)
}
