package main

import (
	"fmt"

	"github.com/alecthomas/kong"
)

// TUICmd is the TUI command using the tui package
type TUICmd struct {
	File           string  `short:"f" help:"Load prompt from file"`
	SystemPrompt   string  `short:"s" help:"System prompt"`
	AppendPrompt   string  `short:"a" help:"Append to system prompt (adds to default or -s prompt)"`
	InitialMessage string  `short:"i" help:"Initial message to send to agent on launch"`
	Resume         bool    `short:"r" help:"Resume last conversation"`
	SessionID      string  `help:"Resume specific session by ID"`
	Model          string  `short:"m" help:"Model to use for this chat"`
	Temperature    float64 `short:"t" help:"Override temperature for this chat"`
	Theme          string  `default:"default" help:"UI theme"`
}

// Run executes the TUI command
func (c *TUICmd) Run(ctx *kong.Context, cli *CLI) error {
	// TUI implementation will be added later
	return fmt.Errorf("TUI not yet implemented")
}
