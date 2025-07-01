package models

// APIKey represents an API key
type APIKey struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Key         string   `json:"key"`
	CreatedAt   int64    `json:"created_at"`
	LastUsed    int64    `json:"last_used,omitempty"`
	UsageCount  int      `json:"usage_count,omitempty"`
	Disabled    bool     `json:"disabled"`
	Permissions []string `json:"permissions"`
	RateLimit   int      `json:"rate_limit,omitempty"`
	ExpiresAt   int64    `json:"expires_at,omitempty"`
}

// APIKeysResponse represents the response from listing API keys
type APIKeysResponse struct {
	Data []APIKey `json:"data"`
}

// CreateAPIKeyRequest represents a request to create an API key
type CreateAPIKeyRequest struct {
	Name        string   `json:"name"`
	Permissions []string `json:"permissions"`
	RateLimit   int      `json:"rate_limit,omitempty"`
	ExpiresIn   int      `json:"expires_in,omitempty"` // Seconds until expiration
}

// UpdateAPIKeyRequest represents a request to update an API key
type UpdateAPIKeyRequest struct {
	Name        *string   `json:"name,omitempty"`
	Disabled    *bool     `json:"disabled,omitempty"`
	RateLimit   *int      `json:"rate_limit,omitempty"`
	Permissions *[]string `json:"permissions,omitempty"`
}

// ExchangeAuthCodeRequest represents a request to exchange an auth code for an API key
type ExchangeAuthCodeRequest struct {
	Code         string `json:"code"`
	CodeVerifier string `json:"code_verifier"`
	RedirectURI  string `json:"redirect_uri"`
}

// ExchangeAuthCodeResponse represents the response from exchanging an auth code
type ExchangeAuthCodeResponse struct {
	APIKey    APIKey `json:"api_key"`
	ExpiresIn int    `json:"expires_in"` // Seconds until expiration
}

// CreditsResponse represents the user's credit information
type CreditsResponse struct {
	TotalCredits     float64 `json:"total_credits"`
	UsedCredits      float64 `json:"used_credits"`
	RemainingCredits float64 `json:"remaining_credits"`
	ResetDate        int64   `json:"reset_date"`
	CreditLimit      float64 `json:"credit_limit"`
}

// CreateCoinbaseChargeRequest represents a request to create a Coinbase charge
type CreateCoinbaseChargeRequest struct {
	Amount      float64 `json:"amount"`
	Currency    string  `json:"currency"`
	Description string  `json:"description"`
}

// CoinbaseChargeResponse represents a Coinbase charge response
type CoinbaseChargeResponse struct {
	ID          string  `json:"id"`
	Code        string  `json:"code"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Amount      float64 `json:"amount"`
	Currency    string  `json:"currency"`
	Status      string  `json:"status"`
	CheckoutURL string  `json:"checkout_url"`
	ExpiresAt   int64   `json:"expires_at"`
}