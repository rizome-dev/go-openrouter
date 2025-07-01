package openrouter

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rizome-dev/go-openrouter/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWebSearchHelper_CreateWithWebSearch(t *testing.T) {
	t.Run("With :online suffix", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req models.ChatCompletionRequest
			err := json.NewDecoder(r.Body).Decode(&req)
			require.NoError(t, err)

			// Verify :online suffix
			assert.Equal(t, "openai/gpt-4o:online", req.Model)

			resp := models.ChatCompletionResponse{
				ID:    "resp-123",
				Model: "openai/gpt-4o:online",
				Choices: []models.Choice{
					{
						Message: &models.Message{
							Role:    models.RoleAssistant,
							Content: json.RawMessage(`"Based on my web search, the latest news is..."`),
							Annotations: []models.Annotation{
								{
									Type: models.AnnotationTypeURLCitation,
									URLCitation: &models.URLCitation{
										URL:        "https://example.com/news",
										Title:      "Latest News",
										Content:    "Breaking news content",
										StartIndex: 0,
										EndIndex:   20,
									},
								},
							},
						},
					},
				},
			}
			json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		client := NewClient("test-key", WithBaseURL(server.URL))
		helper := NewWebSearchHelper(client)

		resp, err := helper.CreateWithWebSearch(context.Background(), "What's the latest news?", "openai/gpt-4o", nil)
		require.NoError(t, err)
		assert.NotNil(t, resp)

		// Check annotations
		assert.Len(t, resp.Choices[0].Message.Annotations, 1)
		assert.Equal(t, models.AnnotationTypeURLCitation, resp.Choices[0].Message.Annotations[0].Type)
	})

	t.Run("With web plugin options", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req models.ChatCompletionRequest
			err := json.NewDecoder(r.Body).Decode(&req)
			require.NoError(t, err)

			// Verify plugin configuration
			assert.Len(t, req.Plugins, 1)
			plugin := req.Plugins[0]
			assert.Equal(t, "web", plugin.ID)
			assert.Equal(t, 10, *plugin.MaxResults)
			assert.Equal(t, "Custom search prompt:", plugin.SearchPrompt)

			resp := models.ChatCompletionResponse{
				ID:    "resp-123",
				Model: "openai/gpt-4o",
				Choices: []models.Choice{
					{
						Message: &models.Message{
							Role:    models.RoleAssistant,
							Content: json.RawMessage(`"Search results processed"`),
						},
					},
				},
			}
			json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		client := NewClient("test-key", WithBaseURL(server.URL))
		helper := NewWebSearchHelper(client)

		opts := &SearchOptions{
			MaxResults:   10,
			SearchPrompt: "Custom search prompt:",
		}

		resp, err := helper.CreateWithWebSearch(context.Background(), "Search query", "openai/gpt-4o", opts)
		require.NoError(t, err)
		assert.NotNil(t, resp)
	})
}

func TestWebSearchHelper_CreateWithNativeWebSearch(t *testing.T) {
	tests := []struct {
		name        string
		contextSize string
		expectError bool
	}{
		{
			name:        "Low context",
			contextSize: "low",
			expectError: false,
		},
		{
			name:        "Medium context",
			contextSize: "medium",
			expectError: false,
		},
		{
			name:        "High context",
			contextSize: "high",
			expectError: false,
		},
		{
			name:        "Empty context",
			contextSize: "",
			expectError: false,
		},
		{
			name:        "Invalid context",
			contextSize: "invalid",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var req models.ChatCompletionRequest
				err := json.NewDecoder(r.Body).Decode(&req)
				require.NoError(t, err)

				// Verify web search options
				if tt.contextSize != "" {
					assert.NotNil(t, req.WebSearchOptions)
					assert.Equal(t, tt.contextSize, req.WebSearchOptions.SearchContextSize)
				}

				resp := models.ChatCompletionResponse{
					ID:    "resp-123",
					Model: "perplexity/sonar",
					Choices: []models.Choice{
						{
							Message: &models.Message{
								Role:    models.RoleAssistant,
								Content: json.RawMessage(`"Native search result"`),
							},
						},
					},
				}
				json.NewEncoder(w).Encode(resp)
			}))
			defer server.Close()

			client := NewClient("test-key", WithBaseURL(server.URL))
			helper := NewWebSearchHelper(client)

			resp, err := helper.CreateWithNativeWebSearch(context.Background(), "Query", "perplexity/sonar", tt.contextSize)
			
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
			}
		})
	}
}

