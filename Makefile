.PHONY: test test-unit test-e2e test-e2e-core test-e2e-streaming test-e2e-tools test-e2e-structured test-e2e-multimodal test-e2e-advanced test-coverage lint fmt vet

# Run all tests
test: test-unit test-e2e

# Run unit tests
test-unit:
	@echo "Running unit tests..."
	@go test ./pkg/... -v -race

# Run all E2E tests
test-e2e:
	@echo "Running E2E tests..."
	@if [ -z "$$OPENROUTER_API_KEY" ]; then \
		echo "Error: OPENROUTER_API_KEY not set"; \
		exit 1; \
	fi
	@go test ./tests/e2e -v -timeout 10m

# Run core E2E tests only
test-e2e-core:
	@echo "Running core E2E tests..."
	@go test ./tests/e2e -v -timeout 5m -run "TestE2ESuite/(TestListModels|TestBasicChatCompletion|TestSystemMessage|TestMultipleMessages|TestGetGeneration|TestJSONMode)"

# Run streaming E2E tests only
test-e2e-streaming:
	@echo "Running streaming E2E tests..."
	@go test ./tests/e2e -v -timeout 5m -run "TestE2ESuite/Test.*Streaming"

# Run tool calling E2E tests only
test-e2e-tools:
	@echo "Running tool calling E2E tests..."
	@go test ./tests/e2e -v -timeout 5m -run "TestE2ESuite/Test.*Tool"

# Run structured output E2E tests only
test-e2e-structured:
	@echo "Running structured output E2E tests..."
	@go test ./tests/e2e -v -timeout 5m -run "TestE2ESuite/Test.*Structured"

# Run multimodal E2E tests only
test-e2e-multimodal:
	@echo "Running multimodal E2E tests..."
	@go test ./tests/e2e -v -timeout 5m -run "TestE2ESuite/Test.*Image|TestE2ESuite/Test.*PDF"

# Run advanced E2E tests only
test-e2e-advanced:
	@echo "Running advanced E2E tests..."
	@go test ./tests/e2e -v -timeout 5m -run "TestE2ESuite/Test.*Concurrent|TestE2ESuite/Test.*Retry|TestE2ESuite/Test.*CircuitBreaker|TestE2ESuite/Test.*Batch|TestE2ESuite/Test.*WebSearch"

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@go test ./pkg/... -race -coverprofile=coverage.out -covermode=atomic
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Lint the code
lint:
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed. Install with: curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin"; \
		exit 1; \
	fi

# Format the code
fmt:
	@echo "Formatting code..."
	@go fmt ./...
	@gofmt -s -w .

# Run go vet
vet:
	@echo "Running go vet..."
	@go vet ./...

# Clean test cache and coverage files
clean:
	@echo "Cleaning..."
	@go clean -testcache
	@rm -f coverage.out coverage.html

# Install dependencies
deps:
	@echo "Installing dependencies..."
	@go mod download
	@go mod tidy

# Build the project
build:
	@echo "Building..."
	@go build -v ./...

# Run a quick check (format, vet, and unit tests)
check: fmt vet test-unit