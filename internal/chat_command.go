package internal

import (
	"fmt"
	"os"
	"strings"

	"github.com/alvinunreal/tmuxai/config"
	"github.com/alvinunreal/tmuxai/logger"
	"github.com/alvinunreal/tmuxai/system"
)

const helpMessage = `Available commands:
- /info: Display system information
- /clear: Clear the chat history
- /reset: Reset the chat history
- /edit: Open an external editor to compose your prompt
- /prepare [pane_id]: Toggles a pane into 'prepared' mode, where all AI commands will wait for completion. Not available in agentic mode.
- /unprepare [pane_id]: Toggles a pane out of 'prepared' mode.
- /watch <prompt>: Start watch mode
- /squash: Summarize the chat history
- /session: List and manage sessions
- /exit: Exit the application`

var commands = []string{
	"/help",
	"/clear",
	"/reset",
	"/exit",
	"/info",
	"/watch",
	"/prepare",
	"/unprepare",
	"/config",
	"/squash",
	"/session",
	"/edit",
}

// checks if the given content is a command
func (m *Manager) IsMessageSubcommand(content string) bool {
	content = strings.TrimSpace(strings.ToLower(content)) // Normalize input

	// Any message starting with / is considered a command
	return strings.HasPrefix(content, "/")
}

// processes a command and returns a message to process, and whether to process it
func (m *Manager) ProcessSubCommand(command string) (string, bool) {
	commandLower := strings.ToLower(strings.TrimSpace(command))
	logger.Info("Processing command: %s", command)

	// Get the first word from the command (e.g., "/watch" from "/watch something")
	parts := strings.Fields(commandLower)
	if len(parts) == 0 {
		m.Println("Empty command")
		return "", false
	}

	commandPrefix := parts[0]

	// Process the command using prefix matching
	switch {
	case prefixMatch(commandPrefix, "/help"):
		m.Println(helpMessage)
		return "", false

	case prefixMatch(commandPrefix, "/edit"):
		content, err := OpenInEditor(m.GetEditor(), "")
		if err != nil {
			m.Println(fmt.Sprintf("Error opening editor: %v", err))
			return "", false
		}
		if content == "" {
			m.Println("Cancelled.")
			return "", false
		}
		return content, true

	case prefixMatch(commandPrefix, "/info"):
		m.formatInfo()
		return "", false

	case prefixMatch(commandPrefix, "/prepare"):
		if m.GetAgenticMode() {
			m.Println("Error: /prepare is not available in agentic mode. The AI manages command execution automatically.")
			return "", false
		}
		parts := strings.Fields(command)
		var targetPane *system.TmuxPaneDetails

		if len(parts) > 1 {
			paneID := parts[1]
			if !strings.HasPrefix(paneID, "%") {
				paneID = "%" + paneID
			}
			panes, _ := m.GetTmuxPanes()
			found := false
			for i, p := range panes {
				if p.Id == paneID {
					targetPane = &panes[i]
					found = true
					break
				}
			}
			if !found {
				m.Println(fmt.Sprintf("Error: Pane with ID %s not found.", paneID))
				return "", false
			}
		} else {
			if m.ExecPane == nil || m.ExecPane.Id == "" {
				m.InitExecPane()
			}
			targetPane = m.ExecPane
		}

		if targetPane == nil || targetPane.Id == "" {
			m.Println("Error: Could not determine a target pane to prepare.")
			return "", false
		}

		m.PreparePane(targetPane)
		return "", false

	case prefixMatch(commandPrefix, "/unprepare"):
		parts := strings.Fields(command)
		var targetPane *system.TmuxPaneDetails

		if len(parts) > 1 {
			paneID := parts[1]
			if !strings.HasPrefix(paneID, "%") {
				paneID = "%" + paneID
			}
			panes, _ := m.GetTmuxPanes()
			found := false
			for i, p := range panes {
				if p.Id == paneID {
					targetPane = &panes[i]
					found = true
					break
				}
			}
			if !found {
				m.Println(fmt.Sprintf("Error: Pane with ID %s not found.", paneID))
				return "", false
			}
		} else {
			if m.ExecPane == nil || m.ExecPane.Id == "" {
				m.InitExecPane()
			}
			targetPane = m.ExecPane
		}
		if targetPane == nil || targetPane.Id == "" {
			m.Println("Error: Could not determine a target pane to unprepare.")
			return "", false
		}
		delete(m.PreparedPanes, targetPane.Id)
		targetPane.IsPrepared = false
		m.Println(fmt.Sprintf("Pane %s is no longer in prepared mode.", targetPane.Id))
		return "", false

	case prefixMatch(commandPrefix, "/clear"):
		m.Messages = []ChatMessage{}
		system.TmuxClearPane(m.PaneId)
		return "", false

	case prefixMatch(commandPrefix, "/reset"):
		m.Status = ""
		m.WatchMode = false // Ensure watch mode is off on reset
		m.Messages = []ChatMessage{}
		system.TmuxClearPane(m.PaneId)
		// Clear all other panes as well
		panes, _ := m.GetTmuxPanes()
		for _, pane := range panes {
			if pane.Id != m.PaneId {
				system.TmuxClearPane(pane.Id)
			}
		}
		// Re-initialize the exec pane after clearing
		m.InitExecPane()
		return "", false

	case prefixMatch(commandPrefix, "/exit"):
		logger.Info("Exit command received, stopping watch mode (if active) and exiting.")
		m.SaveSession()
		os.Exit(0)
		return "", false

	case prefixMatch(commandPrefix, "/squash"):
		m.squashHistory()
		return "", false

	case prefixMatch(commandPrefix, "/session"):
		sessions, err := ListSessions()
		if err != nil {
			m.Println(fmt.Sprintf("Error listing sessions: %v", err))
			return "", false
		}
		if len(sessions) == 0 {
			m.Println("No saved sessions found in this directory.")
			return "", false
		}

		selectedPath, err := ShowSessionList(sessions)
		if err != nil {
			// This can happen if the UI fails to start, but not on user quit (q)
			m.Println(fmt.Sprintf("Could not show session list: %v", err))
			return "", false
		}

		if selectedPath != "" {
			// Save the current session before switching.
			if err := m.SaveSession(); err != nil {
				m.Println(fmt.Sprintf("Warning: could not save current session: %v", err))
				logger.Error("Failed to save session before switching: %v", err)
			}

			// Clear current state before loading
			m.Messages = []ChatMessage{}
			m.ExecHistory = []CommandExecHistory{}
			m.ReadFiles = []string{}
			m.SessionPath = ""
			m.Status = "running" // Ensure status is active

			if err := m.LoadSession(selectedPath); err != nil {
				m.Println(fmt.Sprintf("Error loading session: %v", err))
			} else {
				// Successfully loaded, clear the screen and show a prompt
				system.TmuxClearPane(m.PaneId)
				m.Println("Session restored. You can continue the conversation.")
			}
		}
		// If no path is selected (user quit), we just return to the prompt
		return "", false

	case prefixMatch(commandPrefix, "/watch") || commandPrefix == "/w":
		parts := strings.Fields(command)
		if len(parts) > 1 {
			watchDesc := strings.Join(parts[1:], " ")
			startWatch := `
1. Find out if there is new content in the pane based on chat history.
2. Comment only considering the new content in this pane output.

Watch for: ` + watchDesc
			m.Status = "running"
			m.WatchMode = true
			m.startWatchMode(startWatch)
			return "", false
		}
		m.Println("Usage: /watch <description>")
		return "", false

	case prefixMatch(commandPrefix, "/config"):
		// Helper function to check if a key is allowed
		isKeyAllowed := func(key string) bool {
			for _, k := range AllowedConfigKeys {
				if k == key {
					return true
				}
			}
			return false
		}

		// Check if it's "config set" for a specific key
		if len(parts) >= 3 && parts[1] == "set" {
			key := parts[2]
			if !isKeyAllowed(key) {
				m.Println(fmt.Sprintf("Cannot set '%s'. Only these keys are allowed: %s", key, strings.Join(AllowedConfigKeys, ", ")))
				return "", false
			}
			value := strings.Join(parts[3:], " ")
			m.SessionOverrides[key] = config.TryInferType(key, value)
			m.Println(fmt.Sprintf("Set %s = %v", key, m.SessionOverrides[key]))
			return "", false
		} else {
			code, _ := system.HighlightCode("yaml", m.FormatConfig())
			fmt.Println(code)
			return "", false
		}

	default:
		m.Println(fmt.Sprintf("Unknown command: %s. Type '/help' to see available commands.", command))
		return "", false
	}
}

