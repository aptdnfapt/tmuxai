package internal

import (
	"fmt"
	"os"
	"os/exec"
)

// OpenInEditor launches the user's configured editor with initial content and returns the new content.
func OpenInEditor(editorCmd string, initialContent string) (string, error) {
	// Create a temporary file with a .md extension so editors might apply markdown highlighting.
	tmpFile, err := os.CreateTemp("", "tmuxai-*.md")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if initialContent != "" {
		if _, err := tmpFile.WriteString(initialContent); err != nil {
			tmpFile.Close() // Close before returning error
			return "", fmt.Errorf("failed to write initial content: %w", err)
		}
	}

	// Important: Close the file before handing it over to the editor.
	if err := tmpFile.Close(); err != nil {
		return "", fmt.Errorf("failed to close temp file before editing: %w", err)
	}

	// We assume editorCmd is a single command like "nano" or "nvim".
	// For more complex commands (e.g., with args), we'd need to parse it.
	// For now, this is sufficient.
	cmd := exec.Command(editorCmd, tmpFile.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run the editor command. This will block until the user closes the editor.
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("editor command failed: %w", err)
	}

	// Read the content back from the temporary file.
	content, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		return "", fmt.Errorf("failed to read temp file after editing: %w", err)
	}

	return string(content), nil
}
