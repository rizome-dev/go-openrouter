package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/rizome-dev/openroutergo/pkg/models"
	"github.com/rizome-dev/openroutergo/pkg/openrouter"
)

func main() {
	// Define flags
	var (
		apiKey      = flag.String("key", os.Getenv("OPENROUTER_API_KEY"), "OpenRouter API key")
		model       = flag.String("model", "openai/gpt-3.5-turbo", "Model to use")
		stream      = flag.Bool("stream", false, "Enable streaming")
		temperature = flag.Float64("temp", 0.7, "Temperature (0-2)")
		maxTokens   = flag.Int("max-tokens", 0, "Maximum tokens (0 for default)")
		system      = flag.String("system", "", "System prompt")
		interactive = flag.Bool("i", false, "Interactive mode")
		listModels  = flag.Bool("list-models", false, "List available models")
		category    = flag.String("category", "", "Filter models by category")
		webSearch   = flag.Bool("web", false, "Enable web search")
	)

	flag.Parse()

	if *apiKey == "" {
		log.Fatal("API key required. Set OPENROUTER_API_KEY or use -key flag")
	}

	// Create client
	client := openrouter.NewClient(*apiKey,
		openrouter.WithHTTPReferer("https://github.com/rizome-dev/openroutergo"),
		openrouter.WithXTitle("OpenRouter CLI"),
	)

	ctx := context.Background()

	// Handle list models
	if *listModels {
		listAvailableModels(ctx, client, *category)
		return
	}

	// Handle interactive mode
	if *interactive {
		runInteractive(ctx, client, *model, *system, *temperature, *maxTokens, *stream, *webSearch)
		return
	}

	// Handle single prompt
	prompt := strings.Join(flag.Args(), " ")
	if prompt == "" {
		fmt.Println("Usage: openrouter-cli [options] <prompt>")
		fmt.Println("       openrouter-cli -i  (interactive mode)")
		fmt.Println("       openrouter-cli -list-models")
		flag.PrintDefaults()
		return
	}

	runSingle(ctx, client, prompt, *model, *system, *temperature, *maxTokens, *stream, *webSearch)
}

func listAvailableModels(ctx context.Context, client *openrouter.Client, category string) {
	opts := &openrouter.ListModelsOptions{}
	if category != "" {
		opts.Category = category
	}

	resp, err := client.ListModels(ctx, opts)
	if err != nil {
		log.Fatalf("Failed to list models: %v", err)
	}

	fmt.Printf("Available Models (%d):\n\n", len(resp.Data))
	for _, model := range resp.Data {
		fmt.Printf("%-40s Context: %6d, Prompt: $%s/M, Completion: $%s/M\n",
			model.ID,
			model.ContextLength,
			model.Pricing.Prompt,
			model.Pricing.Completion,
		)
	}
}

func runSingle(ctx context.Context, client *openrouter.Client, prompt, model, system string, temp float64, maxTokens int, stream, webSearch bool) {
	messages := []models.Message{}
	
	if system != "" {
		messages = append(messages, models.NewTextMessage(models.RoleSystem, system))
	}
	messages = append(messages, models.NewTextMessage(models.RoleUser, prompt))

	req := models.ChatCompletionRequest{
		Model:       model,
		Messages:    messages,
		Temperature: &temp,
	}

	if maxTokens > 0 {
		req.MaxTokens = &maxTokens
	}

	if webSearch {
		req.Model = model + ":online"
	}

	if stream {
		runStreaming(ctx, client, req)
	} else {
		resp, err := client.CreateChatCompletion(ctx, req)
		if err != nil {
			log.Fatalf("Request failed: %v", err)
		}

		content, _ := resp.Choices[0].Message.GetTextContent()
		fmt.Println(content)

		// Show citations if web search was used
		if webSearch {
			citations := openrouter.ExtractCitations(resp)
			if len(citations) > 0 {
				fmt.Println("\nSources:")
				for _, c := range citations {
					fmt.Printf("- %s: %s\n", c.Title, c.URL)
				}
			}
		}
	}
}

func runStreaming(ctx context.Context, client *openrouter.Client, req models.ChatCompletionRequest) {
	stream, err := client.CreateChatCompletionStream(ctx, req)
	if err != nil {
		log.Fatalf("Failed to create stream: %v", err)
	}
	defer stream.Close()

	for {
		chunk, err := stream.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("Stream error: %v", err)
		}

		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta != nil {
			content, _ := chunk.Choices[0].Delta.GetTextContent()
			fmt.Print(content)
		}
	}
	fmt.Println()
}

func runInteractive(ctx context.Context, client *openrouter.Client, model, system string, temp float64, maxTokens int, stream, webSearch bool) {
	fmt.Printf("Interactive mode. Model: %s\n", model)
	fmt.Println("Type 'exit' or 'quit' to end the session.")
	fmt.Println("Type 'clear' to start a new conversation.")
	fmt.Println()

	messages := []models.Message{}
	if system != "" {
		messages = append(messages, models.NewTextMessage(models.RoleSystem, system))
	}

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		if input == "exit" || input == "quit" {
			fmt.Println("Goodbye!")
			break
		}

		if input == "clear" {
			messages = []models.Message{}
			if system != "" {
				messages = append(messages, models.NewTextMessage(models.RoleSystem, system))
			}
			fmt.Println("Conversation cleared.")
			continue
		}

		// Add user message
		messages = append(messages, models.NewTextMessage(models.RoleUser, input))

		// Create request
		req := models.ChatCompletionRequest{
			Model:       model,
			Messages:    messages,
			Temperature: &temp,
		}

		if maxTokens > 0 {
			req.MaxTokens = &maxTokens
		}

		if webSearch {
			req.Model = model + ":online"
		}

		fmt.Print("\n")

		// Get response
		if stream {
			streamResp(ctx, client, req, &messages)
		} else {
			resp, err := client.CreateChatCompletion(ctx, req)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				// Remove the last message on error
				messages = messages[:len(messages)-1]
				continue
			}

			content, _ := resp.Choices[0].Message.GetTextContent()
			fmt.Println(content)

			// Add assistant message
			messages = append(messages, *resp.Choices[0].Message)

			// Show citations if web search was used
			if webSearch {
				citations := openrouter.ExtractCitations(resp)
				if len(citations) > 0 {
					fmt.Println("\nSources:")
					for _, c := range citations {
						fmt.Printf("- %s\n", c.URL)
					}
				}
			}
		}

		fmt.Println()
	}
}

func streamResp(ctx context.Context, client *openrouter.Client, req models.ChatCompletionRequest, messages *[]models.Message) {
	stream, err := client.CreateChatCompletionStream(ctx, req)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		// Remove the last message on error
		*messages = (*messages)[:len(*messages)-1]
		return
	}
	defer stream.Close()

	var contentBuilder strings.Builder
	for {
		chunk, err := stream.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Printf("\nStream error: %v\n", err)
			return
		}

		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta != nil {
			content, _ := chunk.Choices[0].Delta.GetTextContent()
			fmt.Print(content)
			contentBuilder.WriteString(content)
		}
	}

	// Add assistant message
	assistantMsg := models.NewTextMessage(models.RoleAssistant, contentBuilder.String())
	*messages = append(*messages, assistantMsg)
}