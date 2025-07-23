package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/alecthomas/kong"
	"github.com/elee1766/gofer/src/aisdk"
	"github.com/elee1766/gofer/src/orclient"
)

// ModelCmd manages model operations
type ModelCmd struct {
	List   ModelListCmd   `cmd:"" help:"List available models"`
	Info   ModelInfoCmd   `cmd:"" help:"Get information about a specific model"`
	Test   ModelTestCmd   `cmd:"" help:"Test a model with a simple prompt"`
	Search ModelSearchCmd `cmd:"" help:"Search for models by name"`
}

// ModelListCmd lists available models
type ModelListCmd struct {
	Format    string `help:"Output format (table, json)" default:"table"`
	WithCosts bool   `help:"Include pricing information"`
}

// Run executes the model list command
func (c *ModelListCmd) Run(ctx *kong.Context, cli *CLI) error {
	client := orclient.NewClient(orclient.Config{
		APIKey: cli.APIKey,
	})

	models, err := client.ListModels(context.Background())
	if err != nil {
		return fmt.Errorf("failed to list models: %w", err)
	}

	switch c.Format {
	case "json":
		return printModelsJSON(models)
	case "table":
		return printModelsTable(models, c.WithCosts)
	default:
		return fmt.Errorf("invalid format: %s", c.Format)
	}
}

// ModelInfoCmd gets information about a specific model
type ModelInfoCmd struct {
	Model  string `arg:"" help:"Model ID or name"`
	Format string `help:"Output format (table, json)" default:"table"`
}

// Run executes the model info command
func (c *ModelInfoCmd) Run(ctx *kong.Context, cli *CLI) error {
	client := orclient.NewClient(orclient.Config{
		APIKey: cli.APIKey,
	})

	model, err := client.GetModelByID(context.Background(), c.Model)
	if err != nil {
		return fmt.Errorf("failed to get model info: %w", err)
	}

	switch c.Format {
	case "json":
		return printModelJSON(model)
	case "table":
		return printModelTable(model)
	default:
		return fmt.Errorf("invalid format: %s", c.Format)
	}
}

// ModelTestCmd tests a model with a simple prompt
type ModelTestCmd struct {
	Model  string `arg:"" help:"Model ID or name"`
	Prompt string `help:"Test prompt" default:"what is 9 + 10?"`
}

// Run executes the model test command
func (c *ModelTestCmd) Run(ctx *kong.Context, cli *CLI) error {
	client := orclient.NewClient(orclient.Config{
		APIKey: cli.APIKey,
	})

	modelClient, err := client.Model(context.Background(), c.Model)
	if err != nil {
		return fmt.Errorf("failed to create model client: %w", err)
	}

	fmt.Printf("Testing model: %s\n", modelClient.GetModelInfo().Name)
	fmt.Printf("Prompt: %s\n\n", c.Prompt)

	maxTokens := 100
	temperature := 0.7
	req := &aisdk.ChatCompletionRequest{
		Messages: []*aisdk.Message{
			{
				Role:    "user",
				Content: c.Prompt,
			},
		},
		MaxTokens:   &maxTokens,
		Temperature: &temperature,
	}

	resp, err := modelClient.CreateChatCompletion(context.Background(), req)
	if err != nil {
		return fmt.Errorf("failed to create chat completion: %w", err)
	}

	if len(resp.Choices) > 0 {
		fmt.Printf("Response: %s\n\n", resp.Choices[0].Message.Content)
	}

	if resp.Usage.TotalTokens > 0 {
		fmt.Printf("Usage:\n")
		fmt.Printf("  Prompt tokens: %d\n", resp.Usage.PromptTokens)
		fmt.Printf("  Completion tokens: %d\n", resp.Usage.CompletionTokens)
		fmt.Printf("  Total tokens: %d\n", resp.Usage.TotalTokens)
	}

	return nil
}

// formatCurrency formats a cost value as currency
func formatCurrency(amount float64, currency string) string {
	switch currency {
	case "USD":
		if amount < 0.01 {
			return "$0.00" // Round very small amounts to zero
		}
		return fmt.Sprintf("$%.2f", amount)
	default:
		return fmt.Sprintf("%.4f %s", amount, currency)
	}
}