func TestExtractCitations(t *testing.T) {
	tests := []struct {
		name      string
		response  *models.ChatCompletionResponse
		expected  int
	}{
		{
			name: "Response with citations",
			response: &models.ChatCompletionResponse{
				Choices: []models.Choice{
					{
						Message: &models.Message{
							Annotations: []models.Annotation{
								{
									Type: models.AnnotationTypeURLCitation,
									URLCitation: &models.URLCitation{
										URL:   "https://example.com",
										Title: "Example",
									},
								},
								{
									Type: models.AnnotationTypeURLCitation,
									URLCitation: &models.URLCitation{
										URL:   "https://test.com",
										Title: "Test",
									},
								},
							},
						},
					},
				},
			},
			expected: 2,
		},
		{
			name: "Response without citations",
			response: &models.ChatCompletionResponse{
				Choices: []models.Choice{
					{
						Message: &models.Message{
							Annotations: []models.Annotation{},
						},
					},
				},
			},
			expected: 0,
		},
		{
			name: "Empty response",
			response: &models.ChatCompletionResponse{
				Choices: []models.Choice{},
			},
			expected: 0,
		},
		{
			name: "Mixed annotation types",
			response: &models.ChatCompletionResponse{
				Choices: []models.Choice{
					{
						Message: &models.Message{
							Annotations: []models.Annotation{
								{
									Type: models.AnnotationTypeURLCitation,
									URLCitation: &models.URLCitation{
										URL: "https://example.com",
									},
								},
								{
									Type: models.AnnotationTypeFile,
									File: &models.FileAnnotation{
										Filename: "file-123",
										FileData: map[string]interface{}{
											"id": "file-123",
										},
									},
								},
							},
						},
					},
				},
			},
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			citations := ExtractCitations(tt.response)
			assert.Len(t, citations, tt.expected)
		})
	}
}

