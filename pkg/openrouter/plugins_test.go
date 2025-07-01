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

func TestWebPlugin(t *testing.T) {
	tests := []struct {
		name         string
		plugin       *models.Plugin
		verifyPlugin func(t *testing.T, p models.Plugin)
	}{
		{
			name:   "Default web plugin",
			plugin: models.NewWebPlugin(),
			verifyPlugin: func(t *testing.T, p models.Plugin) {
				assert.Equal(t, "web", p.ID)
				assert.Nil(t, p.MaxResults) // Default web plugin doesn't set MaxResults
				assert.Empty(t, p.SearchPrompt)
			},
		},
		{
			name:   "Web plugin with max results",
			plugin: models.NewWebPlugin().WithMaxResults(10),
			verifyPlugin: func(t *testing.T, p models.Plugin) {
				assert.Equal(t, "web", p.ID)
				require.NotNil(t, p.MaxResults)
				assert.Equal(t, 10, *p.MaxResults)
			},
		},
		{
			name:   "Web plugin with custom prompt",
			plugin: models.NewWebPlugin().WithSearchPrompt("Search for recent developments:"),
			verifyPlugin: func(t *testing.T, p models.Plugin) {
				assert.Equal(t, "web", p.ID)
				assert.Equal(t, "Search for recent developments:", p.SearchPrompt)
			},
		},
		{
			name: "Web plugin with all options",
			plugin: models.NewWebPlugin().
				WithMaxResults(3).
				WithSearchPrompt("Find authoritative sources:"),
			verifyPlugin: func(t *testing.T, p models.Plugin) {
				assert.Equal(t, "web", p.ID)
				require.NotNil(t, p.MaxResults)
				assert.Equal(t, 3, *p.MaxResults)
				assert.Equal(t, "Find authoritative sources:", p.SearchPrompt)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.verifyPlugin(t, *tt.plugin)

			// Test serialization
			data, err := json.Marshal(tt.plugin)
			require.NoError(t, err)

			var decoded models.Plugin
			err = json.Unmarshal(data, &decoded)
			require.NoError(t, err)

			tt.verifyPlugin(t, decoded)
		})
	}
}

func TestPDFPlugin(t *testing.T) {
	tests := []struct {
		name         string
		plugin       *models.Plugin
		verifyPlugin func(t *testing.T, p models.Plugin)
	}{
		{
			name:   "PDF plugin with Mistral OCR",
			plugin: models.NewPDFPlugin(models.PDFEngineMistralOCR),
			verifyPlugin: func(t *testing.T, p models.Plugin) {
				assert.Equal(t, "file-parser", p.ID)
				assert.NotNil(t, p.PDF)
				assert.Equal(t, models.PDFEngineMistralOCR, p.PDF.Engine)
			},
		},
		{
			name:   "PDF plugin with PDF text",
			plugin: models.NewPDFPlugin(models.PDFEngineText),
			verifyPlugin: func(t *testing.T, p models.Plugin) {
				assert.Equal(t, "file-parser", p.ID)
				assert.NotNil(t, p.PDF)
				assert.Equal(t, models.PDFEngineText, p.PDF.Engine)
			},
		},
		{
			name:   "PDF plugin with native",
			plugin: models.NewPDFPlugin(models.PDFEngineNative),
			verifyPlugin: func(t *testing.T, p models.Plugin) {
				assert.Equal(t, "file-parser", p.ID)
				assert.NotNil(t, p.PDF)
				assert.Equal(t, models.PDFEngineNative, p.PDF.Engine)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.verifyPlugin(t, *tt.plugin)

			// Test serialization
			data, err := json.Marshal(tt.plugin)
			require.NoError(t, err)

			var decoded models.Plugin
			err = json.Unmarshal(data, &decoded)
			require.NoError(t, err)

			tt.verifyPlugin(t, decoded)
		})
	}
}

func TestPluginsInRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req models.ChatCompletionRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)

		// Verify plugins
		assert.Len(t, req.Plugins, 2)
		
		// Check web plugin
		webPlugin := req.Plugins[0]
		assert.Equal(t, "web", webPlugin.ID)
		require.NotNil(t, webPlugin.MaxResults)
		assert.Equal(t, 10, *webPlugin.MaxResults)

		// Check PDF plugin
		pdfPlugin := req.Plugins[1]
		assert.Equal(t, "file-parser", pdfPlugin.ID)
		assert.Equal(t, models.PDFEngineMistralOCR, pdfPlugin.PDF.Engine)

		resp := models.ChatCompletionResponse{
			ID:    "resp-123",
			Model: req.Model,
			Choices: []models.Choice{
				{
					Message: &models.Message{
						Role:    models.RoleAssistant,
						Content: json.RawMessage(`"Response with plugins"`),
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))

	req := models.ChatCompletionRequest{
		Model: "openai/gpt-4",
		Messages: []models.Message{
			models.NewTextMessage(models.RoleUser, "Test with plugins"),
		},
		Plugins: []models.Plugin{
			*models.NewWebPlugin().WithMaxResults(10),
			*models.NewPDFPlugin(models.PDFEngineMistralOCR),
		},
	}

	resp, err := client.CreateChatCompletion(context.Background(), req)
	require.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestOnlineModelShortcut(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req models.ChatCompletionRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)

		// Verify :online suffix
		assert.Equal(t, "openai/gpt-4o:online", req.Model)

		resp := models.ChatCompletionResponse{
			ID:    "resp-123",
			Model: req.Model,
			Choices: []models.Choice{
				{
					Message: &models.Message{
						Role:    models.RoleAssistant,
						Content: json.RawMessage(`"Web search enabled response"`),
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))

	// Using :online suffix should enable web search
	resp, err := client.CreateChatCompletion(context.Background(), models.ChatCompletionRequest{
		Model: "openai/gpt-4o:online",
		Messages: []models.Message{
			models.NewTextMessage(models.RoleUser, "What's the latest news?"),
		},
	})

	require.NoError(t, err)
	assert.Contains(t, resp.Model, ":online")
}

func TestWebSearchOptions(t *testing.T) {
	tests := []struct {
		name        string
		options     *models.WebSearchOptions
		expectError bool
	}{
		{
			name: "Low search context",
			options: &models.WebSearchOptions{
				SearchContextSize: "low",
			},
			expectError: false,
		},
		{
			name: "Medium search context",
			options: &models.WebSearchOptions{
				SearchContextSize: "medium",
			},
			expectError: false,
		},
		{
			name: "High search context",
			options: &models.WebSearchOptions{
				SearchContextSize: "high",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var req models.ChatCompletionRequest
				err := json.NewDecoder(r.Body).Decode(&req)
				require.NoError(t, err)

				// Verify web search options
				if tt.options != nil {
					assert.NotNil(t, req.WebSearchOptions)
					assert.Equal(t, tt.options.SearchContextSize, req.WebSearchOptions.SearchContextSize)
				}

				resp := models.ChatCompletionResponse{
					ID:    "resp-123",
					Model: req.Model,
					Choices: []models.Choice{
						{
							Message: &models.Message{
								Role:    models.RoleAssistant,
								Content: json.RawMessage(`"Search response"`),
							},
						},
					},
				}
				json.NewEncoder(w).Encode(resp)
			}))
			defer server.Close()

			client := NewClient("test-key", WithBaseURL(server.URL))

			req := models.ChatCompletionRequest{
				Model: "perplexity/sonar",
				Messages: []models.Message{
					models.NewTextMessage(models.RoleUser, "Search query"),
				},
				WebSearchOptions: tt.options,
			}

			resp, err := client.CreateChatCompletion(context.Background(), req)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
			}
		})
	}
}

func TestPluginCombinations(t *testing.T) {
	// Test various plugin combinations
	tests := []struct {
		name    string
		plugins []models.Plugin
	}{
		{
			name:    "No plugins",
			plugins: []models.Plugin{},
		},
		{
			name: "Web only",
			plugins: []models.Plugin{
				*models.NewWebPlugin(),
			},
		},
		{
			name: "PDF only",
			plugins: []models.Plugin{
				*models.NewPDFPlugin(models.PDFEngineText),
			},
		},
		{
			name: "Web and PDF",
			plugins: []models.Plugin{
				*models.NewWebPlugin().WithMaxResults(3),
				*models.NewPDFPlugin(models.PDFEngineMistralOCR),
			},
		},
		{
			name: "Multiple PDF engines (last one wins)",
			plugins: []models.Plugin{
				*models.NewPDFPlugin(models.PDFEngineText),
				*models.NewPDFPlugin(models.PDFEngineMistralOCR),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := models.ChatCompletionRequest{
				Model: "openai/gpt-4",
				Messages: []models.Message{
					models.NewTextMessage(models.RoleUser, "Test"),
				},
				Plugins: tt.plugins,
			}

			// Test serialization
			data, err := json.Marshal(req)
			require.NoError(t, err)

			var decoded models.ChatCompletionRequest
			err = json.Unmarshal(data, &decoded)
			require.NoError(t, err)

			assert.Equal(t, len(tt.plugins), len(decoded.Plugins))
		})
	}
}

func TestPluginEdgeCases(t *testing.T) {
	t.Run("Empty plugin ID", func(t *testing.T) {
		plugin := models.Plugin{
			ID: "",
		}
		
		data, err := json.Marshal(plugin)
		require.NoError(t, err)
		
		var decoded models.Plugin
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)
		assert.Empty(t, decoded.ID)
	})

	t.Run("Negative max results", func(t *testing.T) {
		// MaxResults should be validated in real implementation
		negOne := -1
		plugin := models.Plugin{
			ID:         "web",
			MaxResults: &negOne,
		}

		data, err := json.Marshal(plugin)
		require.NoError(t, err)
		assert.Contains(t, string(data), "-1")
	})

	t.Run("Very long search prompt", func(t *testing.T) {
		longPrompt := ""
		for i := 0; i < 10000; i++ {
			longPrompt += "a"
		}

		plugin := models.NewWebPlugin().WithSearchPrompt(longPrompt)
		assert.Equal(t, longPrompt, plugin.SearchPrompt)
	})

	t.Run("Invalid PDF engine", func(t *testing.T) {
		plugin := models.Plugin{
			ID: "file-parser",
			PDF: &models.PDFConfig{
				Engine: "invalid-engine",
			},
		}

		data, err := json.Marshal(plugin)
		require.NoError(t, err)
		assert.Contains(t, string(data), "invalid-engine")
	})
}

func TestPluginMethod(t *testing.T) {
	// Test method chaining
	plugin := models.NewWebPlugin().
		WithMaxResults(10).
		WithSearchPrompt("Find recent info:").
		WithMaxResults(5) // Should override previous value

	require.NotNil(t, plugin.MaxResults)
	assert.Equal(t, 5, *plugin.MaxResults)
	assert.Equal(t, "Find recent info:", plugin.SearchPrompt)
}