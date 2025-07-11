package internal

import (
	"fmt"
	"io"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	appStyle                 = lipgloss.NewStyle().Padding(1, 2)
	titleStyle               = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFDF5")).Background(lipgloss.Color("#25A065")).Padding(0, 1)
	statusStyle              = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#343433", Dark: "#C8C8C2"}).Render
	itemStyle                = lipgloss.NewStyle().PaddingLeft(4)
	selectedStyle            = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170"))
	descriptionStyle         = lipgloss.NewStyle().PaddingLeft(4).Faint(true)
	selectedDescriptionStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170")).Faint(true)
)

// sessionItem implements the list.Item interface
type sessionItem struct {
	title       string
	description string
	filePath    string
}

func (i sessionItem) Title() string       { return i.title }
func (i sessionItem) Description() string { return i.description }
func (i sessionItem) FilterValue() string { return i.title }

type itemDelegate struct{}

func (d itemDelegate) Height() int                               { return 2 }
func (d itemDelegate) Spacing() int                              { return 1 }
func (d itemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	s, ok := listItem.(sessionItem)
	if !ok {
		return
	}

	var mainStr, descStr string
	if index == m.Index() {
		mainStr = selectedStyle.Render("> " + s.Title())
		descStr = selectedDescriptionStyle.Render("  " + s.Description())
	} else {
		mainStr = itemStyle.Render(s.Title())
		descStr = descriptionStyle.Render(s.Description())
	}

	fmt.Fprintf(w, "%s\n%s", mainStr, descStr)
}

// sessionListModel is the bubbletea model for the session list
type sessionListModel struct {
	list     list.Model
	choice   string
	quitting bool
}

func newSessionListModel(sessions []SessionInfo) sessionListModel {
	items := make([]list.Item, len(sessions))
	for i, s := range sessions {
		items[i] = sessionItem{
			title:       s.Title,
			description: fmt.Sprintf("Last updated: %s", s.Timestamp.Format(time.RFC822)),
			filePath:    s.FilePath,
		}
	}

	l := list.New(items, itemDelegate{}, 0, 0)
	l.Title = "Select a Session to Restore"
	l.Styles.Title = titleStyle
	l.SetShowStatusBar(true)
	l.SetStatusBarItemName("session", "sessions")
	l.StatusMessageLifetime = 0
	l.SetShowHelp(true)

	return sessionListModel{list: l}
}

func (m sessionListModel) Init() tea.Cmd {
	return nil
}

func (m sessionListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetSize(msg.Width, msg.Height)
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		case "enter":
			if i, ok := m.list.SelectedItem().(sessionItem); ok {
				m.choice = i.filePath
			}
			m.quitting = true
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m sessionListModel) View() string {
	if m.quitting {
		if m.choice != "" {
			return statusStyle(fmt.Sprintf("Restoring session: %s...", m.choice))
		}
		return statusStyle("No session selected.")
	}
	return appStyle.Render(m.list.View())
}

// ShowSessionList initializes and runs the bubbletea UI
func ShowSessionList(sessions []SessionInfo) (string, error) {
	if len(sessions) == 0 {
		return "", fmt.Errorf("no sessions found to display")
	}

	m := newSessionListModel(sessions)
	p := tea.NewProgram(m, tea.WithAltScreen())

	result, err := p.Run()
	if err != nil {
		return "", fmt.Errorf("error running session list UI: %w", err)
	}

	finalModel, ok := result.(sessionListModel)
	if !ok {
		return "", fmt.Errorf("could not cast result model")
	}

	return finalModel.choice, nil
}
