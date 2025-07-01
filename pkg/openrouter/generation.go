package openrouter

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/rizome-dev/go-openrouter/pkg/models"
)

// GetGeneration retrieves metadata about a specific generation
func (c *Client) GetGeneration(ctx context.Context, generationID string) (*models.GenerationResponse, error) {
	if generationID == "" {
		return nil, fmt.Errorf("generation ID is required")
	}
	
	params := url.Values{}
	params.Set("id", generationID)
	endpoint := "/generation?" + params.Encode()
	
	resp, err := c.doRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	var generationResp models.GenerationResponse
	if err := json.NewDecoder(resp.Body).Decode(&generationResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	
	return &generationResp, nil
}