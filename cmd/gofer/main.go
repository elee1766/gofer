package main

import (
	"fmt"
	"os"

	"github.com/alecthomas/kong"
)

// CLI represents the main CLI structure
type CLI struct {
	APIKey   string `env:"OPENROUTER_API_KEY" help:"OpenRouter API key"`
	NoTools  bool   `help:"Disable tool usage"`
	BaseURL  string `help:"Custom API base URL"`
	LogLevel string `default:"warn" help:"Log level"`

	// TUI is the default command - interactive chat interface
	// TUI TUICmd `default:"1" cmd:"" help:"Start interactive TUI (default)"`

	// Other commands
	Prompt  PromptCmd  `cmd:"" help:"Execute a single prompt"`
	Migrate MigrateCmd `cmd:"" help:"Database migrations"`
	Model   ModelCmd   `cmd:"" help:"Model management and information"`
}

func main() {
	var cli CLI
	ctx := kong.Parse(&cli,
		kong.Name("gofer"),
		kong.Description("AI assistant powered by OpenRouter"),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
		}),
	)

	err := ctx.Run(&cli)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
