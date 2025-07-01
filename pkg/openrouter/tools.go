package openrouter

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rizome-dev/go-openrouter/pkg/models"
)

// ToolExecutor is an interface for executing tool calls
type ToolExecutor interface {
	Execute(toolCall models.ToolCall) (string, error)
}

// ToolExecutorFunc is a function adapter for ToolExecutor
type ToolExecutorFunc func(models.ToolCall) (string, error)

// Execute implements ToolExecutor
func (f ToolExecutorFunc) Execute(toolCall models.ToolCall) (string, error) {
	return f(toolCall)
}

// ToolRegistry manages tool executors
type ToolRegistry struct {
	executors map[string]ToolExecutor
}

// NewToolRegistry creates a new tool registry
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		executors: make(map[string]ToolExecutor),
	}
}

// Register registers a tool executor
func (r *ToolRegistry) Register(name string, executor ToolExecutor) {
	r.executors[name] = executor
}

// RegisterFunc registers a tool executor function
func (r *ToolRegistry) RegisterFunc(name string, fn func(models.ToolCall) (string, error)) {
	r.executors[name] = ToolExecutorFunc(fn)
}

// Execute executes a tool call
func (r *ToolRegistry) Execute(toolCall models.ToolCall) (string, error) {
	executor, exists := r.executors[toolCall.Function.Name]
	if !exists {
		return "", fmt.Errorf("tool %s not registered", toolCall.Function.Name)
	}
	return executor.Execute(toolCall)
}

// Agent represents an autonomous agent that can handle tool calls
type Agent struct {
	client   *Client
	registry *ToolRegistry
	model    string
}

// NewAgent creates a new agent
func NewAgent(client *Client, model string) *Agent {
	return &Agent{
		client:   client,
		registry: NewToolRegistry(),
		model:    model,
	}
}

// RegisterTool registers a tool with the agent
func (a *Agent) RegisterTool(tool models.Tool, executor ToolExecutor) {
	a.registry.Register(tool.Function.Name, executor)
}

// RegisterToolFunc registers a tool function with the agent
func (a *Agent) RegisterToolFunc(tool models.Tool, fn func(models.ToolCall) (string, error)) {
	a.registry.RegisterFunc(tool.Function.Name, fn)
}

// RunOptions contains options for running the agent
type RunOptions struct {
	MaxIterations int
	Tools         []models.Tool
	ToolChoice    models.ToolChoice
}

// Run runs the agent with the given messages
func (a *Agent) Run(ctx context.Context, messages []models.Message, opts RunOptions) ([]models.Message, error) {
	if opts.MaxIterations <= 0 {
		opts.MaxIterations = 10
	}

	// Copy messages to avoid modifying the original
	conversationMessages := make([]models.Message, len(messages))
	copy(conversationMessages, messages)

	for iteration := 0; iteration < opts.MaxIterations; iteration++ {
		// Create request
		req := models.ChatCompletionRequest{
			Model:      a.model,
			Messages:   conversationMessages,
			Tools:      opts.Tools,
			ToolChoice: opts.ToolChoice,
		}

		// Get response
		resp, err := a.client.CreateChatCompletion(ctx, req)
		if err != nil {
			return conversationMessages, fmt.Errorf("iteration %d: %w", iteration, err)
		}

		if len(resp.Choices) == 0 {
			return conversationMessages, fmt.Errorf("no choices in response")
		}

		assistantMessage := resp.Choices[0].Message
		if assistantMessage == nil {
			return conversationMessages, fmt.Errorf("no message in choice")
		}

		// Add assistant message to conversation
		conversationMessages = append(conversationMessages, *assistantMessage)

		// Check if there are tool calls
		if len(assistantMessage.ToolCalls) == 0 {
			// No tool calls, we're done
			break
		}

		// Execute tool calls
		for _, toolCall := range assistantMessage.ToolCalls {
			result, err := a.registry.Execute(toolCall)
			if err != nil {
				result = fmt.Sprintf("Error executing tool: %v", err)
			}

			// Add tool result to conversation
			toolMessage := models.NewToolMessage(toolCall.ID, toolCall.Function.Name, result)
			conversationMessages = append(conversationMessages, toolMessage)
		}
	}

	return conversationMessages, nil
}

// StreamOptions contains options for streaming with tool support
type StreamOptions struct {
	Tools      []models.Tool
	ToolChoice models.ToolChoice
	OnChunk    func(chunk *models.ChatCompletionResponse) error
	OnToolCall func(toolCall models.ToolCall, result string) error
}

// RunStream runs the agent with streaming support
func (a *Agent) RunStream(ctx context.Context, messages []models.Message, opts StreamOptions) ([]models.Message, error) {
	conversationMessages := make([]models.Message, len(messages))
	copy(conversationMessages, messages)

	// Create request
	req := models.ChatCompletionRequest{
		Model:      a.model,
		Messages:   conversationMessages,
		Tools:      opts.Tools,
		ToolChoice: opts.ToolChoice,
		Stream:     true,
	}

	// Create stream
	stream, err := a.client.CreateChatCompletionStream(ctx, req)
	if err != nil {
		return conversationMessages, err
	}
	defer stream.Close()

	// Accumulate assistant message
	var assistantMessage models.Message
	assistantMessage.Role = models.RoleAssistant
	var contentBuilder []byte
	var toolCalls []models.ToolCall

	// Read stream
	for {
		chunk, err := stream.Read()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return conversationMessages, err
		}

		// Call chunk callback if provided
		if opts.OnChunk != nil {
			if err := opts.OnChunk(chunk); err != nil {
				return conversationMessages, err
			}
		}

		// Process chunk
		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta != nil {
			delta := chunk.Choices[0].Delta

			// Accumulate content
			if content, err := delta.GetTextContent(); err == nil && content != "" {
				contentBuilder = append(contentBuilder, content...)
			}

			// Accumulate tool calls
			if len(delta.ToolCalls) > 0 {
				toolCalls = append(toolCalls, delta.ToolCalls...)
			}
		}
	}

	// Set final content
	if len(contentBuilder) > 0 {
		contentJSON, _ := json.Marshal(string(contentBuilder))
		assistantMessage.Content = contentJSON
	}
	assistantMessage.ToolCalls = toolCalls

	// Add assistant message
	conversationMessages = append(conversationMessages, assistantMessage)

	// Execute tool calls if any
	for _, toolCall := range toolCalls {
		result, err := a.registry.Execute(toolCall)
		if err != nil {
			result = fmt.Sprintf("Error executing tool: %v", err)
		}

		// Call tool callback if provided
		if opts.OnToolCall != nil {
			if err := opts.OnToolCall(toolCall, result); err != nil {
				return conversationMessages, err
			}
		}

		// Add tool result
		toolMessage := models.NewToolMessage(toolCall.ID, toolCall.Function.Name, result)
		conversationMessages = append(conversationMessages, toolMessage)
	}

	// If there were tool calls, make another request to get the final response
	if len(toolCalls) > 0 {
		finalReq := models.ChatCompletionRequest{
			Model:    a.model,
			Messages: conversationMessages,
		}

		finalResp, err := a.client.CreateChatCompletion(ctx, finalReq)
		if err != nil {
			return conversationMessages, err
		}

		if len(finalResp.Choices) > 0 && finalResp.Choices[0].Message != nil {
			conversationMessages = append(conversationMessages, *finalResp.Choices[0].Message)
		}
	}

	return conversationMessages, nil
}