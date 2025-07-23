package dialog

// Dialog represents a placeholder for TUI dialog components
type Dialog struct {
	// Placeholder for dialog functionality
}

// NewFilePicker creates a new file picker dialog
func NewFilePicker() *Dialog {
	return &Dialog{}
}

// Show shows the dialog
func (d *Dialog) Show() (string, error) {
	return "", nil
}

// Close closes the dialog
func (d *Dialog) Close() error {
	return nil
}