package main

import (
	"fmt"
	"log"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// model holds the application's state
type model struct {
	textarea textarea.Model
	quitting bool
	saved    bool
}

// initialModel sets up the initial state
func initialModel() model {
	ta := textarea.New()
	ta.Placeholder = "Type your long-form text here..."
	ta.Focus()
	ta.SetWidth(80)
	ta.SetHeight(5)
	ta.ShowLineNumbers = false
	
	// Style the textarea with very light green cursor and prompt
	greenStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#C8F7C5"))
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.BlurredStyle.CursorLine = lipgloss.NewStyle()
	ta.FocusedStyle.Prompt = greenStyle
	ta.BlurredStyle.Prompt = greenStyle
	ta.Cursor.Style = greenStyle

	return model{
		textarea: ta,
		quitting: false,
		saved:    false,
	}
}

// Init returns the initial command for the program
func (m model) Init() tea.Cmd {
	return textarea.Blink
}

// Update handles messages and updates the model
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			// Save and exit
			m.saved = true
			m.quitting = true
			return m, tea.Quit
		case tea.KeyCtrlJ:
			// Insert a new line
			m.textarea.InsertString("\n")
			return m, nil
		case tea.KeyCtrlD:
			// Save and exit (alternative)
			m.saved = true
			m.quitting = true
			return m, tea.Quit
		case tea.KeyEsc, tea.KeyCtrlC:
			// Quit without saving
			m.saved = false
			m.quitting = true
			return m, tea.Quit
		}
	}

	// Handle character input and blinking
	m.textarea, cmd = m.textarea.Update(msg)
	return m, cmd
}

// View renders the UI
func (m model) View() string {
	if m.quitting {
		return ""
	}
	return fmt.Sprintf(
		"Multi-line text editor:\n\n%s\n\n%s",
		m.textarea.View(),
		"(Enter to submit, Ctrl+J for new line, Esc or Ctrl+C to quit without saving)",
	) + "\n"
}

// main function to run the program
func main() {
	p := tea.NewProgram(initialModel())
	
	finalModel, err := p.Run()
	if err != nil {
		log.Fatal(err)
	}

	// Output the final text value with appropriate message
	if m, ok := finalModel.(model); ok {
		if m.saved {
			fmt.Println("Text saved:")
			fmt.Println(m.textarea.Value())
		} else {
			fmt.Println("Exited without saving.")
		}
	}
}