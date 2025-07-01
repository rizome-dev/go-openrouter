package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/rizome-dev/go-openrouter/pkg/openrouter"
)

func main() {
	// Get API key from environment
	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		log.Fatal("Please set OPENROUTER_API_KEY environment variable")
	}

	// Create client
	client := openrouter.NewClient(apiKey,
		openrouter.WithHTTPReferer("https://github.com/rizome-dev/go-openrouter"),
		openrouter.WithXTitle("OpenRouterGo Multi-Modal Example"),
	)

	// Example 1: Image analysis
	fmt.Println("=== Image Analysis Example ===")
	imageAnalysisExample(client)

	// Example 2: PDF processing
	fmt.Println("\n=== PDF Processing Example ===")
	pdfProcessingExample(client)

	// Example 3: Web search
	fmt.Println("\n=== Web Search Example ===")
	webSearchExample(client)

	// Example 4: Research agent
	fmt.Println("\n=== Research Agent Example ===")
	researchAgentExample(client)
}

func imageAnalysisExample(client *openrouter.Client) {
	helper := openrouter.NewMultiModalHelper(client)
	ctx := context.Background()

	// Example with image URL
	fmt.Println("\n--- Analyzing image from URL ---")
	
	imageURL := "https://upload.wikimedia.org/wikipedia/commons/thumb/d/dd/Gfp-wisconsin-madison-the-nature-boardwalk.jpg/2560px-Gfp-wisconsin-madison-the-nature-boardwalk.jpg"
	image := openrouter.ImageInput{
		URL:    imageURL,
		Detail: "high",
	}

	resp, err := helper.CreateWithImage(ctx,
		"What can you tell me about this image? Describe the scene in detail.",
		image,
		"google/gemini-2.0-flash-001",
	)

	if err != nil {
		log.Printf("Error analyzing image: %v", err)
		return
	}

	content, _ := resp.Choices[0].Message.GetTextContent()
	fmt.Printf("Analysis: %s\n", content)

	// Example with multiple images
	fmt.Println("\n--- Comparing multiple images ---")
	
	images := []openrouter.ImageInput{
		{URL: "https://upload.wikimedia.org/wikipedia/commons/thumb/d/dd/Gfp-wisconsin-madison-the-nature-boardwalk.jpg/320px-Gfp-wisconsin-madison-the-nature-boardwalk.jpg"},
		{URL: "https://upload.wikimedia.org/wikipedia/commons/thumb/a/aa/Polarlicht_2.jpg/320px-Polarlicht_2.jpg"},
	}

	resp, err = helper.CreateWithImages(ctx,
		"Compare these two images. What are the main differences between them?",
		images,
		"google/gemini-2.0-flash-001",
	)

	if err != nil {
		log.Printf("Error comparing images: %v", err)
		return
	}

	content, _ = resp.Choices[0].Message.GetTextContent()
	fmt.Printf("Comparison: %s\n", content)
}

func pdfProcessingExample(client *openrouter.Client) {

	// Note: This example requires an actual PDF file
	// For demonstration, we'll show the structure
	fmt.Println("\n--- PDF Processing Structure ---")
	fmt.Println("To process a PDF file:")
	fmt.Println("1. Load PDF from file: pdf := openrouter.LoadPDFFromFile('document.pdf', models.PDFEngineText)")
	fmt.Println("2. Process with helper: resp := helper.CreateWithPDF(ctx, 'Summarize this document', pdf, 'google/gemini-2.0-flash-001')")
	
	// Example of processing with annotations for cost savings
	fmt.Println("\n--- Reusing PDF Annotations ---")
	fmt.Println("After first processing, save annotations from response:")
	fmt.Println("annotations := resp.Choices[0].Message.Annotations")
	fmt.Println("Then reuse in subsequent requests to avoid reprocessing costs")
}

func webSearchExample(client *openrouter.Client) {
	helper := openrouter.NewWebSearchHelper(client)
	ctx := context.Background()

	// Simple web search
	fmt.Println("\n--- Simple Web Search ---")
	
	resp, err := helper.CreateWithWebSearch(ctx,
		"What are the latest developments in quantum computing in 2024?",
		"openai/gpt-4",
		nil, // Use default options
	)

	if err != nil {
		log.Printf("Error with web search: %v", err)
		return
	}

	content, _ := resp.Choices[0].Message.GetTextContent()
	fmt.Printf("Response: %s\n", content)

	// Extract and display citations
	citations := openrouter.ExtractCitations(resp)
	if len(citations) > 0 {
		fmt.Println("\nSources:")
		for _, citation := range citations {
			fmt.Printf("- %s: %s\n", citation.Title, citation.URL)
		}
	}

	// Custom web search options
	fmt.Println("\n--- Custom Web Search ---")
	
	resp, err = helper.CreateWithWebSearch(ctx,
		"Recent AI breakthroughs",
		"openai/gpt-4",
		&openrouter.SearchOptions{
			MaxResults:   10,
			SearchPrompt: "Latest AI research papers and breakthroughs from 2024:",
		},
	)

	if err != nil {
		log.Printf("Error with custom web search: %v", err)
		return
	}

	content, _ = resp.Choices[0].Message.GetTextContent()
	fmt.Printf("Response: %s\n", content[:min(500, len(content))] + "...")

	// Native web search with context size
	fmt.Println("\n--- Native Web Search ---")
	
	resp, err = helper.CreateWithNativeWebSearch(ctx,
		"OpenAI GPT models comparison",
		"openai/gpt-4o-search-preview",
		"high", // Use high search context
	)

	if err != nil {
		log.Printf("Error with native web search: %v", err)
		return
	}

	content, _ = resp.Choices[0].Message.GetTextContent()
	fmt.Printf("Response: %s\n", content[:min(500, len(content))] + "...")
}

func researchAgentExample(client *openrouter.Client) {
	helper := openrouter.NewWebSearchHelper(client)
	agent := helper.CreateResearchAgent("openai/gpt-4")
	ctx := context.Background()

	fmt.Println("\n--- Research Agent ---")
	fmt.Println("Researching: The Future of Renewable Energy")
	
	research, err := agent.Research(ctx, "The Future of Renewable Energy", 3)
	if err != nil {
		log.Printf("Research error: %v", err)
		return
	}

	// Display research results
	fmt.Printf("\n=== Research Topic: %s ===\n", research.Topic)
	
	if research.Summary != "" {
		fmt.Printf("\nSummary:\n%s\n", research.Summary)
	}

	for _, section := range research.Sections {
		fmt.Printf("\n--- %s ---\n", section.Title)
		fmt.Printf("%s\n", section.Content[:min(300, len(section.Content))] + "...")
		
		if len(section.Citations) > 0 {
			fmt.Println("\nReferences:")
			for _, citation := range section.Citations {
				fmt.Printf("- %s\n", citation.URL)
			}
		}
	}

	fmt.Printf("\nTotal citations collected: %d\n", len(research.Citations))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}