// Helper function to check if a command matches a prefix
func prefixMatch(command, target string) bool {
	return strings.HasPrefix(target, command)
}

// formats system information and tmux details into a readable string
func (m *Manager) formatInfo() {
	formatter := system.NewInfoFormatter()
	const labelWidth = 18 // Width of the label column
	formatLine := func(key string, value any) {
		fmt.Print(formatter.LabelColor.Sprintf("%-*s", labelWidth, key))
		fmt.Print("  ")
		fmt.Println(value)
	}
	// Display general information
	fmt.Println(formatter.FormatSection("\nGeneral"))
	formatLine("Version", Version)
	formatLine("Max Capture Lines", m.Config.MaxCaptureLines)
	formatLine("Wait Interval", m.Config.WaitInterval)

	// Display context information section
	fmt.Println(formatter.FormatSection("\nContext"))
	formatLine("Messages", len(m.Messages))
	var totalTokens int
	for _, msg := range m.Messages {
		totalTokens += system.EstimateTokenCount(msg.Content)
	}

	usagePercent := 0.0
	if m.GetMaxContextSize() > 0 {
		usagePercent = float64(totalTokens) / float64(m.GetMaxContextSize()) * 100
	}
	fmt.Print(formatter.LabelColor.Sprintf("%-*s", labelWidth, "Context Size~"))
	fmt.Print("  ") // Two spaces for separation
	fmt.Printf("%s\n", fmt.Sprintf("%d tokens", totalTokens))
	fmt.Printf("%-*s  %s\n", labelWidth, "", formatter.FormatProgressBar(usagePercent, 10))
	formatLine("Max Size", fmt.Sprintf("%d tokens", m.GetMaxContextSize()))

	// Display tmux panes section
	fmt.Println()
	fmt.Println(formatter.FormatSection("Tmux Window Panes"))

	panes, _ := m.GetTmuxPanes()
	isAgentic := m.GetAgenticMode()
	for _, pane := range panes {
		pane.Refresh(m.GetMaxCaptureLines())
		fmt.Println(pane.FormatInfo(formatter, isAgentic))
	}
}
