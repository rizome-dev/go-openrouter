package e2e

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"

	"github.com/rizome-dev/go-openrouter/pkg/models"
	"github.com/rizome-dev/go-openrouter/pkg/openrouter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (suite *E2ETestSuite) TestImageFromURL() {
	ctx := context.Background()

	// Create message with image URL
	message, err := models.NewMultiContentMessage(models.RoleUser,
		models.TextContent{
			Type: models.ContentTypeText,
			Text: "What do you see in this image? Describe it briefly.",
		},
		models.ImageContent{
			Type: models.ContentTypeImageURL,
			ImageURL: models.ImageURL{
				URL: "https://upload.wikimedia.org/wikipedia/commons/thumb/3/3a/Cat03.jpg/200px-Cat03.jpg",
			},
		},
	)
	require.NoError(suite.T(), err)

	req := models.ChatCompletionRequest{
		Model:       "openai/gpt-4o-mini",
		Messages:    []models.Message{message},
		MaxTokens:   intPtr(100),
		Temperature: float64Ptr(0.3),
	}

	resp, err := suite.client.CreateChatCompletion(ctx, req)
	require.NoError(suite.T(), err)

	content, err := resp.Choices[0].Message.GetTextContent()
	assert.NoError(suite.T(), err)
	assert.NotEmpty(suite.T(), content)

	// Should mention a cat
	assert.Contains(suite.T(), strings.ToLower(content), "cat")
}

func (suite *E2ETestSuite) TestImageBase64() {
	ctx := context.Background()

	// Create a simple 1x1 red pixel PNG
	redPixelPNG := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, // PNG signature
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52, // IHDR chunk
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53,
		0xDE, 0x00, 0x00, 0x00, 0x0C, 0x49, 0x44, 0x41,
		0x54, 0x08, 0xD7, 0x63, 0xF8, 0xCF, 0xC0, 0x00,
		0x00, 0x03, 0x01, 0x01, 0x00, 0x18, 0xDD, 0x8D,
		0xB4, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4E,
		0x44, 0xAE, 0x42, 0x60, 0x82,
	}

	imageData := base64.StdEncoding.EncodeToString(redPixelPNG)

	message, err := models.NewMultiContentMessage(models.RoleUser,
		models.TextContent{
			Type: models.ContentTypeText,
			Text: "What color is this image?",
		},
		models.ImageContent{
			Type: models.ContentTypeImageURL,
			ImageURL: models.ImageURL{
				URL: "data:image/png;base64," + imageData,
			},
		},
	)
	require.NoError(suite.T(), err)

	req := models.ChatCompletionRequest{
		Model:       "openai/gpt-4o-mini",
		Messages:    []models.Message{message},
		MaxTokens:   intPtr(50),
		Temperature: float64Ptr(0.0),
	}

	resp, err := suite.client.CreateChatCompletion(ctx, req)
	require.NoError(suite.T(), err)

	content, err := resp.Choices[0].Message.GetTextContent()
	assert.NoError(suite.T(), err)
	assert.Contains(suite.T(), strings.ToLower(content), "red")
}

func (suite *E2ETestSuite) TestMultipleImages() {
	ctx := context.Background()

	helper := openrouter.NewMultiModalHelper(suite.client)

	// Two different images
	images := []openrouter.ImageInput{
		{URL: "https://upload.wikimedia.org/wikipedia/commons/thumb/3/3a/Cat03.jpg/200px-Cat03.jpg"},
		{URL: "https://upload.wikimedia.org/wikipedia/commons/thumb/archive/5/5a/20120703164907%21Black_Labrador_Retriever_portrait.jpg/200px-Black_Labrador_Retriever_portrait.jpg"},
	}

	resp, err := helper.CreateWithImages(ctx,
		"What animals do you see in these images? List them.",
		images,
		"openai/gpt-4o-mini",
	)
	require.NoError(suite.T(), err)

	content, err := resp.Choices[0].Message.GetTextContent()
	assert.NoError(suite.T(), err)
	assert.NotEmpty(suite.T(), content)

	// Should mention both cat and dog
	contentLower := strings.ToLower(content)
	assert.Contains(suite.T(), contentLower, "cat")
	assert.Contains(suite.T(), contentLower, "dog")
}