// ModelSearchCmd searches for models by name
type ModelSearchCmd struct {
	Query  string `arg:"" help:"Search query"`
	Format string `help:"Output format (table, json)" default:"table"`
}

// Run executes the model search command
func (c *ModelSearchCmd) Run(ctx *kong.Context, cli *CLI) error {
	client := orclient.NewClient(orclient.Config{
		APIKey: cli.APIKey,
	})

	models, err := client.ListModels(context.Background())
	if err != nil {
		return fmt.Errorf("failed to list models: %w", err)
	}

	// Filter models based on query
	var matches []*aisdk.ModelInfo
	query := strings.ToLower(c.Query)
	for _, model := range models {
		if strings.Contains(strings.ToLower(model.ID), query) ||
			strings.Contains(strings.ToLower(model.Name), query) {
			matches = append(matches, model)
		}
	}

	if len(matches) == 0 {
		fmt.Printf("No models found matching '%s'\n", c.Query)
		return nil
	}

	fmt.Printf("Found %d models matching '%s':\n\n", len(matches), c.Query)

	switch c.Format {
	case "json":
		return printModelsJSON(matches)
	case "table":
		return printModelsTable(matches, false)
	default:
		return fmt.Errorf("invalid format: %s", c.Format)
	}
}

// Helper functions for printing

func printModelsJSON(models []*aisdk.ModelInfo) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(models)
}

func printModelsTable(models []*aisdk.ModelInfo, withCosts bool) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	if withCosts {
		fmt.Fprintln(w, "ID\tName\tContext\tPrompt Cost\tCompletion Cost")
		fmt.Fprintln(w, "---\t----\t-------\t-----------\t---------------")
		for _, model := range models {
			promptCost := "N/A"
			completionCost := "N/A"
			if model.Pricing != nil {
				if model.Pricing.Prompt != "" {
					promptCost = model.Pricing.Prompt
				}
				if model.Pricing.Completion != "" {
					completionCost = model.Pricing.Completion
				}
			}
			fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%s\n",
				model.ID, model.Name, model.ContextLength, promptCost, completionCost)
		}
	} else {
		fmt.Fprintln(w, "ID\tName\tContext Length")
		fmt.Fprintln(w, "---\t----\t--------------")
		for _, model := range models {
			fmt.Fprintf(w, "%s\t%s\t%d\n", model.ID, model.Name, model.ContextLength)
		}
	}

	return nil
}

func printModelJSON(model *aisdk.ModelInfo) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(model)
}

func printModelTable(model *aisdk.ModelInfo) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	fmt.Fprintf(w, "ID:\t%s\n", model.ID)
	fmt.Fprintf(w, "Name:\t%s\n", model.Name)
	fmt.Fprintf(w, "Description:\t%s\n", model.Description)
	fmt.Fprintf(w, "Context Length:\t%d\n", model.ContextLength)

	if model.Pricing != nil {
		fmt.Fprintln(w, "\nPricing:")
		if model.Pricing.Prompt != "" {
			fmt.Fprintf(w, "  Prompt:\t%s per 1K tokens\n", model.Pricing.Prompt)
		}
		if model.Pricing.Completion != "" {
			fmt.Fprintf(w, "  Completion:\t%s per 1K tokens\n", model.Pricing.Completion)
		}
		if model.Pricing.Request != "" {
			fmt.Fprintf(w, "  Request:\t%s per request\n", model.Pricing.Request)
		}
		if model.Pricing.Image != "" {
			fmt.Fprintf(w, "  Image:\t%s per image\n", model.Pricing.Image)
		}
	}

	if model.Architecture != nil {
		fmt.Fprintln(w, "\nArchitecture:")
		if model.Architecture.Modality != "" {
			fmt.Fprintf(w, "  Modality:\t%s\n", model.Architecture.Modality)
		}
		if model.Architecture.Tokenizer != "" {
			fmt.Fprintf(w, "  Tokenizer:\t%s\n", model.Architecture.Tokenizer)
		}
		if model.Architecture.InstructType != nil && *model.Architecture.InstructType != "" {
			fmt.Fprintf(w, "  Instruct Type:\t%s\n", *model.Architecture.InstructType)
		}
	}

	if len(model.Modality) > 0 {
		fmt.Fprintf(w, "Modalities:\t%s\n", strings.Join(model.Modality, ", "))
	}

	return nil
}

