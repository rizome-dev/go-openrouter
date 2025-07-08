package e2e

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rizome-dev/go-openrouter/pkg/errors"
	"github.com/rizome-dev/go-openrouter/pkg/models"
	"github.com/rizome-dev/go-openrouter/pkg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (suite *E2ETestSuite) TestConcurrentRequests() {
	ctx := context.Background()

	// Create concurrent client with limit
	concurrent := pkg.NewConcurrentClient(suite.apiKey, 3)

	// Create multiple requests
	requests := []models.ChatCompletionRequest{}
	for i := 0; i < 5; i++ {
		requests = append(requests, models.ChatCompletionRequest{
			Model: "mistralai/mistral-small-3.2-24b-instruct:free",
			Messages: []models.Message{
				models.NewTextMessage(models.RoleUser, fmt.Sprintf("Say 'Response %d' and nothing else", i+1)),
			},
			MaxTokens:   intPtr(10),
			Temperature: float64Ptr(0.0),
		})
	}

	// Execute concurrently
	results := concurrent.CreateChatCompletionsConcurrent(ctx, requests)

	successCount := 0
	errorCount := 0
	responses := make(map[int]string)

	for _, result := range results {
		if result.Error != nil {
			errorCount++
			suite.T().Logf("Request %d failed: %v", result.Index, result.Error)
		} else {
			successCount++
			content, err := result.Response.Choices[0].Message.GetTextContent()
			assert.NoError(suite.T(), err)
			responses[result.Index] = content
		}
	}

	// Should have completed all requests
	assert.Equal(suite.T(), len(requests), successCount+errorCount)
	assert.Equal(suite.T(), len(requests), successCount)

	// Verify responses match request indices
	for i := 0; i < len(requests); i++ {
		assert.Contains(suite.T(), responses[i], fmt.Sprintf("%d", i+1))
	}
}

func (suite *E2ETestSuite) TestConcurrentStreaming() {
	ctx := context.Background()

	concurrent := pkg.NewConcurrentClient(suite.apiKey, 2)

	// Create streaming requests
	requests := []models.ChatCompletionRequest{
		{
			Model: "mistralai/mistral-small-3.2-24b-instruct:free",
			Messages: []models.Message{
				models.NewTextMessage(models.RoleUser, "Count from 1 to 3"),
			},
			MaxTokens: intPtr(20),
			Stream:    true,
		},
		{
			Model: "mistralai/mistral-small-3.2-24b-instruct:free",
			Messages: []models.Message{
				models.NewTextMessage(models.RoleUser, "Say hello"),
			},
			MaxTokens: intPtr(10),
			Stream:    true,
		},
	}

	results := concurrent.CreateChatCompletionsStreamConcurrent(ctx, requests)

	streamContent := make(map[int]string)
	completedStreams := 0
	var mu sync.Mutex

	for result := range results {
		if result.Error != nil {
			suite.T().Logf("Stream %d error: %v", result.Index, result.Error)
			continue
		}

		if result.Stream != nil {
			if result.Stream.Choices != nil && len(result.Stream.Choices) > 0 {
				if result.Stream.Choices[0].Delta != nil {
					content, _ := result.Stream.Choices[0].Delta.GetTextContent()
					mu.Lock()
					streamContent[result.Index] += content
					mu.Unlock()
				}
			}
		}

		if result.Final {
			mu.Lock()
			completedStreams++
			mu.Unlock()
		}
	}

	// Both streams should complete
	assert.Equal(suite.T(), 2, completedStreams)
	assert.NotEmpty(suite.T(), streamContent[0])
	assert.NotEmpty(suite.T(), streamContent[1])
}

func (suite *E2ETestSuite) TestRetryClient() {
	ctx := context.Background()

	// Create retry client with aggressive settings for testing
	retryClient := pkg.NewRetryClient(suite.apiKey,
		&pkg.RetryConfig{
			MaxRetries:    3,
			InitialDelay:  100 * time.Millisecond,
			MaxDelay:      1 * time.Second,
			BackoffFactor: 2.0,
		},
	)

	// Test successful request (should work on first try)
	req := models.ChatCompletionRequest{
		Model: "mistralai/mistral-small-3.2-24b-instruct:free",
		Messages: []models.Message{
			models.NewTextMessage(models.RoleUser, "Hello"),
		},
		MaxTokens: intPtr(10),
	}

	resp, err := retryClient.CreateChatCompletion(ctx, req)
	require.NoError(suite.T(), err)
	assert.NotNil(suite.T(), resp)

	// Add delay before next request
	time.Sleep(1 * time.Second)

	// Test with invalid model (should retry and fail)
	req.Model = "invalid/model-that-does-not-exist"
	startTime := time.Now()

	_, err = retryClient.CreateChatCompletion(ctx, req)
	assert.Error(suite.T(), err)

	// Should have taken some time due to retries
	elapsed := time.Since(startTime)
	assert.Greater(suite.T(), elapsed, 100*time.Millisecond)
}

func (suite *E2ETestSuite) TestWebSearch() {
	ctx := context.Background()

	webHelper := pkg.NewWebSearchHelper(suite.client)

	// Test basic web search
	resp, err := webHelper.CreateWithWebSearch(ctx,
		"What is the current version of Go programming language?",
		"google/gemini-2.5-flash",
		&pkg.SearchOptions{
			MaxResults: 5,
		},
	)
	require.NoError(suite.T(), err)

	content, err := resp.Choices[0].Message.GetTextContent()
	assert.NoError(suite.T(), err)
	assert.NotEmpty(suite.T(), content)

	// Should mention Go and version
	assert.Contains(suite.T(), strings.ToLower(content), "go")

	// Check for citations if available
	if resp.Choices[0].Message.Annotations != nil {
		citations := pkg.ExtractCitations(resp)
		if len(citations) > 0 {
			suite.T().Logf("Found %d citations", len(citations))
			for _, c := range citations {
				assert.NotEmpty(suite.T(), c.URL)
			}
		}
	}
}

