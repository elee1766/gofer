package main

import (
	"context"
	"os"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/elee1766/gofer/src/app"
	"github.com/elee1766/gofer/src/goferagent/toolsutil"
)

// PromptCmd represents the single prompt command
type PromptCmd struct {
	Text         []string `arg:"" help:"The prompt text to send"`
	SystemPrompt string   `short:"s" help:"System prompt"`
	File         string   `short:"f" help:"Load prompt from file"`
	Output       string   `short:"o" help:"Output format (text, json, markdown)" default:"text"`
	Raw          bool     `help:"Output raw response without formatting"`
	Model        string   `short:"m" help:"Model to use for this prompt" default:"google/gemini-2.5-flash"`
	Temperature  float64  `help:"Override temperature for this prompt"`
	MaxTokens    int      `help:"Override max tokens for this prompt"`
	MaxTurns     int      `help:"Maximum conversation turns" default:"3"`
	Resume       bool     `short:"r" help:"Resume last conversation"`
	SessionID    string   `help:"Resume specific session by ID"`
}

func (p *PromptCmd) Run(ctx *kong.Context, cli *CLI) error {
	// Create CLI logger for prompt mode
	logger := createCLILogger(cli.LogLevel)

	// Set the logger for tools
	toolsutil.SetLogger(logger)

	// Create app instance with shared state
	projectDir, _ := os.Getwd()
	appInstance, err := app.InitializeAgentAppWithTools(context.Background(), app.AppConfig{
		APIKey:       cli.APIKey,
		BaseURL:      cli.BaseURL,
		Model:        p.Model,
		SystemPrompt: p.SystemPrompt,
		EnableTools:  !cli.NoTools,
		Logger:       logger,
		ProjectDir:   projectDir,
	})
	if err != nil {
		return err
	}
	cctx := context.Background()

	return RunPrompt(cctx, appInstance, RunPromptParams{
		Text:         strings.Join(p.Text, " "),
		SystemPrompt: p.SystemPrompt,
		Output:       p.Output,
		Raw:          p.Raw,
		EnableTools:  !cli.NoTools,
		Logger:       logger,
		Resume:       p.Resume,
		SessionID:    p.SessionID,
		MaxTurns:     p.MaxTurns,
		Model:        p.Model,
		APIKey:       cli.APIKey,
	})
}