func (suite *E2ETestSuite) TestImageWithStructuredOutput() {
	ctx := context.Background()

	// Define structure for image analysis
	type ImageAnalysis struct {
		Description string   `json:"description"`
		Objects     []string `json:"objects"`
		Colors      []string `json:"dominant_colors"`
		Scene       string   `json:"scene_type"`
	}

	structured := openrouter.NewStructuredOutput(suite.client)

	// Create message with image
	message, err := models.NewMultiContentMessage(models.RoleUser,
		models.TextContent{
			Type: models.ContentTypeText,
			Text: "Analyze this image and provide structured information",
		},
		models.ImageContent{
			Type: models.ContentTypeImageURL,
			ImageURL: models.ImageURL{
				URL: "https://upload.wikimedia.org/wikipedia/commons/thumb/3/3a/Cat03.jpg/200px-Cat03.jpg",
			},
		},
	)
	require.NoError(suite.T(), err)

	req := models.ChatCompletionRequest{
		Model:       "openai/gpt-4o-mini",
		Messages:    []models.Message{message},
		Temperature: float64Ptr(0.3),
		MaxTokens:   intPtr(200),
	}

	resp, err := structured.CreateWithSchema(ctx, req, "image_analysis", ImageAnalysis{})
	require.NoError(suite.T(), err)

	var analysis ImageAnalysis
	err = openrouter.ParseStructuredResponse(resp, &analysis)
	require.NoError(suite.T(), err)

	// Validate structured response
	assert.NotEmpty(suite.T(), analysis.Description)
	assert.Greater(suite.T(), len(analysis.Objects), 0)
	assert.Contains(suite.T(), analysis.Objects, "cat")
	assert.Greater(suite.T(), len(analysis.Colors), 0)
	assert.NotEmpty(suite.T(), analysis.Scene)
}

func (suite *E2ETestSuite) TestImageWithTools() {
	ctx := context.Background()

	// Define a tool for image analysis
	identifyTool, err := models.NewTool("identify_object",
		"Identify and provide information about an object",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"object_name": map[string]interface{}{
					"type":        "string",
					"description": "Name of the identified object",
				},
				"confidence": map[string]interface{}{
					"type":        "number",
					"description": "Confidence level 0-1",
				},
				"category": map[string]interface{}{
					"type":        "string",
					"description": "Category of the object",
				},
			},
			"required": []string{"object_name", "confidence", "category"},
		},
	)
	require.NoError(suite.T(), err)

	// Create message with image
	message, err := models.NewMultiContentMessage(models.RoleUser,
		models.TextContent{
			Type: models.ContentTypeText,
			Text: "Identify the main subject in this image",
		},
		models.ImageContent{
			Type: models.ContentTypeImageURL,
			ImageURL: models.ImageURL{
				URL: "https://upload.wikimedia.org/wikipedia/commons/thumb/3/3a/Cat03.jpg/200px-Cat03.jpg",
			},
		},
	)
	require.NoError(suite.T(), err)

	req := models.ChatCompletionRequest{
		Model:      "openai/gpt-4o-mini",
		Messages:   []models.Message{message},
		Tools:      []models.Tool{*identifyTool},
		ToolChoice: models.ToolChoiceAuto,
		MaxTokens:  intPtr(150),
	}

	resp, err := suite.client.CreateChatCompletion(ctx, req)
	require.NoError(suite.T(), err)

	// Should have tool call
	if len(resp.Choices[0].Message.ToolCalls) > 0 {
		toolCall := resp.Choices[0].Message.ToolCalls[0]
		assert.Equal(suite.T(), "identify_object", toolCall.Function.Name)

		var args map[string]interface{}
		err = json.Unmarshal([]byte(toolCall.Function.Arguments), &args)
		require.NoError(suite.T(), err)

		assert.Contains(suite.T(), strings.ToLower(args["object_name"].(string)), "cat")
		assert.Equal(suite.T(), "animal", strings.ToLower(args["category"].(string)))
	}
}

// Note: PDF tests would require actual PDF handling which might need additional setup
// Skipping PDF tests for now as they require more complex test data preparation