func (suite *E2ETestSuite) TestBatchProcessor() {
	ctx := context.Background()

	// Create batch processor
	concurrent := pkg.NewConcurrentClient(suite.apiKey, 2)
	processor := pkg.NewBatchProcessor(concurrent, 2)

	// Create batch of requests
	requests := []models.ChatCompletionRequest{}
	for i := 0; i < 5; i++ {
		requests = append(requests, models.ChatCompletionRequest{
			Model: "mistralai/mistral-small-3.2-24b-instruct:free",
			Messages: []models.Message{
				models.NewTextMessage(models.RoleUser, fmt.Sprintf("Reply with just the number %d", i)),
			},
			MaxTokens:   intPtr(5),
			Temperature: float64Ptr(0.0),
		})
	}

	results := []pkg.ChatCompletionResult{}
	var mu sync.Mutex

	err := processor.ProcessBatch(ctx, requests, func(result pkg.ChatCompletionResult) {
		mu.Lock()
		results = append(results, result)
		mu.Unlock()
	})

	require.NoError(suite.T(), err)
	assert.Len(suite.T(), results, 5)

	// Verify all succeeded
	for _, result := range results {
		assert.NoError(suite.T(), result.Error)
		assert.NotNil(suite.T(), result.Response)
	}
}

func (suite *E2ETestSuite) TestErrorHandling() {
	ctx := context.Background()

	testCases := []struct {
		name          string
		model         string
		messages      []models.Message
		expectedError errors.ErrorCode
	}{
		{
			name:  "Invalid Model",
			model: "invalid/non-existent-model",
			messages: []models.Message{
				models.NewTextMessage(models.RoleUser, "Hello"),
			},
			expectedError: errors.ErrorCodeBadRequest,
		},
		{
			name:          "Empty Messages",
			model:         "mistralai/mistral-small-3.2-24b-instruct:free",
			messages:      []models.Message{},
			expectedError: errors.ErrorCodeBadRequest,
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			req := models.ChatCompletionRequest{
				Model:    tc.model,
				Messages: tc.messages,
			}

			_, err := suite.client.CreateChatCompletion(ctx, req)
			require.Error(t, err)

			apiErr, ok := err.(*errors.APIError)
			if ok {
				assert.Equal(t, tc.expectedError, apiErr.Code)
			}
		})
	}
}

func (suite *E2ETestSuite) TestProviderRouting() {
	ctx := context.Background()

	// Test with specific provider preferences
	provider := models.NewProviderPreferences().
		WithFallbacks(true).
		WithSort(models.SortByThroughput).
		WithDataCollection(models.DataCollectionAllow)

	req := models.ChatCompletionRequest{
		Model: "mistralai/mistral-small-3.2-24b-instruct:free",
		Messages: []models.Message{
			models.NewTextMessage(models.RoleUser, "Hello"),
		},
		MaxTokens: intPtr(10),
		Provider:  provider,
	}

	resp, err := suite.client.CreateChatCompletion(ctx, req)
	require.NoError(suite.T(), err)

	// Wait before getting generation details
	time.Sleep(2 * time.Second)

	// Get generation to see which provider was used
	gen, err := suite.client.GetGeneration(ctx, resp.ID)
	require.NoError(suite.T(), err)

	// Provider field may be empty in some cases, just verify we got the generation data
	assert.NotEmpty(suite.T(), gen.Data.Model)
	if gen.Data.Provider != "" {
		suite.T().Logf("Used provider: %s", gen.Data.Provider)
	}
}

func (suite *E2ETestSuite) TestCircuitBreaker() {
	ctx := context.Background()

	// Create circuit breaker with low threshold for testing
	breaker := pkg.NewCircuitBreaker(suite.client, 2, 5*time.Second)

	// Make valid requests
	for i := 0; i < 2; i++ {
		resp, err := breaker.CreateChatCompletion(ctx, models.ChatCompletionRequest{
			Model: "mistralai/mistral-small-3.2-24b-instruct:free",
			Messages: []models.Message{
				models.NewTextMessage(models.RoleUser, "Hello"),
			},
			MaxTokens: intPtr(10),
		})
		require.NoError(suite.T(), err)
		assert.NotNil(suite.T(), resp)
		// Add delay between requests
		time.Sleep(500 * time.Millisecond)
	}

	// Make failing requests
	for i := 0; i < 2; i++ {
		_, err := breaker.CreateChatCompletion(ctx, models.ChatCompletionRequest{
			Model: "invalid/model",
			Messages: []models.Message{
				models.NewTextMessage(models.RoleUser, "Hello"),
			},
		})
		assert.Error(suite.T(), err)
		// Add delay between failing requests
		time.Sleep(500 * time.Millisecond)
	}

	// Next request should fail immediately
	_, err := breaker.CreateChatCompletion(ctx, models.ChatCompletionRequest{
		Model: "mistralai/mistral-small-3.2-24b-instruct:free",
		Messages: []models.Message{
			models.NewTextMessage(models.RoleUser, "Hello"),
		},
	})
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "circuit breaker is open")
}
