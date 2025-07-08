package pkg

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/rizome-dev/go-openrouter/pkg/models"
)

// Logger interface for custom logging
type Logger interface {
	Debug(msg string, fields ...interface{})
	Info(msg string, fields ...interface{})
	Warn(msg string, fields ...interface{})
	Error(msg string, fields ...interface{})
}

// MetricsCollector interface for metrics collection
type MetricsCollector interface {
	RecordLatency(operation string, duration time.Duration, labels map[string]string)
	RecordTokens(promptTokens, completionTokens int, labels map[string]string)
	RecordCost(cost float64, labels map[string]string)
	RecordError(operation string, err error, labels map[string]string)
}

// RequestHook is called before a request is made
type RequestHook func(ctx context.Context, operation string, request interface{}) context.Context

// ResponseHook is called after a response is received
type ResponseHook func(ctx context.Context, operation string, request interface{}, response interface{}, err error)

// ObservableClient wraps a client with observability features
type ObservableClient struct {
	*Client
	logger        Logger
	metrics       MetricsCollector
	requestHooks  []RequestHook
	responseHooks []ResponseHook
	logRequests   bool
	logResponses  bool
	trackCosts    bool
}

// ObservabilityOptions contains options for observability
type ObservabilityOptions struct {
	Logger       Logger
	Metrics      MetricsCollector
	LogRequests  bool
	LogResponses bool
	TrackCosts   bool
}

// NewObservableClient creates a new observable client
func NewObservableClient(apiKey string, obsOpts ObservabilityOptions, clientOpts ...Option) *ObservableClient {
	return &ObservableClient{
		Client:       NewClient(apiKey, clientOpts...),
		logger:       obsOpts.Logger,
		metrics:      obsOpts.Metrics,
		logRequests:  obsOpts.LogRequests,
		logResponses: obsOpts.LogResponses,
		trackCosts:   obsOpts.TrackCosts,
	}
}

// AddRequestHook adds a request hook
func (o *ObservableClient) AddRequestHook(hook RequestHook) {
	o.requestHooks = append(o.requestHooks, hook)
}

// AddResponseHook adds a response hook
func (o *ObservableClient) AddResponseHook(hook ResponseHook) {
	o.responseHooks = append(o.responseHooks, hook)
}

// CreateChatCompletion creates a chat completion with observability
func (o *ObservableClient) CreateChatCompletion(ctx context.Context, req models.ChatCompletionRequest) (*models.ChatCompletionResponse, error) {
	start := time.Now()
	operation := "chat_completion"

	// Run request hooks
	for _, hook := range o.requestHooks {
		ctx = hook(ctx, operation, req)
	}

	// Log request if enabled
	if o.logRequests && o.logger != nil {
		o.logger.Info("Creating chat completion",
			"model", req.Model,
			"messages", len(req.Messages),
			"stream", req.Stream,
		)
	}

	// Make request
	resp, err := o.Client.CreateChatCompletion(ctx, req)

	// Calculate metrics
	duration := time.Since(start)
	labels := map[string]string{
		"model":     req.Model,
		"operation": operation,
		"status":    "success",
	}

	if err != nil {
		labels["status"] = "error"
		if o.metrics != nil {
			o.metrics.RecordError(operation, err, labels)
		}
		if o.logger != nil {
			o.logger.Error("Chat completion failed",
				"error", err,
				"model", req.Model,
				"duration", duration,
			)
		}
	} else {
		// Log response if enabled
		if o.logResponses && o.logger != nil {
			o.logger.Info("Chat completion succeeded",
				"model", resp.Model,
				"duration", duration,
				"choices", len(resp.Choices),
			)
		}

		// Record metrics
		if o.metrics != nil {
			o.metrics.RecordLatency(operation, duration, labels)

			if resp.Usage != nil {
				o.metrics.RecordTokens(
					resp.Usage.PromptTokens,
					resp.Usage.CompletionTokens,
					labels,
				)
			}
		}

		// Track costs if enabled
		if o.trackCosts && resp.ID != "" {
			go o.trackGenerationCost(ctx, resp.ID, labels)
		}
	}

	// Run response hooks
	for _, hook := range o.responseHooks {
		hook(ctx, operation, req, resp, err)
	}

	return resp, err
}

