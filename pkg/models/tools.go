package models

import "encoding/json"

// Tool represents a tool that can be called by the model
type Tool struct {
	Type     string              `json:"type"`
	Function FunctionDescription `json:"function"`
}

// FunctionDescription describes a function that can be called
type FunctionDescription struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters"`
}

// ToolChoice represents how the model should choose tools
type ToolChoice interface {
	toolChoice()
}

// StringToolChoice represents string tool choices like "none" or "auto"
type StringToolChoice string

const (
	ToolChoiceNone StringToolChoice = "none"
	ToolChoiceAuto StringToolChoice = "auto"
)

func (StringToolChoice) toolChoice() {}

// FunctionToolChoice forces the model to call a specific function
type FunctionToolChoice struct {
	Type     string `json:"type"`
	Function struct {
		Name string `json:"name"`
	} `json:"function"`
}

func (FunctionToolChoice) toolChoice() {}

// ToolCall represents a tool call made by the model
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function FunctionCall `json:"function"`
}

// FunctionCall represents a specific function call
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// NewTool creates a new tool with the given function
func NewTool(name, description string, parameters interface{}) (*Tool, error) {
	params, err := json.Marshal(parameters)
	if err != nil {
		return nil, err
	}

	return &Tool{
		Type: "function",
		Function: FunctionDescription{
			Name:        name,
			Description: description,
			Parameters:  params,
		},
	}, nil
}

// NewFunctionToolChoice creates a tool choice that forces a specific function
func NewFunctionToolChoice(functionName string) ToolChoice {
	return FunctionToolChoice{
		Type: "function",
		Function: struct {
			Name string `json:"name"`
		}{
			Name: functionName,
		},
	}
}
