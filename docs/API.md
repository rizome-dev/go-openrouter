# OpenRouterGo API Documentation

## Table of Contents

1. [Client](#client)
2. [Chat Completions](#chat-completions)
3. [Streaming](#streaming)
4. [Tool Calling](#tool-calling)
5. [Structured Outputs](#structured-outputs)
6. [Multi-Modal](#multi-modal)
7. [Web Search](#web-search)
8. [Advanced Features](#advanced-features)

## Client

### Creating a Client

```go
client := openrouter.NewClient("your-api-key",
    openrouter.WithBaseURL("https://custom.api.com"),      // Optional: custom endpoint
    openrouter.WithTimeout(30 * time.Second),              // Optional: custom timeout
    openrouter.WithHTTPReferer("https://your-app.com"),    // Optional: for rankings
    openrouter.WithXTitle("Your App Name"),                // Optional: for rankings
    openrouter.WithUserAgent("YourApp/1.0"),              // Optional: custom user agent
)
```

### Available Options

- `WithBaseURL(string)` - Set custom API endpoint
- `WithHTTPClient(*http.Client)` - Use custom HTTP client
- `WithTimeout(time.Duration)` - Set request timeout
- `WithHTTPReferer(string)` - Set HTTP referer for OpenRouter rankings
- `WithXTitle(string)` - Set app title for OpenRouter rankings
- `WithUserAgent(string)` - Set custom user agent

## Chat Completions

### Basic Usage

```go
resp, err := client.CreateChatCompletion(ctx, models.ChatCompletionRequest{
    Model: "openai/gpt-4",
    Messages: []models.Message{
        models.NewTextMessage(models.RoleSystem, "You are a helpful assistant."),
        models.NewTextMessage(models.RoleUser, "Hello!"),
    },
    Temperature: float64Ptr(0.7),
    MaxTokens:   intPtr(100),
})

content, _ := resp.Choices[0].Message.GetTextContent()
fmt.Println(content)
```

### Message Types

```go
// Text message
msg := models.NewTextMessage(models.RoleUser, "Hello!")

// Multi-content message with image
msg, _ := models.NewMultiContentMessage(models.RoleUser,
    models.TextContent{Type: models.ContentTypeText, Text: "What's in this image?"},
    models.ImageContent{
        Type: models.ContentTypeImageURL,
        ImageURL: models.ImageURL{URL: "https://example.com/image.jpg"},
    },
)

// Tool response message
msg := models.NewToolMessage("tool-call-id", "function_name", "result")
```

### Parameters

All standard OpenAI parameters are supported:

```go
req := models.ChatCompletionRequest{
    Model:             "openai/gpt-4",
    Messages:          messages,
    Temperature:       float64Ptr(0.7),      // 0-2
    TopP:              float64Ptr(0.9),      // 0-1
    TopK:              intPtr(40),           // 1+
    MaxTokens:         intPtr(100),          // 1-context_length
    Stop:              []string{"\n\n"},     // Stop sequences
    FrequencyPenalty:  float64Ptr(0.5),      // -2 to 2
    PresencePenalty:   float64Ptr(0.5),      // -2 to 2
    RepetitionPenalty: float64Ptr(1.1),      // 0-2
    Seed:              intPtr(42),           // For deterministic output
    LogitBias:         map[string]float64{"50256": -100}, // Token biases
    User:              "user-123",           // For tracking
}
```

## Streaming

### Basic Streaming

```go
stream, err := client.CreateChatCompletionStream(ctx, models.ChatCompletionRequest{
    Model: "openai/gpt-4",
    Messages: messages,
})
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

### Stream Cancellation

```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

stream, err := client.CreateChatCompletionStream(ctx, req)
// Stream will be cancelled when context is cancelled
```

### Concurrent Streaming

```go
concurrent := openrouter.NewConcurrentClient(apiKey, 5) // Max 5 concurrent

results := concurrent.CreateChatCompletionsStreamConcurrent(ctx, requests)
for result := range results {
    if result.Error != nil {
        log.Printf("Stream %d error: %v", result.Index, result.Error)
        continue
    }
    
    if result.Stream != nil {
        // Process stream chunk
    }
    
    if result.Final {
        fmt.Printf("Stream %d completed\n", result.Index)
    }
}
```

## Tool Calling

### Defining Tools

```go
tool, err := models.NewTool("get_weather",
    "Get the current weather for a location",
    map[string]interface{}{
        "type": "object",
        "properties": map[string]interface{}{
            "location": map[string]interface{}{
                "type": "string",
                "description": "City name",
            },
            "unit": map[string]interface{}{
                "type": "string",
                "enum": []string{"celsius", "fahrenheit"},
                "description": "Temperature unit",
            },
        },
        "required": []string{"location"},
    },
)
```

### Manual Tool Calling

```go
resp, err := client.CreateChatCompletion(ctx, models.ChatCompletionRequest{
    Model:      "openai/gpt-4",
    Messages:   messages,
    Tools:      []models.Tool{*tool},
    ToolChoice: models.ToolChoiceAuto, // or "none", or specific function
})

// Check for tool calls
if len(resp.Choices[0].Message.ToolCalls) > 0 {
    for _, toolCall := range resp.Choices[0].Message.ToolCalls {
        // Execute tool
        result := executeYourTool(toolCall.Function.Arguments)
        
        // Add result to conversation
        messages = append(messages, models.NewToolMessage(
            toolCall.ID,
            toolCall.Function.Name,
            result,
        ))
    }
    
    // Get final response
    finalResp, err := client.CreateChatCompletion(ctx, models.ChatCompletionRequest{
        Model:    "openai/gpt-4",
        Messages: messages,
    })
}
```

### Using the Agent

```go
agent := openrouter.NewAgent(client, "openai/gpt-4")

// Register tools
agent.RegisterToolFunc(*weatherTool, func(tc models.ToolCall) (string, error) {
    var args struct {
        Location string `json:"location"`
        Unit     string `json:"unit"`
    }
    json.Unmarshal([]byte(tc.Function.Arguments), &args)
    
    // Your tool implementation
    return getWeather(args.Location, args.Unit)
})

// Run agent
finalMessages, err := agent.Run(ctx, messages, openrouter.RunOptions{
    Tools:         []models.Tool{*weatherTool},
    ToolChoice:    models.ToolChoiceAuto,
    MaxIterations: 5,
})
```

## Structured Outputs

### Using JSON Schema

```go
structured := openrouter.NewStructuredOutput(client)

// Define schema
schema := map[string]interface{}{
    "type": "object",
    "properties": map[string]interface{}{
        "summary": map[string]interface{}{
            "type": "string",
            "description": "Brief summary",
        },
        "key_points": map[string]interface{}{
            "type": "array",
            "items": map[string]interface{}{
                "type": "string",
            },
            "description": "Key points",
        },
        "sentiment": map[string]interface{}{
            "type": "string",
            "enum": []string{"positive", "negative", "neutral"},
        },
    },
    "required": []string{"summary", "key_points", "sentiment"},
}

resp, err := structured.CreateWithSchema(ctx,
    models.ChatCompletionRequest{
        Model: "openai/gpt-4",
        Messages: messages,
    },
    "analysis_result",
    schema,
)
```

### Using Go Structs

```go
type Analysis struct {
    Summary    string   `json:"summary" description:"Brief summary"`
    KeyPoints  []string `json:"key_points" description:"Key points"`
    Sentiment  string   `json:"sentiment" description:"positive, negative, or neutral"`
}

// Generate schema from struct
resp, err := structured.CreateWithSchema(ctx, req, "analysis", Analysis{})

// Parse response
var result Analysis
err = openrouter.ParseStructuredResponse(resp, &result)
```

## Multi-Modal

### Images

```go
helper := openrouter.NewMultiModalHelper(client)

// From URL
resp, err := helper.CreateWithImage(ctx,
    "What's in this image?",
    openrouter.ImageInput{URL: "https://example.com/image.jpg"},
    "google/gemini-2.0-flash-001",
)

// From file
image, _ := openrouter.LoadImageFromFile("photo.jpg")
resp, err := helper.CreateWithImage(ctx, "Describe this", image, "openai/gpt-4-vision")

// Multiple images
resp, err := helper.CreateWithImages(ctx,
    "Compare these images",
    []openrouter.ImageInput{
        {URL: "https://example.com/1.jpg"},
        {Path: "local/2.jpg"},
    },
    "openai/gpt-4-vision",
)
```

### PDFs

```go
// Load PDF
pdf, _ := openrouter.LoadPDFFromFile("document.pdf", models.PDFEngineText)

// Process PDF
resp, err := helper.CreateWithPDF(ctx,
    "Summarize this document",
    pdf,
    "google/gemini-2.0-flash-001",
)

// Reuse annotations to save costs
annotations := resp.Choices[0].Message.Annotations
// Use annotations in subsequent requests
```

## Web Search

### Basic Web Search

```go
webHelper := openrouter.NewWebSearchHelper(client)

// Simple search (using :online shortcut)
resp, err := webHelper.CreateWithWebSearch(ctx,
    "Latest AI news",
    "openai/gpt-4",
    nil,
)

// With options
resp, err := webHelper.CreateWithWebSearch(ctx,
    "Quantum computing breakthroughs",
    "openai/gpt-4",
    &openrouter.SearchOptions{
        MaxResults:   10,
        SearchPrompt: "Recent research papers:",
    },
)

// Extract citations
citations := openrouter.ExtractCitations(resp)
for _, c := range citations {
    fmt.Printf("%s: %s\n", c.Title, c.URL)
}
```

### Research Agent

```go
agent := webHelper.CreateResearchAgent("openai/gpt-4")

research, err := agent.Research(ctx, "Future of renewable energy", 5) // 5 subtopics

fmt.Println(research.Summary)
for _, section := range research.Sections {
    fmt.Printf("\n%s\n%s\n", section.Title, section.Content)
}
```

## Advanced Features

### Provider Routing

```go
// Specific provider order
provider := models.NewProviderPreferences().
    WithOrder("anthropic", "openai", "together").
    WithFallbacks(true)

// Price constraints
provider := models.NewProviderPreferences().
    WithMaxPrice(0.01, 0.02).  // $0.01/M prompt, $0.02/M completion
    WithSort(models.SortByPrice)

// Quantization filtering
provider := models.NewProviderPreferences().
    WithQuantizations(models.QuantizationFP16, models.QuantizationBF16)

// Data privacy
provider := models.NewProviderPreferences().
    WithDataCollection(models.DataCollectionDeny).
    WithRequireParameters(true)  // Only providers supporting all params

req.Provider = provider
```

### Retry and Circuit Breaker

```go
// Retry client
retryClient := openrouter.NewRetryClient(apiKey,
    &openrouter.RetryConfig{
        MaxRetries:    5,
        InitialDelay:  1 * time.Second,
        MaxDelay:      30 * time.Second,
        BackoffFactor: 2.0,
    },
)

// Circuit breaker
breaker := openrouter.NewCircuitBreaker(client,
    5,                    // Failure threshold
    30 * time.Second,     // Reset timeout
)
```

### Observability

```go
// Create observable client
obsClient := openrouter.NewObservableClient(apiKey,
    openrouter.ObservabilityOptions{
        Logger:       logger,
        Metrics:      metricsCollector,
        LogRequests:  true,
        LogResponses: true,
        TrackCosts:   true,
    },
)

// Add hooks
obsClient.AddRequestHook(func(ctx context.Context, op string, req interface{}) context.Context {
    // Pre-request logic
    return ctx
})

obsClient.AddResponseHook(func(ctx context.Context, op string, req, resp interface{}, err error) {
    // Post-response logic
})
```

### Batch Processing

```go
processor := openrouter.NewBatchProcessor(
    openrouter.NewConcurrentClient(apiKey, 10),
    5, // Batch size
)

err := processor.ProcessBatch(ctx, requests, func(result openrouter.ChatCompletionResult) {
    if result.Error != nil {
        log.Printf("Request %d failed: %v", result.Index, result.Error)
    } else {
        // Process result
    }
})
```

## Error Handling

```go
resp, err := client.CreateChatCompletion(ctx, req)
if err != nil {
    if apiErr, ok := err.(*errors.APIError); ok {
        switch apiErr.Code {
        case errors.ErrorCodeRateLimited:
            // Wait and retry
        case errors.ErrorCodeInsufficientCredits:
            // Add credits
        case errors.ErrorCodeForbidden:
            if mod, ok := apiErr.GetModerationMetadata(); ok {
                fmt.Printf("Flagged: %v\n", mod.Reasons)
            }
        case errors.ErrorCodeModelDown:
            // Try different model/provider
        }
    }
}
```

## Best Practices

1. **Always use context** for cancellation and timeouts
2. **Handle errors gracefully** - check error types and codes
3. **Use streaming for long responses** to improve UX
4. **Implement retry logic** for production applications
5. **Track costs** using the generation endpoint or metrics
6. **Use provider preferences** to optimize for your needs
7. **Cache responses** when appropriate
8. **Set reasonable timeouts** based on model and request size
9. **Use structured outputs** for reliable parsing
10. **Monitor usage** with observability features