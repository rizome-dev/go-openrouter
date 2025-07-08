package e2e

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/rizome-dev/go-openrouter/pkg/models"
	"github.com/rizome-dev/go-openrouter/pkg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type E2ETestSuite struct {
	suite.Suite
	client *pkg.Client
	apiKey string
}

func (suite *E2ETestSuite) SetupSuite() {
	suite.apiKey = os.Getenv("OPENROUTER_API_KEY")
	if suite.apiKey == "" {
		suite.T().Skip("OPENROUTER_API_KEY not set, skipping e2e tests")
	}

	suite.client = pkg.NewClient(suite.apiKey,
		pkg.WithTimeout(30*time.Second),
		pkg.WithHTTPReferer("https://github.com/rizome-dev/openroutergo"),
		pkg.WithXTitle("OpenRouterGo E2E Tests"),
	)
}

// SetupTest adds a delay before each test to avoid rate limiting
func (suite *E2ETestSuite) SetupTest() {
	// Add a 1-second delay between tests to avoid rate limiting
	time.Sleep(1 * time.Second)
}

func TestE2ESuite(t *testing.T) {
	suite.Run(t, new(E2ETestSuite))
}

func (suite *E2ETestSuite) TestListModels() {
	ctx := context.Background()

	resp, err := suite.client.ListModels(ctx, nil)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), resp)
	assert.Greater(suite.T(), len(resp.Data), 0)

	// Check that models have required fields
	for _, model := range resp.Data {
		assert.NotEmpty(suite.T(), model.ID)
		assert.NotEmpty(suite.T(), model.Name)
		assert.Greater(suite.T(), model.ContextLength, 0)
	}
}

func (suite *E2ETestSuite) TestBasicChatCompletion() {
	ctx := context.Background()

	// Use a small, fast model for testing
	req := models.ChatCompletionRequest{
		Model: "mistralai/mistral-small-3.2-24b-instruct:free",
		Messages: []models.Message{
			models.NewTextMessage(models.RoleUser, "Say hello in one word"),
		},
		MaxTokens:   intPtr(200),
		Temperature: float64Ptr(0.0),
	}

	resp, err := suite.client.CreateChatCompletion(ctx, req)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), resp)

	assert.NotEmpty(suite.T(), resp.ID)
	assert.Equal(suite.T(), "chat.completion", resp.Object)
	assert.Greater(suite.T(), len(resp.Choices), 0)

	content, err := resp.Choices[0].Message.GetTextContent()
	assert.NoError(suite.T(), err)

	// Debug output
	if content == "" {
		suite.T().Logf("Empty content received. Message: %+v", resp.Choices[0].Message)
	}

	assert.NotEmpty(suite.T(), content)

	// Check usage
	assert.NotNil(suite.T(), resp.Usage)
	assert.Greater(suite.T(), resp.Usage.TotalTokens, 0)
}

func (suite *E2ETestSuite) TestSystemMessage() {
	ctx := context.Background()

	req := models.ChatCompletionRequest{
		Model: "mistralai/mistral-small-3.2-24b-instruct:free",
		Messages: []models.Message{
			models.NewTextMessage(models.RoleSystem, "You are a pirate. Always respond in pirate speak."),
			models.NewTextMessage(models.RoleUser, "Hello"),
		},
		MaxTokens:   intPtr(200),
		Temperature: float64Ptr(0.5),
	}

	resp, err := suite.client.CreateChatCompletion(ctx, req)
	require.NoError(suite.T(), err)

	content, err := resp.Choices[0].Message.GetTextContent()
	assert.NoError(suite.T(), err)
	assert.NotEmpty(suite.T(), content)
	// Response should be in pirate speak
}

