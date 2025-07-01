package openrouter

import (
	"context"
	"fmt"
	"strings"

	"github.com/rizome-dev/go-openrouter/pkg/models"
)

// WebSearchHelper provides utilities for web search functionality
type WebSearchHelper struct {
	client *Client
}

// NewWebSearchHelper creates a new web search helper
func NewWebSearchHelper(client *Client) *WebSearchHelper {
	return &WebSearchHelper{client: client}
}

// SearchOptions represents options for web search
type SearchOptions struct {
	MaxResults   int
	SearchPrompt string
}

// CreateWithWebSearch creates a chat completion with web search enabled
func (w *WebSearchHelper) CreateWithWebSearch(ctx context.Context, prompt string, model string, opts *SearchOptions) (*models.ChatCompletionResponse, error) {
	// Use :online shortcut if no specific options
	if opts == nil {
		return w.client.CreateChatCompletion(ctx, models.ChatCompletionRequest{
			Model: model + ":online",
			Messages: []models.Message{
				models.NewTextMessage(models.RoleUser, prompt),
			},
		})
	}

	// Create web plugin with options
	plugin := models.NewWebPlugin()
	if opts.MaxResults > 0 {
		plugin = plugin.WithMaxResults(opts.MaxResults)
	}
	if opts.SearchPrompt != "" {
		plugin = plugin.WithSearchPrompt(opts.SearchPrompt)
	}

	// Create request with plugin
	return w.client.CreateChatCompletion(ctx, models.ChatCompletionRequest{
		Model: model,
		Messages: []models.Message{
			models.NewTextMessage(models.RoleUser, prompt),
		},
		Plugins: []models.Plugin{*plugin},
	})
}

// CreateWithNativeWebSearch creates a chat completion using native web search models
func (w *WebSearchHelper) CreateWithNativeWebSearch(ctx context.Context, prompt string, model string, contextSize string) (*models.ChatCompletionResponse, error) {
	// Validate context size
	validSizes := map[string]bool{"low": true, "medium": true, "high": true}
	if contextSize != "" && !validSizes[contextSize] {
		return nil, fmt.Errorf("invalid search context size: %s (must be 'low', 'medium', or 'high')", contextSize)
	}

	req := models.ChatCompletionRequest{
		Model: model,
		Messages: []models.Message{
			models.NewTextMessage(models.RoleUser, prompt),
		},
	}

	// Add web search options for native models
	if contextSize != "" {
		req.WebSearchOptions = &models.WebSearchOptions{
			SearchContextSize: contextSize,
		}
	}

	return w.client.CreateChatCompletion(ctx, req)
}

// ExtractCitations extracts URL citations from a response
func ExtractCitations(resp *models.ChatCompletionResponse) []models.URLCitation {
	var citations []models.URLCitation

	if len(resp.Choices) == 0 || resp.Choices[0].Message == nil {
		return citations
	}

	message := resp.Choices[0].Message
	for _, annotation := range message.Annotations {
		if annotation.Type == models.AnnotationTypeURLCitation && annotation.URLCitation != nil {
			citations = append(citations, *annotation.URLCitation)
		}
	}

	return citations
}

// FormatCitationsAsMarkdown formats citations as markdown links
func FormatCitationsAsMarkdown(citations []models.URLCitation) string {
	var links []string
	for _, citation := range citations {
		// Extract domain from URL for link text
		domain := extractDomain(citation.URL)
		link := fmt.Sprintf("[%s](%s)", domain, citation.URL)
		links = append(links, link)
	}
	return strings.Join(links, ", ")
}

// CreateResearchAgent creates an agent specialized for research tasks
func (w *WebSearchHelper) CreateResearchAgent(model string) *ResearchAgent {
	return &ResearchAgent{
		client: w.client,
		model:  model,
	}
}

// ResearchAgent is an agent specialized for research tasks
type ResearchAgent struct {
	client *Client
	model  string
}

