package e2e

import (
	"context"
	"encoding/json"

	"github.com/rizome-dev/go-openrouter/pkg/models"
	"github.com/rizome-dev/go-openrouter/pkg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (suite *E2ETestSuite) TestStructuredOutputWithSchema() {
	ctx := context.Background()

	req := models.ChatCompletionRequest{
		Model: "google/gemini-2.5-flash",
		Messages: []models.Message{
			models.NewTextMessage(models.RoleUser, "Create a book summary for '1984' by George Orwell. Return only JSON in the exact format: {\"title\": \"string\", \"author\": \"string\", \"summary\": \"string\", \"rating\": number, \"genres\": [\"string\"]}"),
		},
		ResponseFormat: &models.ResponseFormat{
			Type: "json_object",
		},
		MaxTokens:   intPtr(200),
		Temperature: float64Ptr(0.3),
	}

	resp, err := suite.client.CreateChatCompletion(ctx, req)
	require.NoError(suite.T(), err)

	// Parse the structured response
	content, err := resp.Choices[0].Message.GetTextContent()
	assert.NoError(suite.T(), err)

	var result struct {
		Title   string   `json:"title"`
		Author  string   `json:"author"`
		Summary string   `json:"summary"`
		Rating  float64  `json:"rating"`
		Genres  []string `json:"genres"`
	}

	err = json.Unmarshal([]byte(content), &result)
	require.NoError(suite.T(), err)

	// Validate the structured output
	assert.Contains(suite.T(), result.Title, "1984")
	assert.Contains(suite.T(), result.Author, "Orwell")
	assert.NotEmpty(suite.T(), result.Summary)
	assert.GreaterOrEqual(suite.T(), result.Rating, float64(1))
	assert.LessOrEqual(suite.T(), result.Rating, float64(5))
	assert.Greater(suite.T(), len(result.Genres), 0)
}

func (suite *E2ETestSuite) TestStructuredOutputWithGoStruct() {
	ctx := context.Background()

	// Define a Go struct
	type WeatherReport struct {
		Location    string  `json:"location" description:"City name"`
		Temperature float64 `json:"temperature" description:"Temperature in Celsius"`
		Conditions  string  `json:"conditions" description:"Weather conditions"`
		Humidity    int     `json:"humidity,omitempty" description:"Humidity percentage"`
		WindSpeed   float64 `json:"wind_speed,omitempty" description:"Wind speed in km/h"`
	}

	req := models.ChatCompletionRequest{
		Model: "google/gemini-2.5-flash",
		Messages: []models.Message{
			models.NewTextMessage(models.RoleUser, "What's the weather like in London today? Make it realistic. Return only JSON in the exact format: {\"location\": \"string\", \"temperature\": number, \"conditions\": \"string\", \"humidity\": number, \"wind_speed\": number}"),
		},
		ResponseFormat: &models.ResponseFormat{
			Type: "json_object",
		},
		Temperature: float64Ptr(0.3),
	}

	resp, err := suite.client.CreateChatCompletion(ctx, req)
	require.NoError(suite.T(), err)

	// Parse using the struct
	var weather WeatherReport
	err = pkg.ParseStructuredResponse(resp, &weather)
	require.NoError(suite.T(), err)

	// Validate
	assert.Contains(suite.T(), weather.Location, "London")
	assert.Greater(suite.T(), weather.Temperature, float64(-20))
	assert.Less(suite.T(), weather.Temperature, float64(50))
	assert.NotEmpty(suite.T(), weather.Conditions)
}

func (suite *E2ETestSuite) TestStructuredOutputArray() {
	ctx := context.Background()

	req := models.ChatCompletionRequest{
		Model: "google/gemini-2.5-flash",
		Messages: []models.Message{
			models.NewTextMessage(models.RoleUser, "Create a todo list with 4 tasks for building a web application. Return only JSON in the exact format: {\"tasks\": [{\"id\": 1, \"description\": \"string\", \"priority\": \"low|medium|high\"}]}"),
		},
		ResponseFormat: &models.ResponseFormat{
			Type: "json_object",
		},
		Temperature: float64Ptr(0.5),
	}

	resp, err := suite.client.CreateChatCompletion(ctx, req)
	require.NoError(suite.T(), err)

	content, err := resp.Choices[0].Message.GetTextContent()
	assert.NoError(suite.T(), err)

	var result struct {
		Tasks []struct {
			ID          int    `json:"id"`
			Description string `json:"description"`
			Priority    string `json:"priority"`
		} `json:"tasks"`
	}

	err = json.Unmarshal([]byte(content), &result)
	require.NoError(suite.T(), err)

	// Validate
	assert.Len(suite.T(), result.Tasks, 4)
	for i, task := range result.Tasks {
		assert.Equal(suite.T(), i+1, task.ID)
		assert.NotEmpty(suite.T(), task.Description)
		assert.Contains(suite.T(), []string{"low", "medium", "high"}, task.Priority)
	}
}