// trackGenerationCost tracks the cost of a generation
func (o *ObservableClient) trackGenerationCost(ctx context.Context, generationID string, labels map[string]string) {
	// Wait a bit for generation to be processed
	time.Sleep(2 * time.Second)

	genResp, err := o.Client.GetGeneration(ctx, generationID)
	if err != nil {
		if o.logger != nil {
			o.logger.Warn("Failed to get generation cost",
				"generation_id", generationID,
				"error", err,
			)
		}
		return
	}

	// Handle usage field which may be interface{} due to API inconsistencies
	var totalCost float64
	if usage, ok := genResp.Data.Usage.(map[string]interface{}); ok {
		if cost, exists := usage["total_cost"]; exists {
			if costFloat, ok := cost.(float64); ok {
				totalCost = costFloat
			}
		}
	}

	if o.metrics != nil && totalCost > 0 {
		o.metrics.RecordCost(totalCost, labels)
	}

	if o.logger != nil {
		o.logger.Debug("Generation cost tracked",
			"generation_id", generationID,
			"cost", totalCost,
			"prompt_tokens", genResp.Data.NativeTokenCounts.PromptTokens,
			"completion_tokens", genResp.Data.NativeTokenCounts.CompletionTokens,
		)
	}
}

// SimpleLogger implements Logger interface with standard log package
type SimpleLogger struct {
	level LogLevel
}

// LogLevel represents logging level
type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
)

// NewSimpleLogger creates a new simple logger
func NewSimpleLogger(level LogLevel) *SimpleLogger {
	return &SimpleLogger{level: level}
}

func (l *SimpleLogger) Debug(msg string, fields ...interface{}) {
	if l.level <= LogLevelDebug {
		log.Printf("[DEBUG] %s %v", msg, fields)
	}
}

func (l *SimpleLogger) Info(msg string, fields ...interface{}) {
	if l.level <= LogLevelInfo {
		log.Printf("[INFO] %s %v", msg, fields)
	}
}

func (l *SimpleLogger) Warn(msg string, fields ...interface{}) {
	if l.level <= LogLevelWarn {
		log.Printf("[WARN] %s %v", msg, fields)
	}
}

func (l *SimpleLogger) Error(msg string, fields ...interface{}) {
	if l.level <= LogLevelError {
		log.Printf("[ERROR] %s %v", msg, fields)
	}
}

// SimpleMetricsCollector implements MetricsCollector with in-memory storage
type SimpleMetricsCollector struct {
	latencies map[string][]time.Duration
	tokens    map[string]int
	costs     float64
	errors    map[string]int
}

// NewSimpleMetricsCollector creates a new simple metrics collector
func NewSimpleMetricsCollector() *SimpleMetricsCollector {
	return &SimpleMetricsCollector{
		latencies: make(map[string][]time.Duration),
		tokens:    make(map[string]int),
		errors:    make(map[string]int),
	}
}

func (m *SimpleMetricsCollector) RecordLatency(operation string, duration time.Duration, labels map[string]string) {
	key := fmt.Sprintf("%s_%s", operation, labels["model"])
	m.latencies[key] = append(m.latencies[key], duration)
}

func (m *SimpleMetricsCollector) RecordTokens(promptTokens, completionTokens int, labels map[string]string) {
	m.tokens["prompt"] += promptTokens
	m.tokens["completion"] += completionTokens
	m.tokens["total"] += promptTokens + completionTokens
}

func (m *SimpleMetricsCollector) RecordCost(cost float64, labels map[string]string) {
	m.costs += cost
}

func (m *SimpleMetricsCollector) RecordError(operation string, err error, labels map[string]string) {
	key := fmt.Sprintf("%s_%s", operation, labels["model"])
	m.errors[key]++
}

// GetSummary returns a summary of collected metrics
func (m *SimpleMetricsCollector) GetSummary() map[string]interface{} {
	summary := map[string]interface{}{
		"total_cost":        m.costs,
		"total_tokens":      m.tokens["total"],
		"prompt_tokens":     m.tokens["prompt"],
		"completion_tokens": m.tokens["completion"],
		"errors":            m.errors,
	}

	// Calculate average latencies
	avgLatencies := make(map[string]float64)
	for key, durations := range m.latencies {
		if len(durations) > 0 {
			var total time.Duration
			for _, d := range durations {
				total += d
			}
			avgLatencies[key] = float64(total) / float64(len(durations)) / float64(time.Millisecond)
		}
	}
	summary["avg_latency_ms"] = avgLatencies

	return summary
}
