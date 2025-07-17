package internal

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ShowPromptEditor launches a bubbletea text area for multiline input.
func ShowPromptEditor() (string, error) {
	p := tea.NewProgram(initialModel())

	m, err := p.Run()
	if err != nil {
		return "", err
	}

	finalModel, ok := m.(model)
	if !ok {
		return "", fmt.Errorf("could not cast model")
	}

	if finalModel.cancelled {
		return "", nil
	}
	return finalModel.textarea.Value(), nil
}

type model struct {
	textarea  textarea.Model
	cancelled bool
	quitting  bool
}

func initialModel() model {
	ti := textarea.New()
	ti.Placeholder = "Enter your prompt here..."
	ti.Focus()
	ti.CharLimit = -1 // No limit
	ti.SetWidth(80)
	ti.SetHeight(5)
	ti.ShowLineNumbers = false
	ti.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ti.BlurredStyle.CursorLine = lipgloss.NewStyle()

	greenStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#C8F7C5"))
	ti.FocusedStyle.Prompt = greenStyle
	ti.BlurredStyle.Prompt = greenStyle
	ti.Cursor.Style = greenStyle

	return model{
		textarea:  ti,
		cancelled: false,
		quitting:  false,
	}
}

func (m model) Init() tea.Cmd {
	return textarea.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlD, tea.KeyEnter: // Submit
			m.quitting = true
			return m, tea.Quit

		case tea.KeyCtrlJ:
			m.textarea.InsertString("\n")
			return m, nil

		case tea.KeyEsc, tea.KeyCtrlC:
			m.cancelled = true
			m.quitting = true
			return m, tea.Quit

		default:
			m.textarea, cmd = m.textarea.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	if m.quitting {
		return ""
	}
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	helpView := helpStyle.Render("(Ctrl+D/Enter to submit, Ctrl+J for newline, Esc/Ctrl+C to cancel)")

	return fmt.Sprintf("Enter prompt:\n%s\n\n%s",
		m.textarea.View(),
		helpView,
	) + "\n"
}