func (suite *E2ETestSuite) TestStructuredOutputHelper() {
	ctx := context.Background()

	// Define analysis schema
	type Analysis struct {
		Sentiment  string   `json:"sentiment" description:"positive, negative, or neutral"`
		Score      float64  `json:"score" description:"Sentiment score from -1 to 1"`
		Keywords   []string `json:"keywords" description:"Key words or phrases"`
		Summary    string   `json:"summary" description:"Brief summary"`
		Confidence float64  `json:"confidence" description:"Confidence level 0-1"`
	}

	req := models.ChatCompletionRequest{
		Model: "google/gemini-2.5-flash",
		Messages: []models.Message{
			models.NewTextMessage(models.RoleUser,
				"Analyze this text: 'The new product launch was incredibly successful. "+
					"Sales exceeded expectations by 200% and customer feedback has been overwhelmingly positive.' Return only JSON in the exact format: {\"sentiment\": \"string\", \"score\": number, \"keywords\": [\"string\"], \"summary\": \"string\", \"confidence\": number}"),
		},
		Temperature: float64Ptr(0.3),
	}

	req.ResponseFormat = &models.ResponseFormat{
		Type: "json_object",
	}

	resp, err := suite.client.CreateChatCompletion(ctx, req)
	require.NoError(suite.T(), err)

	var analysis Analysis
	content, err := resp.Choices[0].Message.GetTextContent()
	require.NoError(suite.T(), err)

	err = json.Unmarshal([]byte(content), &analysis)
	require.NoError(suite.T(), err)

	// Validate
	assert.Equal(suite.T(), "positive", analysis.Sentiment)
	assert.Greater(suite.T(), analysis.Score, float64(0))
	assert.Greater(suite.T(), len(analysis.Keywords), 0)
	assert.NotEmpty(suite.T(), analysis.Summary)
	assert.Greater(suite.T(), analysis.Confidence, float64(0.5))
}

func (suite *E2ETestSuite) TestStructuredOutputNested() {
	ctx := context.Background()

	// Complex nested structure
	type Address struct {
		Street  string `json:"street"`
		City    string `json:"city"`
		Country string `json:"country"`
		ZipCode string `json:"zip_code,omitempty"`
	}

	type Person struct {
		Name    string   `json:"name"`
		Age     int      `json:"age"`
		Email   string   `json:"email"`
		Address Address  `json:"address"`
		Hobbies []string `json:"hobbies"`
	}

	type Company struct {
		Name      string   `json:"name"`
		Founded   int      `json:"founded"`
		CEO       Person   `json:"ceo"`
		Employees []Person `json:"employees"`
		Revenue   float64  `json:"revenue,omitempty"`
	}

	req := models.ChatCompletionRequest{
		Model: "google/gemini-2.5-flash",
		Messages: []models.Message{
			models.NewTextMessage(models.RoleUser,
				"Create a fictional tech company with a CEO and 2 employees. Make it realistic. Return only JSON in the exact format: {\"name\": \"string\", \"founded\": number, \"ceo\": {\"name\": \"string\", \"age\": number, \"email\": \"string\", \"address\": {\"street\": \"string\", \"city\": \"string\", \"country\": \"string\", \"zip_code\": \"string\"}, \"hobbies\": [\"string\"]}, \"employees\": [...], \"revenue\": number}"),
		},
		Temperature: float64Ptr(0.7),
		MaxTokens:   intPtr(500),
	}

	req.ResponseFormat = &models.ResponseFormat{
		Type: "json_object",
	}

	resp, err := suite.client.CreateChatCompletion(ctx, req)
	require.NoError(suite.T(), err)

	var company Company
	content, err := resp.Choices[0].Message.GetTextContent()
	require.NoError(suite.T(), err)

	err = json.Unmarshal([]byte(content), &company)
	require.NoError(suite.T(), err)

	// Validate structure
	assert.NotEmpty(suite.T(), company.Name)
	assert.Greater(suite.T(), company.Founded, 1900)
	assert.LessOrEqual(suite.T(), company.Founded, 2024)

	// CEO validation
	assert.NotEmpty(suite.T(), company.CEO.Name)
	assert.Greater(suite.T(), company.CEO.Age, 20)
	assert.Contains(suite.T(), company.CEO.Email, "@")
	assert.NotEmpty(suite.T(), company.CEO.Address.City)

	// Employees validation
	assert.Len(suite.T(), company.Employees, 2)
	for _, emp := range company.Employees {
		assert.NotEmpty(suite.T(), emp.Name)
		assert.Greater(suite.T(), emp.Age, 18)
		assert.Contains(suite.T(), emp.Email, "@")
	}
}