// Research performs a multi-step research process
func (r *ResearchAgent) Research(ctx context.Context, topic string, depth int) (*ResearchResult, error) {
	result := &ResearchResult{
		Topic:     topic,
		Sections:  make([]ResearchSection, 0),
		Citations: make([]models.URLCitation, 0),
	}

	// Initial search
	searchResp, err := r.client.CreateChatCompletion(ctx, models.ChatCompletionRequest{
		Model: r.model + ":online",
		Messages: []models.Message{
			models.NewTextMessage(models.RoleSystem, "You are a research assistant. Provide comprehensive, well-sourced information."),
			models.NewTextMessage(models.RoleUser, fmt.Sprintf("Research the topic: %s. Provide an overview and identify %d key subtopics to explore further.", topic, depth)),
		},
	})

	if err != nil {
		return nil, fmt.Errorf("initial research failed: %w", err)
	}

	// Extract initial content and citations
	if len(searchResp.Choices) > 0 && searchResp.Choices[0].Message != nil {
		content, _ := searchResp.Choices[0].Message.GetTextContent()
		citations := ExtractCitations(searchResp)

		result.Sections = append(result.Sections, ResearchSection{
			Title:     "Overview",
			Content:   content,
			Citations: citations,
		})

		result.Citations = append(result.Citations, citations...)
	}

	// Extract subtopics from the response
	subtopics := r.extractSubtopics(searchResp, depth)

	// Research each subtopic
	for i, subtopic := range subtopics {
		if i >= depth {
			break
		}

		subResp, err := r.client.CreateChatCompletion(ctx, models.ChatCompletionRequest{
			Model: r.model + ":online",
			Messages: []models.Message{
				models.NewTextMessage(models.RoleUser, fmt.Sprintf("Provide detailed information about: %s (in the context of %s)", subtopic, topic)),
			},
		})

		if err != nil {
			continue // Skip failed subtopics
		}

		if len(subResp.Choices) > 0 && subResp.Choices[0].Message != nil {
			content, _ := subResp.Choices[0].Message.GetTextContent()
			citations := ExtractCitations(subResp)

			result.Sections = append(result.Sections, ResearchSection{
				Title:     subtopic,
				Content:   content,
				Citations: citations,
			})

			result.Citations = append(result.Citations, citations...)
		}
	}

	// Generate summary
	summaryResp, err := r.client.CreateChatCompletion(ctx, models.ChatCompletionRequest{
		Model: r.model,
		Messages: []models.Message{
			models.NewTextMessage(models.RoleUser, fmt.Sprintf("Based on the research about %s, provide a concise summary of the key findings.", topic)),
		},
	})

	if err == nil && len(summaryResp.Choices) > 0 && summaryResp.Choices[0].Message != nil {
		result.Summary, _ = summaryResp.Choices[0].Message.GetTextContent()
	}

	return result, nil
}

// extractSubtopics attempts to extract subtopics from the response
func (r *ResearchAgent) extractSubtopics(resp *models.ChatCompletionResponse, maxCount int) []string {
	if len(resp.Choices) == 0 || resp.Choices[0].Message == nil {
		return []string{}
	}

	content, _ := resp.Choices[0].Message.GetTextContent()

	// Simple extraction based on common patterns
	var subtopics []string
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Look for numbered lists or bullet points
		var topic string
		
		// Check for numbered lists (1., 2., 3., etc.)
		if len(line) > 2 && line[1] == '.' && line[0] >= '0' && line[0] <= '9' {
			topic = strings.TrimSpace(line[2:])
		} else if strings.HasPrefix(line, "•") {
			topic = strings.TrimSpace(strings.TrimPrefix(line, "•"))
		} else if strings.HasPrefix(line, "-") {
			topic = strings.TrimSpace(strings.TrimPrefix(line, "-"))
		} else if strings.HasPrefix(line, "*") {
			topic = strings.TrimSpace(strings.TrimPrefix(line, "*"))
		}

		if topic != "" && len(subtopics) < maxCount {
			subtopics = append(subtopics, topic)
		}
	}

	if len(subtopics) == 0 {
		return []string{}
	}
	return subtopics
}

// ResearchResult represents the result of a research process
type ResearchResult struct {
	Topic     string
	Summary   string
	Sections  []ResearchSection
	Citations []models.URLCitation
}

// ResearchSection represents a section of research
type ResearchSection struct {
	Title     string
	Content   string
	Citations []models.URLCitation
}

// extractDomain extracts the domain from a URL
func extractDomain(url string) string {
	// Remove protocol
	domain := strings.TrimPrefix(url, "https://")
	domain = strings.TrimPrefix(domain, "http://")

	// Get domain part
	if idx := strings.Index(domain, "/"); idx > 0 {
		domain = domain[:idx]
	}

	// Remove www
	domain = strings.TrimPrefix(domain, "www.")

	return domain
}
