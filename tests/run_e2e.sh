#!/bin/bash

# Run OpenRouter Go SDK E2E Tests

set -e

# Check if API key is set
if [ -z "$OPENROUTER_API_KEY" ]; then
    echo "Error: OPENROUTER_API_KEY environment variable is not set"
    echo "Please set it with: export OPENROUTER_API_KEY='your-api-key'"
    exit 1
fi

echo "Running OpenRouter Go SDK E2E Tests..."
echo "=========================================="

# Run all tests with verbose output
echo "Running all E2E tests..."
go test ./tests/e2e -v -timeout 10m

# Run specific test suites if needed
if [ "$1" == "core" ]; then
    echo "Running core tests only..."
    go test ./tests/e2e -v -run "TestE2ESuite/(TestListModels|TestBasicChatCompletion|TestSystemMessage|TestMultipleMessages|TestGetGeneration|TestJSONMode)"
elif [ "$1" == "streaming" ]; then
    echo "Running streaming tests only..."
    go test ./tests/e2e -v -run "TestE2ESuite/Test.*Streaming"
elif [ "$1" == "tools" ]; then
    echo "Running tool tests only..."
    go test ./tests/e2e -v -run "TestE2ESuite/Test.*Tool"
elif [ "$1" == "structured" ]; then
    echo "Running structured output tests only..."
    go test ./tests/e2e -v -run "TestE2ESuite/Test.*Structured"
elif [ "$1" == "multimodal" ]; then
    echo "Running multimodal tests only..."
    go test ./tests/e2e -v -run "TestE2ESuite/Test.*Image|TestE2ESuite/Test.*PDF"
elif [ "$1" == "advanced" ]; then
    echo "Running advanced tests only..."
    go test ./tests/e2e -v -run "TestE2ESuite/Test.*Concurrent|TestE2ESuite/Test.*Retry|TestE2ESuite/Test.*CircuitBreaker|TestE2ESuite/Test.*Batch|TestE2ESuite/Test.*WebSearch"
fi

echo "=========================================="
echo "E2E tests completed!"