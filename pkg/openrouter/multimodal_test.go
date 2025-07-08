package openrouter

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/rizome-dev/go-openrouter/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMultiModalHelper_CreateWithImage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		var req models.ChatCompletionRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)

		// Check message structure
		assert.Equal(t, 1, len(req.Messages))
		msg := req.Messages[0]

		// Verify content is multi-part
		var contents []interface{}
		err = json.Unmarshal(msg.Content, &contents)
		require.NoError(t, err)
		assert.Equal(t, 2, len(contents))

		// Send response
		resp := models.ChatCompletionResponse{
			ID:    "resp-123",
			Model: "test-model",
			Choices: []models.Choice{
				{
					Message: &models.Message{
						Role:    models.RoleAssistant,
						Content: json.RawMessage(`"I see an image"`),
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	helper := NewMultiModalHelper(client)

	tests := []struct {
		name  string
		image ImageInput
	}{
		{
			name: "Image from URL",
			image: ImageInput{
				URL: "https://example.com/image.jpg",
			},
		},
		{
			name: "Image from base64 data",
			image: ImageInput{
				Data: []byte("test image data"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := helper.CreateWithImage(context.Background(), "What's in this image?", tt.image, "test-model")
			require.NoError(t, err)
			assert.NotNil(t, resp)

			content, err := resp.Choices[0].Message.GetTextContent()
			assert.NoError(t, err)
			assert.Equal(t, "I see an image", content)
		})
	}
}

func TestMultiModalHelper_CreateWithImages(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req models.ChatCompletionRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)

		// Verify multiple images
		var contents []interface{}
		err = json.Unmarshal(req.Messages[0].Content, &contents)
		require.NoError(t, err)
		assert.Equal(t, 3, len(contents)) // 1 text + 2 images

		resp := models.ChatCompletionResponse{
			ID:    "resp-123",
			Model: "test-model",
			Choices: []models.Choice{
				{
					Message: &models.Message{
						Role:    models.RoleAssistant,
						Content: json.RawMessage(`"I see multiple images"`),
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	helper := NewMultiModalHelper(client)

	images := []ImageInput{
		{URL: "https://example.com/image1.jpg"},
		{Data: []byte("image data 2")},
	}

	resp, err := helper.CreateWithImages(context.Background(), "Compare these images", images, "test-model")
	require.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestMultiModalHelper_CreateWithPDF(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req models.ChatCompletionRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)

		// Verify PDF content and plugin
		var contents []interface{}
		err = json.Unmarshal(req.Messages[0].Content, &contents)
		require.NoError(t, err)
		assert.Equal(t, 2, len(contents)) // text + PDF

		// Check for PDF plugin
		if len(req.Plugins) > 0 {
			plugin := req.Plugins[0]
			assert.Equal(t, "file-parser", plugin.ID)
			if plugin.PDF != nil {
				assert.Equal(t, models.PDFEngineMistralOCR, plugin.PDF.Engine)
			}
		}

		resp := models.ChatCompletionResponse{
			ID:    "resp-123",
			Model: "test-model",
			Choices: []models.Choice{
				{
					Message: &models.Message{
						Role:    models.RoleAssistant,
						Content: json.RawMessage(`"PDF processed"`),
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	helper := NewMultiModalHelper(client)

	pdf := PDFInput{
		Data:     []byte("PDF content"),
		Filename: "test.pdf",
		Engine:   models.PDFEngineMistralOCR,
	}

	resp, err := helper.CreateWithPDF(context.Background(), "Summarize this PDF", pdf, "test-model")
	require.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestMultiModalHelper_CreateWithMixed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req models.ChatCompletionRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)

		// Verify mixed content
		var contents []interface{}
		err = json.Unmarshal(req.Messages[0].Content, &contents)
		require.NoError(t, err)
		assert.Equal(t, 4, len(contents)) // 1 text + 1 image + 2 PDFs

		resp := models.ChatCompletionResponse{
			ID:    "resp-123",
			Model: "test-model",
			Choices: []models.Choice{
				{
					Message: &models.Message{
						Role:    models.RoleAssistant,
						Content: json.RawMessage(`"Mixed content processed"`),
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	helper := NewMultiModalHelper(client)

	images := []ImageInput{{URL: "https://example.com/image.jpg"}}
	pdfs := []PDFInput{
		{Data: []byte("PDF 1"), Filename: "doc1.pdf"},
		{Data: []byte("PDF 2"), Filename: "doc2.pdf", Engine: models.PDFEngineText},
	}

	resp, err := helper.CreateWithMixed(context.Background(), "Analyze all files", images, pdfs, "test-model")
	require.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestImageContentTypeDetection(t *testing.T) {
	helper := &MultiModalHelper{}

	tests := []struct {
		name     string
		data     []byte
		expected string
	}{
		{
			name:     "PNG signature",
			data:     []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x00},
			expected: "image/png",
		},
		{
			name:     "JPEG signature",
			data:     []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46, 0x00, 0x01},
			expected: "image/jpeg",
		},
		{
			name:     "WebP signature",
			data:     []byte("RIFF----WEBP"),
			expected: "image/webp",
		},
		{
			name:     "Unknown format",
			data:     []byte{0x00, 0x00, 0x00, 0x00},
			expected: "image/jpeg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := helper.detectImageContentType(tt.data)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestImagePathContentType(t *testing.T) {
	helper := &MultiModalHelper{}

	tests := []struct {
		path     string
		expected string
	}{
		{"image.png", "image/png"},
		{"IMAGE.PNG", "image/png"},
		{"photo.jpg", "image/jpeg"},
		{"photo.jpeg", "image/jpeg"},
		{"image.webp", "image/webp"},
		{"document.pdf", "image/jpeg"}, // Default for unknown
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := helper.getImageContentType(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLoadImageFromFile(t *testing.T) {
	// Create temp image file
	tmpDir := t.TempDir()
	imagePath := filepath.Join(tmpDir, "test.jpg")
	err := os.WriteFile(imagePath, []byte("fake image data"), 0644)
	require.NoError(t, err)

	// Test loading existing file
	img, err := LoadImageFromFile(imagePath)
	assert.NoError(t, err)
	assert.Equal(t, imagePath, img.Path)

	// Test loading non-existent file
	_, err = LoadImageFromFile(filepath.Join(tmpDir, "nonexistent.jpg"))
	assert.Error(t, err)
}

func TestLoadPDFFromFile(t *testing.T) {
	// Create temp PDF file
	tmpDir := t.TempDir()
	pdfPath := filepath.Join(tmpDir, "test.pdf")
	err := os.WriteFile(pdfPath, []byte("fake PDF data"), 0644)
	require.NoError(t, err)

	// Test loading existing file
	pdf, err := LoadPDFFromFile(pdfPath, models.PDFEngineMistralOCR)
	assert.NoError(t, err)
	assert.Equal(t, pdfPath, pdf.Path)
	assert.Equal(t, models.PDFEngineMistralOCR, pdf.Engine)

	// Test loading non-existent file
	_, err = LoadPDFFromFile(filepath.Join(tmpDir, "nonexistent.pdf"), models.PDFEngineText)
	assert.Error(t, err)
}

func TestPrepareImageContent(t *testing.T) {
	helper := &MultiModalHelper{}

	// Test with URL
	img := ImageInput{URL: "https://example.com/image.jpg", Detail: "high"}
	content, err := helper.prepareImageContent(img)
	require.NoError(t, err)

	imageContent, ok := content.(models.ImageContent)
	require.True(t, ok)
	assert.Equal(t, "https://example.com/image.jpg", imageContent.ImageURL.URL)
	assert.Equal(t, "high", imageContent.ImageURL.Detail)

	// Test with data
	img = ImageInput{Data: []byte("test data")}
	content, err = helper.prepareImageContent(img)
	require.NoError(t, err)

	imageContent, ok = content.(models.ImageContent)
	require.True(t, ok)
	assert.Contains(t, imageContent.ImageURL.URL, "data:image/jpeg;base64,")
	assert.Equal(t, "auto", imageContent.ImageURL.Detail) // Default

	// Test with no source
	img = ImageInput{}
	_, err = helper.prepareImageContent(img)
	assert.Error(t, err)
}

func TestPreparePDFContent(t *testing.T) {
	helper := &MultiModalHelper{}

	// Test with data
	pdf := PDFInput{
		Data:     []byte("PDF content"),
		Filename: "document.pdf",
	}
	content, err := helper.preparePDFContent(pdf)
	require.NoError(t, err)

	fileContent, ok := content.(models.FileContent)
	require.True(t, ok)
	assert.Equal(t, "document.pdf", fileContent.File.Filename)
	assert.Contains(t, fileContent.File.FileData, "data:application/pdf;base64,")

	// Test with no filename
	pdf = PDFInput{Data: []byte("PDF content")}
	content, err = helper.preparePDFContent(pdf)
	require.NoError(t, err)

	fileContent, ok = content.(models.FileContent)
	require.True(t, ok)
	assert.Equal(t, "document.pdf", fileContent.File.Filename) // Default

	// Test with no source
	pdf = PDFInput{}
	_, err = helper.preparePDFContent(pdf)
	assert.Error(t, err)
}

func TestMultiModalWithAnnotations(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return response with file annotations
		resp := models.ChatCompletionResponse{
			ID:    "resp-123",
			Model: "test-model",
			Choices: []models.Choice{
				{
					Message: &models.Message{
						Role:    models.RoleAssistant,
						Content: json.RawMessage(`"The PDF contains important information"`),
						Annotations: []models.Annotation{
							{
								Type: models.AnnotationTypeFile,
								File: &models.FileAnnotation{
									Filename: "document.pdf",
									FileData: map[string]interface{}{
										"id": "file-123",
									},
								},
							},
						},
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	helper := NewMultiModalHelper(client)

	pdf := PDFInput{
		Data:     []byte("PDF content"),
		Filename: "document.pdf",
	}

	resp, err := helper.CreateWithPDF(context.Background(), "Analyze this PDF", pdf, "test-model")
	require.NoError(t, err)
	assert.NotNil(t, resp)

	// Check annotations
	assert.Len(t, resp.Choices[0].Message.Annotations, 1)
	assert.Equal(t, models.AnnotationTypeFile, resp.Choices[0].Message.Annotations[0].Type)
}

func TestBase64Encoding(t *testing.T) {
	testData := []byte("Test data for encoding")
	encoded := base64.StdEncoding.EncodeToString(testData)

	// Verify it's valid base64
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	require.NoError(t, err)
	assert.Equal(t, testData, decoded)

	// Test in image content
	helper := &MultiModalHelper{}
	img := ImageInput{Data: testData}
	content, err := helper.prepareImageContent(img)
	require.NoError(t, err)

	imageContent := content.(models.ImageContent)
	assert.Contains(t, imageContent.ImageURL.URL, encoded)
}
