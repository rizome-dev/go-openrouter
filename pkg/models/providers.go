package models

// Provider represents an AI provider
type Provider struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Status      string   `json:"status"` // operational, degraded, down
	Models      []string `json:"models"`
}

// ProvidersResponse represents the response from listing providers
type ProvidersResponse struct {
	Data []Provider `json:"data"`
}

// ModelEndpoint represents an endpoint for a specific model
type ModelEndpoint struct {
	ID         string `json:"id"`
	Provider   string `json:"provider"`
	Model      string `json:"model"`
	URL        string `json:"url"`
	Region     string `json:"region"`
	Latency    int    `json:"latency"`    // milliseconds
	Throughput int    `json:"throughput"` // requests per second
	Status     string `json:"status"`     // healthy, degraded, unhealthy
}

// ModelEndpointsResponse represents the response from listing model endpoints
type ModelEndpointsResponse struct {
	Data []ModelEndpoint `json:"data"`
}