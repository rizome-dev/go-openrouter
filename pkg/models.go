package pkg

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/rizome-dev/go-openrouter/pkg/models"
)

// ListModelsOptions represents options for listing models
type ListModelsOptions struct {
	Category string
}

// ListModels lists available models
func (c *Client) ListModels(ctx context.Context, opts *ListModelsOptions) (*models.ModelsResponse, error) {
	endpoint := "/models"

	if opts != nil && opts.Category != "" {
		params := url.Values{}
		params.Set("category", opts.Category)
		endpoint = endpoint + "?" + params.Encode()
	}

	resp, err := c.doRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var modelsResp models.ModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &modelsResp, nil
}