func TestFormatCitationsAsMarkdown(t *testing.T) {
	tests := []struct {
		name      string
		citations []models.URLCitation
		expected  string
	}{
		{
			name: "Single citation",
			citations: []models.URLCitation{
				{URL: "https://example.com/page"},
			},
			expected: "[example.com](https://example.com/page)",
		},
		{
			name: "Multiple citations",
			citations: []models.URLCitation{
				{URL: "https://example.com/page"},
				{URL: "https://test.org/article"},
			},
			expected: "[example.com](https://example.com/page), [test.org](https://test.org/article)",
		},
		{
			name: "Citation with www",
			citations: []models.URLCitation{
				{URL: "https://www.example.com/page"},
			},
			expected: "[example.com](https://www.example.com/page)",
		},
		{
			name: "Citation without protocol",
			citations: []models.URLCitation{
				{URL: "example.com/page"},
			},
			expected: "[example.com](example.com/page)",
		},
		{
			name:      "Empty citations",
			citations: []models.URLCitation{},
			expected:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatCitationsAsMarkdown(tt.citations)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractDomain(t *testing.T) {
	tests := []struct {
		url      string
		expected string
	}{
		{"https://example.com", "example.com"},
		{"https://example.com/page", "example.com"},
		{"http://www.example.com/page", "example.com"},
		{"example.com/page", "example.com"},
		{"https://sub.example.com/page", "sub.example.com"},
		{"https://example.com:8080/page", "example.com:8080"},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			result := extractDomain(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestResearchAgent(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		
		var req models.ChatCompletionRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)

		var content string
		var annotations []models.Annotation

		if callCount == 1 {
			// Initial research call
			content = `Here's an overview of AI:
1. Machine Learning - algorithms that learn from data
2. Neural Networks - systems inspired by the brain
3. Natural Language Processing - understanding human language`
			
			annotations = []models.Annotation{
				{
					Type: models.AnnotationTypeURLCitation,
					URLCitation: &models.URLCitation{
						URL:   "https://ai.example.com",
						Title: "AI Overview",
					},
				},
			}
		} else if callCount <= 4 {
			// Subtopic research calls
			content = "Detailed information about the subtopic..."
			annotations = []models.Annotation{
				{
					Type: models.AnnotationTypeURLCitation,
					URLCitation: &models.URLCitation{
						URL:   "https://subtopic.example.com",
						Title: "Subtopic Details",
					},
				},
			}
		} else {
			// Summary call
			content = "In summary, AI encompasses various technologies..."
		}

		resp := models.ChatCompletionResponse{
			ID:    "resp-" + string(rune(callCount)),
			Model: req.Model,
			Choices: []models.Choice{
				{
					Message: &models.Message{
						Role: models.RoleAssistant,
						Content: func() json.RawMessage {
							contentJSON, _ := json.Marshal(content)
							return json.RawMessage(contentJSON)
						}(),
						Annotations: annotations,
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	helper := NewWebSearchHelper(client)
	agent := helper.CreateResearchAgent("openai/gpt-4o")

	result, err := agent.Research(context.Background(), "Artificial Intelligence", 3)
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Verify research result
	assert.Equal(t, "Artificial Intelligence", result.Topic)
	assert.NotEmpty(t, result.Summary)
	assert.GreaterOrEqual(t, len(result.Sections), 1)
	assert.NotEmpty(t, result.Citations)

	// Check that overview section exists
	assert.Equal(t, "Overview", result.Sections[0].Title)
	assert.NotEmpty(t, result.Sections[0].Content)
}

func TestExtractSubtopics(t *testing.T) {
	agent := &ResearchAgent{}

	tests := []struct {
		name     string
		content  string
		expected []string
	}{
		{
			name: "Numbered list",
			content: `Overview:
1. First topic
2. Second topic
3. Third topic`,
			expected: []string{"First topic", "Second topic", "Third topic"},
		},
		{
			name: "Bullet points",
			content: `Topics:
• Topic A
• Topic B
- Topic C
* Topic D`,
			expected: []string{"Topic A", "Topic B", "Topic C", "Topic D"},
		},
		{
			name:     "No topics",
			content:  "This is just plain text without any list",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &models.ChatCompletionResponse{
				Choices: []models.Choice{
					{
						Message: &models.Message{
							Content: func() json.RawMessage {
								contentJSON, _ := json.Marshal(tt.content)
								return json.RawMessage(contentJSON)
							}(),
						},
					},
				},
			}
			
			subtopics := agent.extractSubtopics(resp, 10)
			assert.Equal(t, tt.expected, subtopics)
		})
	}
}

func TestWebSearchWithStreaming(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req models.ChatCompletionRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)

		// Verify streaming is disabled for web search
		assert.False(t, req.Stream)
		assert.Contains(t, req.Model, ":online")

		resp := models.ChatCompletionResponse{
			ID:    "resp-123",
			Model: req.Model,
			Choices: []models.Choice{
				{
					Message: &models.Message{
						Role:    models.RoleAssistant,
						Content: json.RawMessage(`"Web search result"`),
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	helper := NewWebSearchHelper(client)

	// Web search should work even if streaming was requested
	resp, err := helper.CreateWithWebSearch(context.Background(), "Query", "openai/gpt-4o", nil)
	require.NoError(t, err)
	assert.NotNil(t, resp)
}