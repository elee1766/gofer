package theme

import "github.com/charmbracelet/lipgloss"

// Theme represents a color theme
var CurrentTheme = struct {
	Primary lipgloss.Color
	Text    lipgloss.Color
	TextMuted lipgloss.Color
	Background lipgloss.Color
}{
	Primary:    lipgloss.Color("#00ff00"),
	Text:       lipgloss.Color("#ffffff"),
	TextMuted:  lipgloss.Color("#808080"),
	Background: lipgloss.Color("#000000"),
}

// SetTheme sets the current theme
func SetTheme(colors struct {
	Primary lipgloss.Color
	Text    lipgloss.Color
	TextMuted lipgloss.Color
	Background lipgloss.Color
}) {
	CurrentTheme = colors
}