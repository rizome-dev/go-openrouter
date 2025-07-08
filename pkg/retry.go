package pkg

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/rizome-dev/go-openrouter/pkg/errors"
	"github.com/rizome-dev/go-openrouter/pkg/models"
	"github.com/rizome-dev/go-openrouter/pkg/streaming"
)

// RetryConfig represents retry configuration
type RetryConfig struct {
	MaxRetries      int
	InitialDelay    time.Duration
	MaxDelay        time.Duration
	BackoffFactor   float64
	JitterFactor    float64
	RetryableErrors map[errors.ErrorCode]bool
}

// DefaultRetryConfig returns default retry configuration
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxRetries:    3,
		InitialDelay:  1 * time.Second,
		MaxDelay:      30 * time.Second,
		BackoffFactor: 2.0,
		JitterFactor:  0.1,
		RetryableErrors: map[errors.ErrorCode]bool{
			errors.ErrorCodeTimeout:          true,
			errors.ErrorCodeRateLimited:      true,
			errors.ErrorCodeModelDown:        true,
			errors.ErrorCodeNoAvailableModel: true,
		},
	}
}

// RetryClient wraps a client with retry logic
type RetryClient struct {
	*Client
	config *RetryConfig
}

// NewRetryClient creates a new retry client
func NewRetryClient(apiKey string, retryConfig *RetryConfig, opts ...Option) *RetryClient {
	if retryConfig == nil {
		retryConfig = DefaultRetryConfig()
	}

	return &RetryClient{
		Client: NewClient(apiKey, opts...),
		config: retryConfig,
	}
}

// CreateChatCompletion creates a chat completion with retry logic
func (r *RetryClient) CreateChatCompletion(ctx context.Context, req models.ChatCompletionRequest) (*models.ChatCompletionResponse, error) {
	var lastErr error

	for attempt := 0; attempt <= r.config.MaxRetries; attempt++ {
		// Calculate delay for this attempt
		if attempt > 0 {
			delay := r.calculateDelay(attempt)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}

		// Make request
		resp, err := r.Client.CreateChatCompletion(ctx, req)
		if err == nil {
			return resp, nil
		}

		lastErr = err

		// Check if error is retryable
		if !r.isRetryable(err) {
			return nil, err
		}

		// Log retry attempt
		if attempt < r.config.MaxRetries {
			fmt.Printf("Retry attempt %d/%d after error: %v\n", attempt+1, r.config.MaxRetries, err)
		}
	}

	return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}

// CreateChatCompletionStream creates a streaming chat completion with retry logic
func (r *RetryClient) CreateChatCompletionStream(ctx context.Context, req models.ChatCompletionRequest) (*streaming.ChatCompletionStreamReader, error) {
	// Streaming requests don't support retries due to the nature of the stream
	return nil, fmt.Errorf("retry not supported for streaming")
}

// calculateDelay calculates the delay for a given attempt
func (r *RetryClient) calculateDelay(attempt int) time.Duration {
	// Exponential backoff
	delay := float64(r.config.InitialDelay) * math.Pow(r.config.BackoffFactor, float64(attempt-1))

	// Apply max delay
	if delay > float64(r.config.MaxDelay) {
		delay = float64(r.config.MaxDelay)
	}

	// Apply jitter
	jitter := delay * r.config.JitterFactor * (2*rand.Float64() - 1)
	delay += jitter

	return time.Duration(delay)
}

// isRetryable checks if an error is retryable
func (r *RetryClient) isRetryable(err error) bool {
	apiErr, ok := err.(*errors.APIError)
	if !ok {
		return false
	}

	return r.config.RetryableErrors[apiErr.Code]
}

// CircuitBreaker implements circuit breaker pattern
type CircuitBreaker struct {
	client           *Client
	failureThreshold int
	resetTimeout     time.Duration

	failures    int
	lastFailure time.Time
	state       CircuitState
}

// CircuitState represents the state of a circuit breaker
type CircuitState int

const (
	CircuitClosed CircuitState = iota
	CircuitOpen
	CircuitHalfOpen
)

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(client *Client, failureThreshold int, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		client:           client,
		failureThreshold: failureThreshold,
		resetTimeout:     resetTimeout,
		state:            CircuitClosed,
	}
}

// CreateChatCompletion creates a chat completion with circuit breaker
func (cb *CircuitBreaker) CreateChatCompletion(ctx context.Context, req models.ChatCompletionRequest) (*models.ChatCompletionResponse, error) {
	// Check circuit state
	if err := cb.checkState(); err != nil {
		return nil, err
	}

	// Make request
	resp, err := cb.client.CreateChatCompletion(ctx, req)

	// Update circuit state based on result
	cb.recordResult(err)

	return resp, err
}

// checkState checks if the circuit allows requests
func (cb *CircuitBreaker) checkState() error {
	switch cb.state {
	case CircuitOpen:
		// Check if we should transition to half-open
		if time.Since(cb.lastFailure) > cb.resetTimeout {
			cb.state = CircuitHalfOpen
			cb.failures = 0
		} else {
			return fmt.Errorf("circuit breaker is open")
		}
	}
	return nil
}

// recordResult records the result of a request
func (cb *CircuitBreaker) recordResult(err error) {
	if err == nil {
		// Success
		if cb.state == CircuitHalfOpen {
			cb.state = CircuitClosed
		}
		cb.failures = 0
	} else {
		// Failure
		cb.failures++
		cb.lastFailure = time.Now()

		if cb.failures >= cb.failureThreshold {
			cb.state = CircuitOpen
		}
	}
}
