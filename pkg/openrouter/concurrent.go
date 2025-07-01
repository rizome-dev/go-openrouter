package openrouter

import (
	"context"
	"sync"

	"github.com/rizome-dev/go-openrouter/pkg/models"
)

// ConcurrentClient wraps Client with concurrent execution capabilities
type ConcurrentClient struct {
	*Client
	maxConcurrency int
	semaphore      chan struct{}
}

// NewConcurrentClient creates a new concurrent client
func NewConcurrentClient(apiKey string, maxConcurrency int, opts ...Option) *ConcurrentClient {
	if maxConcurrency <= 0 {
		maxConcurrency = 10 // Default concurrency
	}

	return &ConcurrentClient{
		Client:         NewClient(apiKey, opts...),
		maxConcurrency: maxConcurrency,
		semaphore:      make(chan struct{}, maxConcurrency),
	}
}

// ChatCompletionResult represents the result of a concurrent chat completion
type ChatCompletionResult struct {
	Response *models.ChatCompletionResponse
	Error    error
	Index    int
}

// CreateChatCompletionsConcurrent executes multiple chat completions concurrently
func (c *ConcurrentClient) CreateChatCompletionsConcurrent(ctx context.Context, requests []models.ChatCompletionRequest) []ChatCompletionResult {
	results := make([]ChatCompletionResult, len(requests))
	var wg sync.WaitGroup

	for i, req := range requests {
		wg.Add(1)
		go func(index int, request models.ChatCompletionRequest) {
			defer wg.Done()

			// Acquire semaphore
			select {
			case c.semaphore <- struct{}{}:
				defer func() { <-c.semaphore }()
			case <-ctx.Done():
				results[index] = ChatCompletionResult{
					Error: ctx.Err(),
					Index: index,
				}
				return
			}

			// Execute request
			resp, err := c.CreateChatCompletion(ctx, request)
			results[index] = ChatCompletionResult{
				Response: resp,
				Error:    err,
				Index:    index,
			}
		}(i, req)
	}

	wg.Wait()
	return results
}

// StreamingResult represents a streaming result
type StreamingResult struct {
	Stream *models.ChatCompletionResponse
	Error  error
	Index  int
	Final  bool
}

// CreateChatCompletionsStreamConcurrent executes multiple streaming chat completions concurrently
func (c *ConcurrentClient) CreateChatCompletionsStreamConcurrent(ctx context.Context, requests []models.ChatCompletionRequest) <-chan StreamingResult {
	resultChan := make(chan StreamingResult, len(requests)*10) // Buffer for performance

	go func() {
		defer close(resultChan)
		var wg sync.WaitGroup

		for i, req := range requests {
			wg.Add(1)
			go func(index int, request models.ChatCompletionRequest) {
				defer wg.Done()

				// Acquire semaphore
				select {
				case c.semaphore <- struct{}{}:
					defer func() { <-c.semaphore }()
				case <-ctx.Done():
					resultChan <- StreamingResult{
						Error: ctx.Err(),
						Index: index,
						Final: true,
					}
					return
				}

				// Create stream
				stream, err := c.CreateChatCompletionStream(ctx, request)
				if err != nil {
					resultChan <- StreamingResult{
						Error: err,
						Index: index,
						Final: true,
					}
					return
				}
				defer stream.Close()

				// Read stream
				for {
					chunk, err := stream.Read()
					if err != nil {
						if err.Error() == "EOF" {
							resultChan <- StreamingResult{
								Index: index,
								Final: true,
							}
						} else {
							resultChan <- StreamingResult{
								Error: err,
								Index: index,
								Final: true,
							}
						}
						break
					}

					resultChan <- StreamingResult{
						Stream: chunk,
						Index:  index,
						Final:  false,
					}
				}
			}(i, req)
		}

		wg.Wait()
	}()

	return resultChan
}

// BatchProcessor processes requests in batches
type BatchProcessor struct {
	client    *ConcurrentClient
	batchSize int
}

// NewBatchProcessor creates a new batch processor
func NewBatchProcessor(client *ConcurrentClient, batchSize int) *BatchProcessor {
	if batchSize <= 0 {
		batchSize = 5
	}
	return &BatchProcessor{
		client:    client,
		batchSize: batchSize,
	}
}

// ProcessBatch processes requests in batches and calls the callback for each result
func (p *BatchProcessor) ProcessBatch(ctx context.Context, requests []models.ChatCompletionRequest, callback func(ChatCompletionResult)) error {
	for i := 0; i < len(requests); i += p.batchSize {
		end := i + p.batchSize
		if end > len(requests) {
			end = len(requests)
		}

		batch := requests[i:end]
		results := p.client.CreateChatCompletionsConcurrent(ctx, batch)

		for _, result := range results {
			callback(result)
		}

		// Check context
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}

	return nil
}
