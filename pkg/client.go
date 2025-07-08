package pkg

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/rizome-dev/go-openrouter/pkg/errors"
)

const (
	// DefaultBaseURL is the default base URL for the OpenRouter API
	DefaultBaseURL = "https://openrouter.ai/api/v1"

	// DefaultTimeout is the default timeout for HTTP requests
	DefaultTimeout = 2 * time.Minute
)

// Client is the main client for interacting with the OpenRouter API
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client

	// Optional headers
	httpReferer string
	xTitle      string

	// User agent for requests
	userAgent string
}

// Option is a function that configures the client
type Option func(*Client)

// NewClient creates a new OpenRouter client with the given API key
func NewClient(apiKey string, opts ...Option) *Client {
	c := &Client{
		baseURL:   DefaultBaseURL,
		apiKey:    apiKey,
		userAgent: "openroutergo/1.0.0",
		httpClient: &http.Client{
			Timeout: DefaultTimeout,
		},
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// WithBaseURL sets a custom base URL for the API
func WithBaseURL(baseURL string) Option {
	return func(c *Client) {
		c.baseURL = baseURL
	}
}

// WithHTTPClient sets a custom HTTP client
func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) {
		c.httpClient = httpClient
	}
}

// WithTimeout sets a custom timeout for HTTP requests
func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		c.httpClient.Timeout = timeout
	}
}

// WithHTTPReferer sets the HTTP-Referer header for rankings on openrouter.ai
func WithHTTPReferer(referer string) Option {
	return func(c *Client) {
		c.httpReferer = referer
	}
}

// WithXTitle sets the X-Title header for rankings on openrouter.ai
func WithXTitle(title string) Option {
	return func(c *Client) {
		c.xTitle = title
	}
}

// WithUserAgent sets a custom user agent for requests
func WithUserAgent(userAgent string) Option {
	return func(c *Client) {
		c.userAgent = userAgent
	}
}

// doRequest performs an HTTP request with the given context
func (c *Client) doRequest(ctx context.Context, method, endpoint string, body interface{}) (*http.Response, error) {
	url := c.baseURL + endpoint

	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", c.userAgent)

	// Set optional headers
	if c.httpReferer != "" {
		req.Header.Set("HTTP-Referer", c.httpReferer)
	}
	if c.xTitle != "" {
		req.Header.Set("X-Title", c.xTitle)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to perform request: %w", err)
	}

	// Check for errors
	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		return nil, c.parseError(resp)
	}

	return resp, nil
}

// parseError parses an error response from the API
func (c *Client) parseError(resp *http.Response) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read error response: %w", err)
	}

	var errResp errors.ErrorResponse
	if err := json.Unmarshal(body, &errResp); err != nil {
		return fmt.Errorf("failed to parse error response: %w", err)
	}

	return errResp.ToError()
}
