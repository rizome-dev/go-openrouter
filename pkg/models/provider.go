package models

// ProviderPreferences represents provider routing preferences
type ProviderPreferences struct {
	// List of provider slugs to try in order
	Order []string `json:"order,omitempty"`
	
	// Whether to allow backup providers
	AllowFallbacks *bool `json:"allow_fallbacks,omitempty"`
	
	// Only use providers that support all parameters
	RequireParameters *bool `json:"require_parameters,omitempty"`
	
	// Data collection policy
	DataCollection DataCollectionPolicy `json:"data_collection,omitempty"`
	
	// List of provider slugs to allow
	Only []string `json:"only,omitempty"`
	
	// List of provider slugs to ignore
	Ignore []string `json:"ignore,omitempty"`
	
	// List of quantization levels to filter by
	Quantizations []QuantizationLevel `json:"quantizations,omitempty"`
	
	// Sort providers by this attribute
	Sort SortStrategy `json:"sort,omitempty"`
	
	// Maximum price limits
	MaxPrice *MaxPrice `json:"max_price,omitempty"`
}

// DataCollectionPolicy represents data collection preferences
type DataCollectionPolicy string

const (
	DataCollectionAllow DataCollectionPolicy = "allow"
	DataCollectionDeny  DataCollectionPolicy = "deny"
)

// QuantizationLevel represents model quantization levels
type QuantizationLevel string

const (
	QuantizationInt4    QuantizationLevel = "int4"
	QuantizationInt8    QuantizationLevel = "int8"
	QuantizationFP4     QuantizationLevel = "fp4"
	QuantizationFP6     QuantizationLevel = "fp6"
	QuantizationFP8     QuantizationLevel = "fp8"
	QuantizationFP16    QuantizationLevel = "fp16"
	QuantizationBF16    QuantizationLevel = "bf16"
	QuantizationFP32    QuantizationLevel = "fp32"
	QuantizationUnknown QuantizationLevel = "unknown"
)

// SortStrategy represents how to sort providers
type SortStrategy string

const (
	SortByPrice      SortStrategy = "price"
	SortByThroughput SortStrategy = "throughput"
	SortByLatency    SortStrategy = "latency"
)

// MaxPrice represents maximum price limits
type MaxPrice struct {
	// Maximum price per million prompt tokens
	Prompt float64 `json:"prompt,omitempty"`
	
	// Maximum price per million completion tokens
	Completion float64 `json:"completion,omitempty"`
	
	// Maximum price per image
	Image float64 `json:"image,omitempty"`
	
	// Maximum price per request
	Request float64 `json:"request,omitempty"`
}

// NewProviderPreferences creates a new ProviderPreferences with defaults
func NewProviderPreferences() *ProviderPreferences {
	return &ProviderPreferences{}
}

// WithOrder sets the provider order
func (p *ProviderPreferences) WithOrder(providers ...string) *ProviderPreferences {
	p.Order = providers
	return p
}

// WithFallbacks sets whether to allow fallbacks
func (p *ProviderPreferences) WithFallbacks(allow bool) *ProviderPreferences {
	p.AllowFallbacks = &allow
	return p
}

// WithRequireParameters sets whether to require parameter support
func (p *ProviderPreferences) WithRequireParameters(require bool) *ProviderPreferences {
	p.RequireParameters = &require
	return p
}

// WithDataCollection sets the data collection policy
func (p *ProviderPreferences) WithDataCollection(policy DataCollectionPolicy) *ProviderPreferences {
	p.DataCollection = policy
	return p
}

// WithOnly sets the list of allowed providers
func (p *ProviderPreferences) WithOnly(providers ...string) *ProviderPreferences {
	p.Only = providers
	return p
}

// WithIgnore sets the list of ignored providers
func (p *ProviderPreferences) WithIgnore(providers ...string) *ProviderPreferences {
	p.Ignore = providers
	return p
}

// WithQuantizations sets the allowed quantization levels
func (p *ProviderPreferences) WithQuantizations(levels ...QuantizationLevel) *ProviderPreferences {
	p.Quantizations = levels
	return p
}

// WithSort sets the sort strategy
func (p *ProviderPreferences) WithSort(strategy SortStrategy) *ProviderPreferences {
	p.Sort = strategy
	return p
}

// WithMaxPrice sets the maximum price limits
func (p *ProviderPreferences) WithMaxPrice(prompt, completion float64) *ProviderPreferences {
	p.MaxPrice = &MaxPrice{
		Prompt:     prompt,
		Completion: completion,
	}
	return p
}