package pkg

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/rizome-dev/go-openrouter/pkg/models"
)

// MultiModalHelper provides utilities for working with multi-modal inputs
type MultiModalHelper struct {
	client *Client
}

// NewMultiModalHelper creates a new multi-modal helper
func NewMultiModalHelper(client *Client) *MultiModalHelper {
	return &MultiModalHelper{client: client}
}

// ImageInput represents an image input
type ImageInput struct {
	URL    string
	Path   string
	Data   []byte
	Detail string // "auto", "low", "high"
}

// PDFInput represents a PDF input
type PDFInput struct {
	Path     string
	Data     []byte
	Filename string
	Engine   models.PDFEngine
}

// CreateWithImage creates a chat completion with image input
func (m *MultiModalHelper) CreateWithImage(ctx context.Context, text string, image ImageInput, model string) (*models.ChatCompletionResponse, error) {
	// Create text content
	textContent := models.TextContent{
		Type: models.ContentTypeText,
		Text: text,
	}

	// Create image content
	imageContent, err := m.prepareImageContent(image)
	if err != nil {
		return nil, err
	}

	// Create message with multiple content parts
	message, err := models.NewMultiContentMessage(models.RoleUser, textContent, imageContent)
	if err != nil {
		return nil, err
	}

	// Create request
	req := models.ChatCompletionRequest{
		Model:    model,
		Messages: []models.Message{message},
	}

	return m.client.CreateChatCompletion(ctx, req)
}

// CreateWithImages creates a chat completion with multiple images
func (m *MultiModalHelper) CreateWithImages(ctx context.Context, text string, images []ImageInput, model string) (*models.ChatCompletionResponse, error) {
	// Create content parts
	contents := []models.Content{
		models.TextContent{
			Type: models.ContentTypeText,
			Text: text,
		},
	}

	// Add images
	for _, image := range images {
		imageContent, err := m.prepareImageContent(image)
		if err != nil {
			return nil, fmt.Errorf("failed to prepare image: %w", err)
		}
		contents = append(contents, imageContent)
	}

	// Create message
	message, err := models.NewMultiContentMessage(models.RoleUser, contents...)
	if err != nil {
		return nil, err
	}

	// Create request
	req := models.ChatCompletionRequest{
		Model:    model,
		Messages: []models.Message{message},
	}

	return m.client.CreateChatCompletion(ctx, req)
}

// CreateWithPDF creates a chat completion with PDF input
func (m *MultiModalHelper) CreateWithPDF(ctx context.Context, text string, pdf PDFInput, model string) (*models.ChatCompletionResponse, error) {
	// Create text content
	textContent := models.TextContent{
		Type: models.ContentTypeText,
		Text: text,
	}

	// Create PDF content
	pdfContent, err := m.preparePDFContent(pdf)
	if err != nil {
		return nil, err
	}

	// Create message
	message, err := models.NewMultiContentMessage(models.RoleUser, textContent, pdfContent)
	if err != nil {
		return nil, err
	}

	// Create request with PDF plugin
	req := models.ChatCompletionRequest{
		Model:    model,
		Messages: []models.Message{message},
	}

	// Add PDF plugin if engine is specified
	if pdf.Engine != "" {
		req.Plugins = []models.Plugin{
			*models.NewPDFPlugin(pdf.Engine),
		}
	}

	return m.client.CreateChatCompletion(ctx, req)
}

// CreateWithMixed creates a chat completion with mixed media
func (m *MultiModalHelper) CreateWithMixed(ctx context.Context, text string, images []ImageInput, pdfs []PDFInput, model string) (*models.ChatCompletionResponse, error) {
	// Create content parts
	contents := []models.Content{
		models.TextContent{
			Type: models.ContentTypeText,
			Text: text,
		},
	}

	// Add images
	for _, image := range images {
		imageContent, err := m.prepareImageContent(image)
		if err != nil {
			return nil, fmt.Errorf("failed to prepare image: %w", err)
		}
		contents = append(contents, imageContent)
	}

	// Add PDFs
	var plugins []models.Plugin
	for _, pdf := range pdfs {
		pdfContent, err := m.preparePDFContent(pdf)
		if err != nil {
			return nil, fmt.Errorf("failed to prepare PDF: %w", err)
		}
		contents = append(contents, pdfContent)

		// Add PDF plugin if needed
		if pdf.Engine != "" && len(plugins) == 0 {
			plugins = append(plugins, *models.NewPDFPlugin(pdf.Engine))
		}
	}

	// Create message
	message, err := models.NewMultiContentMessage(models.RoleUser, contents...)
	if err != nil {
		return nil, err
	}

	// Create request
	req := models.ChatCompletionRequest{
		Model:    model,
		Messages: []models.Message{message},
		Plugins:  plugins,
	}

	return m.client.CreateChatCompletion(ctx, req)
}

