package storage

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// GenerateID generates a unique ID for storage entities
func GenerateID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes)
}

// Previous builder types removed; construct entities directly as needed.
// DefaultSettings, DefaultUserPreferences remain.

func DefaultSettings() map[string]string {
	return map[string]string{
		"theme":                 "system",
		"language":              "en",
		"model":                 "claude-3-opus-20240229",
		"max_tokens":            "4096",
		"temperature":           "0.7",
		"stream_responses":      "true",
		"save_conversations":    "true",
		"auto_save_interval":    "30",
		"session_timeout":       "24",
		"show_timestamps":       "true",
		"enable_syntax_highlight": "true",
		"enable_markdown":       "true",
		"enable_code_completion": "false",
		"max_history_items":     "100",
		"prompt_caching":        "true",
	}
}

func DefaultUserPreferences() map[string]string {
	return map[string]string{
		"preferred_model":        "claude-3-opus-20240229",
		"code_style":             "google",
		"editor_theme":           "monokai",
		"font_size":              "14",
		"line_numbers":           "true",
		"word_wrap":              "false",
		"auto_indent":            "true",
		"tab_size":               "4",
		"use_spaces":             "false",
		"show_invisible_chars":   "false",
		"highlight_active_line":  "true",
		"match_brackets":         "true",
		"auto_close_brackets":    "true",
		"enable_autocomplete":    "true",
		"autocomplete_delay":     "300",
	}
}
