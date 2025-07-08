package openrouter

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/rizome-dev/go-openrouter/pkg/models"
)

// ListAPIKeysOptions contains options for listing API keys
type ListAPIKeysOptions struct {
	Offset          int
	IncludeDisabled bool
}

// ListAPIKeys returns a list of all API keys associated with the account
// Requires a Provisioning API key
func (c *Client) ListAPIKeys(ctx context.Context, opts *ListAPIKeysOptions) (*models.APIKeysResponse, error) {
	endpoint := "/api/v1/keys"
	if opts != nil {
		params := url.Values{}
		if opts.Offset > 0 {
			params.Set("offset", fmt.Sprintf("%d", opts.Offset))
		}
		if opts.IncludeDisabled {
			params.Set("include_disabled", "true")
		}
		if len(params) > 0 {
			endpoint += "?" + params.Encode()
		}
	}
	
	resp, err := c.doRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result models.APIKeysResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

// CreateAPIKey creates a new API key
// Requires a Provisioning API key
func (c *Client) CreateAPIKey(ctx context.Context, req models.CreateAPIKeyRequest) (*models.APIKey, error) {
	resp, err := c.doRequest(ctx, "POST", "/api/v1/keys", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result models.APIKey
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetAPIKey returns details about a specific API key
// Requires a Provisioning API key
func (c *Client) GetAPIKey(ctx context.Context, keyHash string) (*models.APIKey, error) {
	resp, err := c.doRequest(ctx, "GET", fmt.Sprintf("/api/v1/keys/%s", keyHash), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result models.APIKey
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetCurrentAPIKey gets information on the API key associated with the current authentication session
func (c *Client) GetCurrentAPIKey(ctx context.Context) (*models.APIKey, error) {
	resp, err := c.doRequest(ctx, "GET", "/api/v1/me/keys", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result models.APIKey
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

// UpdateAPIKey updates an existing API key
// Requires a Provisioning API key
func (c *Client) UpdateAPIKey(ctx context.Context, keyHash string, req models.UpdateAPIKeyRequest) (*models.APIKey, error) {
	resp, err := c.doRequest(ctx, "PATCH", fmt.Sprintf("/api/v1/keys/%s", keyHash), req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result models.APIKey
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

// DeleteAPIKey deletes an API key
// Requires a Provisioning API key
func (c *Client) DeleteAPIKey(ctx context.Context, keyHash string) error {
	resp, err := c.doRequest(ctx, "DELETE", fmt.Sprintf("/api/v1/keys/%s", keyHash), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

// GetCredits returns the total credits purchased and used for the authenticated user
func (c *Client) GetCredits(ctx context.Context) (*models.CreditsResponse, error) {
	resp, err := c.doRequest(ctx, "GET", "/api/v1/me/credits", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result models.CreditsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ListProviders returns a list of providers available through the API
func (c *Client) ListProviders(ctx context.Context) (*models.ProvidersResponse, error) {
	resp, err := c.doRequest(ctx, "GET", "/api/v1/providers", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result models.ProvidersResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ListModelEndpoints returns the available endpoints/providers for a specific model
// The model parameter should be in the format "author/slug" (e.g., "openai/gpt-4")
func (c *Client) ListModelEndpoints(ctx context.Context, model string) (*models.ModelEndpointsResponse, error) {
	// Split the model into author and slug
	parts := strings.Split(model, "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid model format: expected 'author/slug', got '%s'", model)
	}
	
	resp, err := c.doRequest(ctx, "GET", fmt.Sprintf("/api/v1/endpoints/%s/%s", parts[0], parts[1]), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result models.ModelEndpointsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ExchangeAuthCodeForAPIKey exchanges an authorization code from the PKCE OAuth flow for a user-controlled API key
func (c *Client) ExchangeAuthCodeForAPIKey(ctx context.Context, req models.ExchangeAuthCodeRequest) (*models.ExchangeAuthCodeResponse, error) {
	resp, err := c.doRequest(ctx, "POST", "/api/v1/auth/keys", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result models.ExchangeAuthCodeResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

// CreateCoinbaseCharge creates and hydrates a Coinbase Commerce charge for cryptocurrency payments
func (c *Client) CreateCoinbaseCharge(ctx context.Context, req models.CreateCoinbaseChargeRequest) (*models.CoinbaseChargeResponse, error) {
	resp, err := c.doRequest(ctx, "POST", "/api/v1/me/coinbase-charge", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result models.CoinbaseChargeResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}