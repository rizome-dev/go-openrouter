# go-openrouter


[![GoDoc](https://pkg.go.dev/badge/github.com/rizome-dev/go-openrouter)](https://pkg.go.dev/github.com/rizome-dev/go-openrouter)
[![Go Report Card](https://goreportcard.com/badge/github.com/rizome-dev/go-openrouter)](https://goreportcard.com/report/github.com/rizome-dev/go-openrouter)

```bash
go get github.com/rizome-dev/go-openrouter
```

built by: [rizome labs](https://rizome.dev)

contact us: [hi (at) rizome.dev](mailto:hi@rizome.dev)

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "github.com/rizome-dev/go-openrouter/pkg/openrouter"
    "github.com/rizome-dev/go-openrouter/pkg/models"
)

func main() {
    // Create a client
    client := openrouter.NewClient("your-api-key",
        openrouter.WithHTTPReferer("https://your-app.com"),
        openrouter.WithXTitle("Your App Name"),
    )
    
    // Create a chat completion
    resp, err := client.CreateChatCompletion(context.Background(), models.ChatCompletionRequest{
        Model: "openai/gpt-4",
        Messages: []models.Message{
            models.NewTextMessage(models.RoleUser, "What is the meaning of life?"),
        },
    })
    
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println(resp.Choices[0].Message.GetTextContent())
}
```

## Advanced Usage

### Streaming Responses

```go
stream, err := client.CreateChatCompletionStream(ctx, models.ChatCompletionRequest{
    Model: "anthropic/claude-3.5-sonnet",
    Messages: []models.Message{
        models.NewTextMessage(models.RoleUser, "Write a short story"),
    },
})
if err != nil {
    log.Fatal(err)
}
defer stream.Close()

for {
    chunk, err := stream.Read()
    if err == io.EOF {
        break
    }
    if err != nil {
        log.Fatal(err)
    }
    
    if chunk.Choices[0].Delta != nil {
        content, _ := chunk.Choices[0].Delta.GetTextContent()
        fmt.Print(content)
    }
}
```

### Tool Calling

```go
// Define a tool
tool, _ := models.NewTool("search_books", 
    "Search for books in Project Gutenberg",
    map[string]interface{}{
        "type": "object",
        "properties": map[string]interface{}{
            "search_terms": map[string]interface{}{
                "type": "array",
                "items": map[string]interface{}{
                    "type": "string",
                },
                "description": "Search terms to find books",
            },
        },
        "required": []string{"search_terms"},
    },
)

// Make request with tool
resp, err := client.CreateChatCompletion(ctx, models.ChatCompletionRequest{
    Model: "openai/gpt-4",
    Messages: []models.Message{
        models.NewTextMessage(models.RoleUser, "Find books by James Joyce"),
    },
    Tools: []models.Tool{*tool},
    ToolChoice: models.ToolChoiceAuto,
})

// Handle tool calls
if resp.Choices[0].Message.ToolCalls != nil {
    for _, toolCall := range resp.Choices[0].Message.ToolCalls {
        // Execute your tool logic here
        result := executeToolCall(toolCall)
        
        // Send tool result back
        messages = append(messages, resp.Choices[0].Message)
        messages = append(messages, models.NewToolMessage(
            toolCall.ID,
            toolCall.Function.Name,
            result,
        ))
    }
}
```

### Multi-Modal Inputs

```go
// With images
imageMessage, _ := models.NewMultiContentMessage(models.RoleUser,
    models.TextContent{
        Type: models.ContentTypeText,
        Text: "What's in this image?",
    },
    models.ImageContent{
        Type: models.ContentTypeImageURL,
        ImageURL: models.ImageURL{
            URL: "https://example.com/image.jpg",
        },
    },
)

// With base64 encoded images
imageData := base64.StdEncoding.EncodeToString(imageBytes)
imageMessage, _ := models.NewMultiContentMessage(models.RoleUser,
    models.TextContent{
        Type: models.ContentTypeText,
        Text: "Analyze this image",
    },
    models.ImageContent{
        Type: models.ContentTypeImageURL,
        ImageURL: models.ImageURL{
            URL: "data:image/jpeg;base64," + imageData,
        },
    },
)

// With PDFs
pdfData := base64.StdEncoding.EncodeToString(pdfBytes)
pdfMessage, _ := models.NewMultiContentMessage(models.RoleUser,
    models.TextContent{
        Type: models.ContentTypeText,
        Text: "Summarize this document",
    },
    models.FileContent{
        Type: models.ContentTypeFile,
        File: models.File{
            Filename: "document.pdf",
            FileData: "data:application/pdf;base64," + pdfData,
        },
    },
)
```

### Provider Routing

```go
// Use specific providers with fallbacks
resp, err := client.CreateChatCompletion(ctx, models.ChatCompletionRequest{
    Model: "meta-llama/llama-3.1-70b-instruct",
    Messages: messages,
    Provider: models.NewProviderPreferences().
        WithOrder("together", "deepinfra").
        WithFallbacks(true).
        WithSort(models.SortByThroughput).
        WithMaxPrice(1.0, 2.0), // $1/M prompt, $2/M completion
})

// Use fastest available provider
resp, err := client.CreateChatCompletion(ctx, models.ChatCompletionRequest{
    Model: "meta-llama/llama-3.1-70b-instruct:nitro",
    Messages: messages,
})

// Require specific features
resp, err := client.CreateChatCompletion(ctx, models.ChatCompletionRequest{
    Model: "openai/gpt-4",
    Messages: messages,
    ResponseFormat: &models.ResponseFormat{
        Type: "json_object",
    },
    Provider: models.NewProviderPreferences().
        WithRequireParameters(true), // Only use providers supporting JSON mode
})
```

### Structured Outputs

```go
schema := map[string]interface{}{
    "type": "object",
    "properties": map[string]interface{}{
        "location": map[string]interface{}{
            "type": "string",
            "description": "City or location name",
        },
        "temperature": map[string]interface{}{
            "type": "number",
            "description": "Temperature in Celsius",
        },
        "conditions": map[string]interface{}{
            "type": "string",
            "description": "Weather conditions",
        },
    },
    "required": []string{"location", "temperature", "conditions"},
    "additionalProperties": false,
}

schemaJSON, _ := json.Marshal(schema)

resp, err := client.CreateChatCompletion(ctx, models.ChatCompletionRequest{
    Model: "openai/gpt-4",
    Messages: []models.Message{
        models.NewTextMessage(models.RoleUser, "What's the weather in London?"),
    },
    ResponseFormat: &models.ResponseFormat{
        Type: "json_schema",
        JSONSchema: &models.JSONSchema{
            Name:   "weather",
            Strict: true,
            Schema: schemaJSON,
        },
    },
})
```

### Web Search Plugin

```go
// Enable web search with :online shortcut
resp, err := client.CreateChatCompletion(ctx, models.ChatCompletionRequest{
    Model: "openai/gpt-4:online",
    Messages: []models.Message{
        models.NewTextMessage(models.RoleUser, "What happened in tech news today?"),
    },
})

// Or with plugin configuration
resp, err := client.CreateChatCompletion(ctx, models.ChatCompletionRequest{
    Model: "openai/gpt-4",
    Messages: messages,
    Plugins: []models.Plugin{
        *models.NewWebPlugin().
            WithMaxResults(10).
            WithSearchPrompt("Recent tech news:"),
    },
})

// Access web citations
for _, annotation := range resp.Choices[0].Message.Annotations {
    if annotation.Type == models.AnnotationTypeURLCitation {
        fmt.Printf("Source: %s - %s\n", 
            annotation.URLCitation.Title,
            annotation.URLCitation.URL,
        )
    }
}
```

## Error Handling

```go
resp, err := client.CreateChatCompletion(ctx, req)
if err != nil {
    if apiErr, ok := err.(*errors.APIError); ok {
        switch apiErr.Code {
        case errors.ErrorCodeRateLimited:
            // Handle rate limiting
        case errors.ErrorCodeInsufficientCredits:
            // Handle insufficient credits
        case errors.ErrorCodeForbidden:
            // Handle moderation
            if moderation, ok := apiErr.GetModerationMetadata(); ok {
                fmt.Printf("Flagged for: %v\n", moderation.Reasons)
            }
        }
    }
}
```

## Configuration Options

### Client Options

- `WithBaseURL(url)` - Use a different API endpoint
- `WithHTTPClient(client)` - Use a custom HTTP client
- `WithTimeout(duration)` - Set request timeout
- `WithHTTPReferer(referer)` - Set referer for rankings
- `WithXTitle(title)` - Set title for rankings
- `WithUserAgent(agent)` - Set custom user agent

### Request Parameters

All standard OpenAI parameters are supported:
- `temperature`, `top_p`, `top_k`
- `max_tokens`, `stop`, `seed`
- `frequency_penalty`, `presence_penalty`, `repetition_penalty`
- `logit_bias`, `top_logprobs`
- `min_p`, `top_a`

### OpenRouter-Specific Features

- Model routing with fallbacks
- Provider preferences and filtering
- Transform pipelines
- User tracking for billing
- Cost and usage tracking

## Examples

See the `/examples` directory for complete examples:
- Basic chat completion
- Streaming responses
- Tool calling and agents
- Multi-modal inputs
- Advanced routing
- Structured outputs

## Testing

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run specific test package
go test ./pkg/openrouter/...

# Run E2E tests (requires OPENROUTER_API_KEY)
cd tests
chmod +x run_e2e.sh
./run_e2e.sh
```

### Test Suite

The project includes comprehensive test coverage:

**Unit Tests** (`pkg/openrouter/`):
- `client_test.go` - Core client functionality
- `structured_test.go` - Structured output handling
- `tools_test.go` - Tool calling functionality
- `api_management_test.go` - API key management
- `errors_test.go` - Error handling
- `message_test.go` - Message construction
- `plugins_test.go` - Plugin system
- `multimodal_test.go` - Multi-modal content
- `routing_test.go` - Provider routing
- `streaming_test.go` - Streaming responses
- `response_format_test.go` - Response formatting
- `websearch_test.go` - Web search plugin
- `retry_test.go` - Retry logic

**End-to-End Tests** (`tests/e2e/`):
- `e2e_test.go` - Basic API functionality
- `advanced_test.go` - Advanced features
- `multimodal_test.go` - Multi-modal inputs
- `streaming_test.go` - Streaming responses
- `structured_test.go` - Structured outputs
- `tools_test.go` - Tool calling

### CI/CD

Tests are automatically run on GitHub Actions for all pull requests and pushes to main. The workflow includes:
- Building the project
- Running all unit tests
- Running E2E tests (when API key is available)

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the LICENSE file for details.
