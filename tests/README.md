# OpenRouterGo E2E Tests

This directory contains end-to-end tests for the OpenRouter Go SDK.

## Running Tests

### Prerequisites

1. Set your OpenRouter API key:
```bash
export OPENROUTER_API_KEY="your-api-key"
```

2. Ensure you have sufficient credits in your OpenRouter account.

### Running All Tests

```bash
# Run all e2e tests
go test ./tests/e2e -v

# Run with timeout
go test ./tests/e2e -v -timeout 5m
```

### Running Specific Test Suites

```bash
# Basic functionality tests
go test ./tests/e2e -v -run TestE2ESuite/TestBasicChatCompletion

# Streaming tests
go test ./tests/e2e -v -run TestE2ESuite/TestBasicStreaming

# Tool calling tests
go test ./tests/e2e -v -run TestE2ESuite/TestBasicToolCalling

# Structured output tests
go test ./tests/e2e -v -run TestE2ESuite/TestStructuredOutputWithSchema
```

### Running with Coverage

```bash
go test ./tests/e2e -v -cover -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

## Test Categories

### Core Tests (`e2e_test.go`)
- List models
- Basic chat completion
- System messages
- Multiple message conversations
- Get generation details
- Provider preferences
- JSON mode

### Streaming Tests (`streaming_test.go`)
- Basic streaming
- Stream cancellation
- Streaming with tool calls
- Timeout handling

### Tool Calling Tests (`tools_test.go`)
- Basic tool calling
- Tool calling with responses
- Multiple tools
- Tool agent
- Specific tool choice

### Structured Output Tests (`structured_test.go`)
- Schema-based outputs
- Go struct integration
- Array outputs
- Nested structures
- Structured output helper

### Multimodal Tests (`multimodal_test.go`)
- Image inputs (URL and base64)
- PDF processing
- Multiple images
- Image with tools

### Advanced Tests (`advanced_test.go`)
- Concurrent requests
- Retry mechanism
- Circuit breaker
- Batch processing
- Web search
- Error handling

## Writing New Tests

1. Add tests to the appropriate file based on functionality
2. Use the test suite setup for consistent client configuration
3. Use small, fast models for basic tests (e.g., `meta-llama/llama-3.2-1b-instruct:free`)
4. Always check for errors and validate responses
5. Clean up resources and use defer for cleanup when needed

## Cost Considerations

These tests use real API calls and will consume credits. To minimize costs:

1. Use free tier models when possible
2. Keep token limits low for tests
3. Skip expensive tests in CI/CD
4. Consider mocking for unit tests

## Debugging

Enable verbose logging:
```bash
go test ./tests/e2e -v -run TestName
```

For specific test debugging:
```bash
go test ./tests/e2e -v -run TestE2ESuite/TestBasicChatCompletion -count=1
```