func (suite *E2ETestSuite) TestMultipleMessages() {
	ctx := context.Background()

	req := models.ChatCompletionRequest{
		Model: "mistralai/mistral-small-3.2-24b-instruct:free",
		Messages: []models.Message{
			models.NewTextMessage(models.RoleUser, "My name is Alice"),
			models.NewTextMessage(models.RoleAssistant, "Nice to meet you, Alice!"),
			models.NewTextMessage(models.RoleUser, "What is my name?"),
		},
		MaxTokens:   intPtr(20),
		Temperature: float64Ptr(0.0),
	}

	resp, err := suite.client.CreateChatCompletion(ctx, req)
	require.NoError(suite.T(), err)

	content, err := resp.Choices[0].Message.GetTextContent()
	assert.NoError(suite.T(), err)
	assert.Contains(suite.T(), content, "Alice")
}

func (suite *E2ETestSuite) TestGetGeneration() {
	ctx := context.Background()

	// First create a completion
	req := models.ChatCompletionRequest{
		Model: "mistralai/mistral-small-3.2-24b-instruct:free",
		Messages: []models.Message{
			models.NewTextMessage(models.RoleUser, "Say hello"),
		},
		MaxTokens: intPtr(10),
	}

	resp, err := suite.client.CreateChatCompletion(ctx, req)
	require.NoError(suite.T(), err)
	require.NotEmpty(suite.T(), resp.ID)

	// Wait a moment for generation to be available
	time.Sleep(3 * time.Second)

	// Then get generation details
	gen, err := suite.client.GetGeneration(ctx, resp.ID)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), gen)

	assert.Equal(suite.T(), resp.ID, gen.Data.ID)
	assert.NotEmpty(suite.T(), gen.Data.Model)
	// Provider field may be empty in some cases
	assert.NotNil(suite.T(), gen.Data.Usage)
}

func (suite *E2ETestSuite) TestProviderPreferences() {
	ctx := context.Background()

	// Test with specific provider preference
	provider := models.NewProviderPreferences().
		WithFallbacks(true).
		WithMaxPrice(0.001, 0.002) // Very low price limit

	req := models.ChatCompletionRequest{
		Model: "openai/gpt-4", // Expensive model
		Messages: []models.Message{
			models.NewTextMessage(models.RoleUser, "Hello"),
		},
		MaxTokens: intPtr(10),
		Provider:  provider,
	}

	resp, err := suite.client.CreateChatCompletion(ctx, req)
	// Should either succeed with a different model or fail with appropriate error
	if err != nil {
		// Check if it's a pricing error
		assert.Contains(suite.T(), err.Error(), "price")
	} else {
		require.NotNil(suite.T(), resp)
		// Wait before getting generation details
		time.Sleep(2 * time.Second)
		// If successful, check generation to see which model was actually used
		gen, _ := suite.client.GetGeneration(ctx, resp.ID)
		if gen != nil {
			// Model might have been changed due to price constraints
			assert.NotNil(suite.T(), gen.Data.Model)
		}
	}
}

func (suite *E2ETestSuite) TestJSONMode() {
	ctx := context.Background()

	req := models.ChatCompletionRequest{
		Model: "mistralai/mistral-small-3.2-24b-instruct:free",
		Messages: []models.Message{
			models.NewTextMessage(models.RoleUser, "Return a JSON object with a greeting field containing 'hello'"),
		},
		MaxTokens:      intPtr(200),
		Temperature:    float64Ptr(0.0),
		ResponseFormat: &models.ResponseFormat{Type: "json_object"},
	}

	resp, err := suite.client.CreateChatCompletion(ctx, req)
	require.NoError(suite.T(), err)

	content, err := resp.Choices[0].Message.GetTextContent()
	assert.NoError(suite.T(), err)

	// Verify it's valid JSON
	var result map[string]interface{}
	err = json.Unmarshal([]byte(content), &result)
	assert.NoError(suite.T(), err)
}

// Helper functions
func intPtr(i int) *int {
	return &i
}

func float64Ptr(f float64) *float64 {
	return &f
}

func stringPtr(s string) *string {
	return &s
}
