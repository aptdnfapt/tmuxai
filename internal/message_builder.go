package internal

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/alvinunreal/tmuxai/system"
)

// SimplifiedMessageBuilder creates messages with fresh state
type SimplifiedMessageBuilder struct {
	Manager *Manager
}

// NewSimplifiedMessageBuilder initializes a new message builder
func NewSimplifiedMessageBuilder(manager *Manager) *SimplifiedMessageBuilder {
	return &SimplifiedMessageBuilder{Manager: manager}
}

// BuildMessage constructs the simplified message
func (b *SimplifiedMessageBuilder) BuildMessage(userInput string) string {
	var buf bytes.Buffer
	
	// Repo Map
	buf.WriteString(b.buildRepoMap())
	buf.WriteString("====\n")
	
	// Files
	buf.WriteString(b.buildFiles())
	buf.WriteString("====\n")
	
	// Pane Content
	buf.WriteString(b.buildPaneContent())
	buf.WriteString("====\n")
	
	// Old Session Summary (AI-generated summary of previous sessions)
	buf.WriteString(b.buildOldSessionSummary())
	buf.WriteString("====\n")
	
	// Conversation History
	buf.WriteString(b.buildConversationHistory())
	buf.WriteString("====\n")
	
	// New User Message
	buf.WriteString(fmt.Sprintf("User: %s\n", userInput))
	
	return buf.String()
}

func (b *SimplifiedMessageBuilder) buildSystemPrompt() string {
	// Return updated system prompt with workflow instructions
	var prompt string
	switch {
	case b.Manager.WatchMode:
		prompt = b.Manager.watchPrompt().Content
	case b.Manager.GetAgenticMode():
		prompt = b.Manager.agenticPrompt().Content
	default:
		prompt = b.Manager.chatAssistantPrompt(b.Manager.ExecPane.IsPrepared).Content
	}
	return prompt
}

func (b *SimplifiedMessageBuilder) buildRepoMap() string {
	if b.Manager.RepoMap != nil {
		repoMap, _ := b.Manager.RepoMap.GetMap()
		return fmt.Sprintf("[Repo Map]\n%s", repoMap)
	}
	return "[Repo Map]\n(No repository map available)"
}

func (b *SimplifiedMessageBuilder) buildFiles() string {
	var buf bytes.Buffer
	buf.WriteString("[Files]\n")
	
	for _, filePath := range b.Manager.ReadFiles {
		content, err := os.ReadFile(filePath)
		if err == nil {
			buf.WriteString(fmt.Sprintf("File: %s\n```%s\n%s\n```\n\n", filePath, getFileExtension(filePath), string(content)))
		}
	}
	
	return buf.String()
}

func (b *SimplifiedMessageBuilder) buildPaneContent() string {
	var buf bytes.Buffer
	buf.WriteString("[Pane Content]\n")
	
	panes, _ := b.Manager.GetTmuxPanes()
	for _, pane := range panes {
		if pane.IsTmuxAiPane {
			continue
		}
		
		// Refresh pane to get updated information like LastLine and Shell
		pane.Refresh(b.Manager.GetMaxCaptureLines())
		
		paneContent, _ := system.TmuxCapturePane(pane.Id, b.Manager.GetMaxCaptureLines())
		
		// Build a comprehensive description of the pane
		paneInfo := fmt.Sprintf("Pane %s", pane.Id)
		
		// Add current command information
		if pane.CurrentCommand != "" {
			paneInfo += fmt.Sprintf(" (Command: %s", pane.CurrentCommand)
			if pane.CurrentCommandArgs != "" {
				paneInfo += fmt.Sprintf(" %s", pane.CurrentCommandArgs)
			}
			paneInfo += ")"
		}
		
		// Add process ID
		if pane.CurrentPid != 0 {
			paneInfo += fmt.Sprintf(" [PID: %d]", pane.CurrentPid)
		}
		
		// Add shell information
		if pane.Shell != "" {
			paneInfo += fmt.Sprintf(" [Shell: %s]", pane.Shell)
		}
		
		// Add OS information
		if pane.OS != "" {
			paneInfo += fmt.Sprintf(" [OS: %s]", pane.OS)
		}
		
		// Add prepared status
		if pane.IsPrepared {
			paneInfo += " [Prepared]"
		}
		
		// Add subshell status
		if pane.IsSubShell {
			paneInfo += " [SubShell]"
		}
		
		buf.WriteString(fmt.Sprintf("%s:\n%s\n\n", paneInfo, paneContent))
	}
	
	return buf.String()
}

func (b *SimplifiedMessageBuilder) buildOldSessionSummary() string {
	var buf bytes.Buffer
	buf.WriteString("[Previous Session Summary]\n")
	
	// When a session is opened for the second time, generate an AI summary
	// of what was accomplished in the previous session instead of sending raw data
	if b.Manager.HasPreviousSession() {
		summary := b.Manager.GetPreviousSessionSummary()
		buf.WriteString(summary)
	}
	
	return buf.String()
}

func (b *SimplifiedMessageBuilder) buildConversationHistory() string {
	var buf bytes.Buffer
	buf.WriteString("[Conversation History]\n")
	
	for _, msg := range b.Manager.Messages {
		role := "AI"
		if msg.FromUser {
			role = "User"
		}
		buf.WriteString(fmt.Sprintf("%s: %s\n", role, msg.Content))
	}
	
	return buf.String()
}

// Helper function to get file extension for syntax highlighting
func getFileExtension(filePath string) string {
	parts := strings.Split(filePath, ".")
	if len(parts) > 1 {
		return parts[len(parts)-1]
	}
	return ""
}