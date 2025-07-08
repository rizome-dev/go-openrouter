package openrouter

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rizome-dev/go-openrouter/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// API Keys Management Tests

func TestListAPIKeys(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/api/v1/keys", r.URL.Path)

		// Check query parameters
		offset := r.URL.Query().Get("offset")
		includeDisabled := r.URL.Query().Get("include_disabled")

		if offset == "10" {
			assert.Equal(t, "10", offset)
		}
		if includeDisabled == "true" {
			assert.Equal(t, "true", includeDisabled)
		}

		resp := models.APIKeysResponse{
			Data: []models.APIKey{
				{
					ID:          "key-123",
					Name:        "Production Key",
					Key:         "sk-or-v1-xxx",
					CreatedAt:   time.Now().Unix(),
					LastUsed:    time.Now().Unix(),
					UsageCount:  1000,
					Disabled:    false,
					Permissions: []string{"chat.completions", "models.list"},
				},
				{
					ID:          "key-456",
					Name:        "Development Key",
					Key:         "sk-or-v1-yyy",
					CreatedAt:   time.Now().Unix(),
					LastUsed:    time.Now().Unix(),
					UsageCount:  50,
					Disabled:    true,
					Permissions: []string{"chat.completions"},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("provisioning-key", WithBaseURL(server.URL))

	// Test without options
	resp, err := client.ListAPIKeys(context.Background(), nil)
	require.NoError(t, err)
	assert.Len(t, resp.Data, 2)
	assert.Equal(t, "key-123", resp.Data[0].ID)

	// Test with options
	resp, err = client.ListAPIKeys(context.Background(), &ListAPIKeysOptions{
		Offset:          10,
		IncludeDisabled: true,
	})
	require.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestCreateAPIKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/api/v1/keys", r.URL.Path)

		var req models.CreateAPIKeyRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)

		assert.Equal(t, "New API Key", req.Name)
		assert.Contains(t, req.Permissions, "chat.completions")
		assert.Equal(t, 100, req.RateLimit)
		assert.Equal(t, 30*24*60*60, req.ExpiresIn) // 30 days

		resp := models.APIKey{
			ID:          "key-789",
			Name:        req.Name,
			Key:         "sk-or-v1-newkey",
			CreatedAt:   time.Now().Unix(),
			Permissions: req.Permissions,
			RateLimit:   req.RateLimit,
			ExpiresAt:   time.Now().Add(time.Duration(req.ExpiresIn) * time.Second).Unix(),
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("provisioning-key", WithBaseURL(server.URL))

	apiKey, err := client.CreateAPIKey(context.Background(), models.CreateAPIKeyRequest{
		Name:        "New API Key",
		Permissions: []string{"chat.completions", "models.list"},
		RateLimit:   100,
		ExpiresIn:   30 * 24 * 60 * 60, // 30 days
	})

	require.NoError(t, err)
	assert.Equal(t, "key-789", apiKey.ID)
	assert.Equal(t, "New API Key", apiKey.Name)
	assert.NotEmpty(t, apiKey.Key)
}

func TestGetAPIKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/api/v1/keys/key-123", r.URL.Path)

		resp := models.APIKey{
			ID:          "key-123",
			Name:        "Production Key",
			Key:         "sk-or-v1-xxx",
			CreatedAt:   time.Now().Unix(),
			LastUsed:    time.Now().Unix(),
			UsageCount:  1000,
			Disabled:    false,
			Permissions: []string{"chat.completions", "models.list"},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("provisioning-key", WithBaseURL(server.URL))

	apiKey, err := client.GetAPIKey(context.Background(), "key-123")
	require.NoError(t, err)
	assert.Equal(t, "key-123", apiKey.ID)
	assert.Equal(t, "Production Key", apiKey.Name)
}

func TestGetCurrentAPIKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/api/v1/me/keys", r.URL.Path)

		// Verify the API key is sent in the header
		assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))

		resp := models.APIKey{
			ID:          "key-current",
			Name:        "Current Key",
			Key:         "sk-or-v1-current",
			CreatedAt:   time.Now().Unix(),
			Permissions: []string{"chat.completions"},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))

	apiKey, err := client.GetCurrentAPIKey(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "key-current", apiKey.ID)
	assert.Equal(t, "Current Key", apiKey.Name)
}

func TestUpdateAPIKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PATCH", r.Method)
		assert.Equal(t, "/api/v1/keys/key-123", r.URL.Path)

		var req models.UpdateAPIKeyRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)

		assert.Equal(t, "Updated Key Name", *req.Name)
		assert.Equal(t, true, *req.Disabled)
		assert.Equal(t, 200, *req.RateLimit)

		resp := models.APIKey{
			ID:          "key-123",
			Name:        *req.Name,
			Key:         "sk-or-v1-xxx",
			CreatedAt:   time.Now().Unix(),
			Disabled:    *req.Disabled,
			RateLimit:   *req.RateLimit,
			Permissions: []string{"chat.completions"},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("provisioning-key", WithBaseURL(server.URL))

	name := "Updated Key Name"
	disabled := true
	rateLimit := 200

	apiKey, err := client.UpdateAPIKey(context.Background(), "key-123", models.UpdateAPIKeyRequest{
		Name:      &name,
		Disabled:  &disabled,
		RateLimit: &rateLimit,
	})

	require.NoError(t, err)
	assert.Equal(t, "Updated Key Name", apiKey.Name)
	assert.True(t, apiKey.Disabled)
	assert.Equal(t, 200, apiKey.RateLimit)
}

func TestDeleteAPIKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		assert.Equal(t, "/api/v1/keys/key-123", r.URL.Path)

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient("provisioning-key", WithBaseURL(server.URL))

	err := client.DeleteAPIKey(context.Background(), "key-123")
	require.NoError(t, err)
}

// Credits and Usage Tests

func TestGetCredits(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/api/v1/me/credits", r.URL.Path)

		resp := models.CreditsResponse{
			TotalCredits:     1000.50,
			UsedCredits:      250.25,
			RemainingCredits: 750.25,
			ResetDate:        time.Now().Add(30 * 24 * time.Hour).Unix(),
			CreditLimit:      5000.00,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))

	credits, err := client.GetCredits(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1000.50, credits.TotalCredits)
	assert.Equal(t, 250.25, credits.UsedCredits)
	assert.Equal(t, 750.25, credits.RemainingCredits)
}

// Providers Tests

func TestListProviders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/api/v1/providers", r.URL.Path)

		resp := models.ProvidersResponse{
			Data: []models.Provider{
				{
					ID:          "openai",
					Name:        "OpenAI",
					Description: "OpenAI API provider",
					Status:      "operational",
					Models: []string{
						"gpt-4",
						"gpt-3.5-turbo",
					},
				},
				{
					ID:          "anthropic",
					Name:        "Anthropic",
					Description: "Anthropic API provider",
					Status:      "operational",
					Models: []string{
						"claude-3-opus",
						"claude-3-sonnet",
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))

	providers, err := client.ListProviders(context.Background())
	require.NoError(t, err)
	assert.Len(t, providers.Data, 2)
	assert.Equal(t, "openai", providers.Data[0].ID)
	assert.Equal(t, "operational", providers.Data[0].Status)
}

// Model Endpoints Tests

func TestListModelEndpoints(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/api/v1/endpoints/openai/gpt-4", r.URL.Path)

		resp := models.ModelEndpointsResponse{
			Data: []models.ModelEndpoint{
				{
					ID:         "endpoint-1",
					Provider:   "openai",
					Model:      "gpt-4",
					URL:        "https://api.openai.com/v1",
					Region:     "us-east-1",
					Latency:    120,
					Throughput: 1000,
					Status:     "healthy",
				},
				{
					ID:         "endpoint-2",
					Provider:   "azure",
					Model:      "gpt-4",
					URL:        "https://azure.openai.com/v1",
					Region:     "eu-west-1",
					Latency:    150,
					Throughput: 800,
					Status:     "healthy",
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))

	endpoints, err := client.ListModelEndpoints(context.Background(), "openai/gpt-4")
	require.NoError(t, err)
	assert.Len(t, endpoints.Data, 2)
	assert.Equal(t, "endpoint-1", endpoints.Data[0].ID)
	assert.Equal(t, "healthy", endpoints.Data[0].Status)
}

// OAuth/PKCE Tests

func TestExchangeAuthCodeForAPIKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/api/v1/auth/keys", r.URL.Path)

		var req models.ExchangeAuthCodeRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)

		assert.Equal(t, "auth-code-123", req.Code)
		assert.Equal(t, "verifier-123", req.CodeVerifier)
		assert.Equal(t, "https://myapp.com/callback", req.RedirectURI)

		resp := models.ExchangeAuthCodeResponse{
			APIKey: models.APIKey{
				ID:          "key-oauth",
				Name:        "OAuth Generated Key",
				Key:         "sk-or-v1-oauth",
				CreatedAt:   time.Now().Unix(),
				Permissions: []string{"chat.completions"},
			},
			ExpiresIn: 3600,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("", WithBaseURL(server.URL)) // No API key needed for auth exchange

	resp, err := client.ExchangeAuthCodeForAPIKey(context.Background(), models.ExchangeAuthCodeRequest{
		Code:         "auth-code-123",
		CodeVerifier: "verifier-123",
		RedirectURI:  "https://myapp.com/callback",
	})

	require.NoError(t, err)
	assert.Equal(t, "key-oauth", resp.APIKey.ID)
	assert.Equal(t, 3600, resp.ExpiresIn)
}

// Coinbase Commerce Tests

func TestCreateCoinbaseCharge(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/api/v1/me/coinbase-charge", r.URL.Path)

		var req models.CreateCoinbaseChargeRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)

		assert.Equal(t, 50.0, req.Amount)
		assert.Equal(t, "USD", req.Currency)
		assert.Equal(t, "Credits purchase", req.Description)

		resp := models.CoinbaseChargeResponse{
			ID:          "charge-123",
			Code:        "CHARGE123",
			Name:        "OpenRouter Credits",
			Description: req.Description,
			Amount:      req.Amount,
			Currency:    req.Currency,
			Status:      "pending",
			CheckoutURL: "https://commerce.coinbase.com/charges/CHARGE123",
			ExpiresAt:   time.Now().Add(15 * time.Minute).Unix(),
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))

	charge, err := client.CreateCoinbaseCharge(context.Background(), models.CreateCoinbaseChargeRequest{
		Amount:      50.0,
		Currency:    "USD",
		Description: "Credits purchase",
	})

	require.NoError(t, err)
	assert.Equal(t, "charge-123", charge.ID)
	assert.Equal(t, "pending", charge.Status)
	assert.NotEmpty(t, charge.CheckoutURL)
}
