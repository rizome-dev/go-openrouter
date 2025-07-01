package models

// Plugin represents a plugin configuration
type Plugin struct {
	ID string `json:"id"`

	// Web search plugin configuration
	MaxResults   *int   `json:"max_results,omitempty"`
	SearchPrompt string `json:"search_prompt,omitempty"`

	// PDF parser plugin configuration
	PDF *PDFConfig `json:"pdf,omitempty"`
}

// PDFConfig represents PDF processing configuration
type PDFConfig struct {
	Engine PDFEngine `json:"engine"`
}

// PDFEngine represents the PDF processing engine
type PDFEngine string

const (
	PDFEngineMistralOCR PDFEngine = "mistral-ocr" // Best for scanned documents ($2/1000 pages)
	PDFEngineText       PDFEngine = "pdf-text"    // Best for text PDFs (free)
	PDFEngineNative     PDFEngine = "native"      // Use model's native file processing
)

// NewWebPlugin creates a new web search plugin
func NewWebPlugin() *Plugin {
	return &Plugin{
		ID: "web",
	}
}

// WithMaxResults sets the maximum number of web search results
func (p *Plugin) WithMaxResults(max int) *Plugin {
	p.MaxResults = &max
	return p
}

// WithSearchPrompt sets a custom search prompt
func (p *Plugin) WithSearchPrompt(prompt string) *Plugin {
	p.SearchPrompt = prompt
	return p
}

// NewPDFPlugin creates a new PDF parser plugin
func NewPDFPlugin(engine PDFEngine) *Plugin {
	return &Plugin{
		ID: "file-parser",
		PDF: &PDFConfig{
			Engine: engine,
		},
	}
}
