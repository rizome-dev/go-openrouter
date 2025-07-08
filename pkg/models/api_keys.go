package models

import "time"

// APIKey represents an API key
type APIKey struct {
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Hash      string    `json:"hash"`
	Label     string    `json:"label,omitempty"`
	Name      string    `json:"name"`
	Disabled  bool      `json:"disabled"`
	Limit     float64   `json:"limit,omitempty"`
	Usage     float64   `json:"usage"`
	Key       string    `json:"key,omitempty"` // Only returned when creating a new key
}

// APIKeysResponse represents the response from listing API keys
type APIKeysResponse struct {
	Data []APIKey `json:"data"`
}

// CreateAPIKeyRequest represents a request to create an API key
type CreateAPIKeyRequest struct {
	Name               string  `json:"name"`
	Label              string  `json:"label,omitempty"`
	Limit              float64 `json:"limit,omitempty"`
	IncludeBYOKInLimit bool    `json:"include_byok_in_limit,omitempty"`
}

// UpdateAPIKeyRequest represents a request to update an API key
type UpdateAPIKeyRequest struct {
	Name     *string  `json:"name,omitempty"`
	Disabled *bool    `json:"disabled,omitempty"`
	Limit    *float64 `json:"limit,omitempty"`
}

// ExchangeAuthCodeRequest represents a request to exchange an auth code for an API key
type ExchangeAuthCodeRequest struct {
	Code                string `json:"code"`
	CodeVerifier        string `json:"code_verifier,omitempty"`
	CodeChallengeMethod string `json:"code_challenge_method,omitempty"`
}

// ExchangeAuthCodeResponse represents the response from exchanging an auth code
type ExchangeAuthCodeResponse struct {
	Key    string `json:"key"`
	UserID string `json:"user_id"`
}

// CreditsResponse represents the user's credit information
type CreditsResponse struct {
	Data struct {
		TotalCredits float64 `json:"total_credits"`
		TotalUsage   float64 `json:"total_usage"`
	} `json:"data"`
}

// CreateCoinbaseChargeRequest represents a request to create a Coinbase charge
type CreateCoinbaseChargeRequest struct {
	Amount  float64 `json:"amount"` // USD amount
	Sender  string  `json:"sender"` // Ethereum address
	ChainID int     `json:"chain_id"`
}

// CoinbaseChargeResponse represents a Coinbase charge response
type CoinbaseChargeResponse struct {
	Data struct {
		ID        string    `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		ExpiresAt time.Time `json:"expires_at"`
		Web3Data  struct {
			TransferIntent struct {
				Metadata struct {
					ChainID         int    `json:"chain_id"`
					ContractAddress string `json:"contract_address"`
					Sender          string `json:"sender"`
				} `json:"metadata"`
				CallData struct {
					RecipientAmount   string `json:"recipient_amount"`
					Deadline          string `json:"deadline"`
					Recipient         string `json:"recipient"`
					RecipientCurrency string `json:"recipient_currency"`
					RefundDestination string `json:"refund_destination"`
					FeeAmount         string `json:"fee_amount"`
					ID                string `json:"id"`
					Operator          string `json:"operator"`
					Signature         string `json:"signature"`
					Prefix            string `json:"prefix"`
				} `json:"call_data"`
			} `json:"transfer_intent"`
		} `json:"web3_data"`
	} `json:"data"`
}