// prepareImageContent prepares image content from various sources
func (m *MultiModalHelper) prepareImageContent(image ImageInput) (models.Content, error) {
	var url string
	detail := image.Detail
	if detail == "" {
		detail = "auto"
	}

	if image.URL != "" {
		// Direct URL
		url = image.URL
	} else if image.Path != "" {
		// Read from file
		data, err := os.ReadFile(image.Path)
		if err != nil {
			return nil, fmt.Errorf("failed to read image file: %w", err)
		}

		// Determine content type
		contentType := m.getImageContentType(image.Path)

		// Create data URL
		url = fmt.Sprintf("data:%s;base64,%s", contentType, base64.StdEncoding.EncodeToString(data))
	} else if image.Data != nil {
		// Use provided data
		contentType := m.detectImageContentType(image.Data)
		url = fmt.Sprintf("data:%s;base64,%s", contentType, base64.StdEncoding.EncodeToString(image.Data))
	} else {
		return nil, fmt.Errorf("no image source provided")
	}

	return models.ImageContent{
		Type: models.ContentTypeImageURL,
		ImageURL: models.ImageURL{
			URL:    url,
			Detail: detail,
		},
	}, nil
}

// preparePDFContent prepares PDF content
func (m *MultiModalHelper) preparePDFContent(pdf PDFInput) (models.Content, error) {
	var data []byte
	filename := pdf.Filename

	if pdf.Path != "" {
		// Read from file
		var err error
		data, err = os.ReadFile(pdf.Path)
		if err != nil {
			return nil, fmt.Errorf("failed to read PDF file: %w", err)
		}
		if filename == "" {
			filename = filepath.Base(pdf.Path)
		}
	} else if pdf.Data != nil {
		// Use provided data
		data = pdf.Data
		if filename == "" {
			filename = "document.pdf"
		}
	} else {
		return nil, fmt.Errorf("no PDF source provided")
	}

	// Create data URL
	dataURL := fmt.Sprintf("data:application/pdf;base64,%s", base64.StdEncoding.EncodeToString(data))

	return models.FileContent{
		Type: models.ContentTypeFile,
		File: models.File{
			Filename: filename,
			FileData: dataURL,
		},
	}, nil
}

// getImageContentType determines content type from file extension
func (m *MultiModalHelper) getImageContentType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".webp":
		return "image/webp"
	default:
		return "image/jpeg" // Default
	}
}

// detectImageContentType detects content type from image data
func (m *MultiModalHelper) detectImageContentType(data []byte) string {
	if len(data) < 12 {
		return "image/jpeg" // Default
	}

	// Check PNG signature
	if data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4E && data[3] == 0x47 {
		return "image/png"
	}

	// Check JPEG signature
	if data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF {
		return "image/jpeg"
	}

	// Check WebP signature
	if len(data) >= 12 && string(data[0:4]) == "RIFF" && string(data[8:12]) == "WEBP" {
		return "image/webp"
	}

	return "image/jpeg" // Default
}

// LoadImageFromURL loads an image from a URL
func LoadImageFromURL(url string) (ImageInput, error) {
	return ImageInput{URL: url}, nil
}

// LoadImageFromFile loads an image from a file
func LoadImageFromFile(path string) (ImageInput, error) {
	// Check if file exists
	if _, err := os.Stat(path); err != nil {
		return ImageInput{}, fmt.Errorf("file not found: %w", err)
	}
	return ImageInput{Path: path}, nil
}

// LoadImageFromReader loads an image from a reader
func LoadImageFromReader(reader io.Reader) (ImageInput, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return ImageInput{}, fmt.Errorf("failed to read image data: %w", err)
	}
	return ImageInput{Data: data}, nil
}

// LoadPDFFromFile loads a PDF from a file
func LoadPDFFromFile(path string, engine models.PDFEngine) (PDFInput, error) {
	// Check if file exists
	if _, err := os.Stat(path); err != nil {
		return PDFInput{}, fmt.Errorf("file not found: %w", err)
	}
	return PDFInput{
		Path:   path,
		Engine: engine,
	}, nil
}

// LoadPDFFromReader loads a PDF from a reader
func LoadPDFFromReader(reader io.Reader, filename string, engine models.PDFEngine) (PDFInput, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return PDFInput{}, fmt.Errorf("failed to read PDF data: %w", err)
	}
	return PDFInput{
		Data:     data,
		Filename: filename,
		Engine:   engine,
	}, nil
}
