package streaming

import (
	"context"
	"io"
	"net/http"
	"sync"
)

// CancellableReader wraps a reader with cancellation support
type CancellableReader struct {
	reader     io.ReadCloser
	ctx        context.Context
	cancelFunc context.CancelFunc
	mu         sync.Mutex
	closed     bool
	transport  http.RoundTripper
}

// NewCancellableReader creates a new cancellable reader
func NewCancellableReader(reader io.ReadCloser, ctx context.Context) *CancellableReader {
	ctx, cancel := context.WithCancel(ctx)
	return &CancellableReader{
		reader:     reader,
		ctx:        ctx,
		cancelFunc: cancel,
	}
}

// Read implements io.Reader
func (r *CancellableReader) Read(p []byte) (n int, err error) {
	// Check if context is cancelled
	select {
	case <-r.ctx.Done():
		return 0, r.ctx.Err()
	default:
	}

	// Read with cancellation check
	done := make(chan struct{})
	var readErr error
	var readN int

	go func() {
		readN, readErr = r.reader.Read(p)
		close(done)
	}()

	select {
	case <-r.ctx.Done():
		// Context cancelled, close the reader
		r.Close()
		return 0, r.ctx.Err()
	case <-done:
		return readN, readErr
	}
}

// Close closes the reader and cancels the context
func (r *CancellableReader) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return nil
	}

	r.closed = true
	r.cancelFunc()
	return r.reader.Close()
}

// Cancel cancels the stream
func (r *CancellableReader) Cancel() {
	r.cancelFunc()
	r.Close()
}

// SupportedProviders lists providers that support stream cancellation
var SupportedProviders = map[string]bool{
	"openai":       true,
	"azure":        true,
	"anthropic":    true,
	"fireworks":    true,
	"mancer":       true,
	"recursal":     true,
	"anyscale":     true,
	"lepton":       true,
	"octoai":       true,
	"novita":       true,
	"deepinfra":    true,
	"together":     true,
	"cohere":       true,
	"hyperbolic":   true,
	"infermatic":   true,
	"avian":        true,
	"xai":          true,
	"cloudflare":   true,
	"sfcompute":    true,
	"nineteen":     true,
	"liquid":       true,
	"friendli":     true,
	"chutes":       true,
	"deepseek":     true,
}

// IsProviderSupported checks if a provider supports stream cancellation
func IsProviderSupported(provider string) bool {
	return SupportedProviders[provider]
}

// StreamController provides advanced stream control
type StreamController struct {
	reader *CancellableReader
	parser *SSEParser
}

// NewStreamController creates a new stream controller
func NewStreamController(reader io.ReadCloser, ctx context.Context) *StreamController {
	cancellableReader := NewCancellableReader(reader, ctx)
	return &StreamController{
		reader: cancellableReader,
		parser: NewSSEParser(cancellableReader),
	}
}

// Read reads the next event
func (c *StreamController) Read() (*SSEEvent, error) {
	return c.parser.ParseNext()
}

// Cancel cancels the stream
func (c *StreamController) Cancel() {
	c.reader.Cancel()
}

// Close closes the stream
func (c *StreamController) Close() error {
	return c.reader.Close()
}