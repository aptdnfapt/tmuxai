package internal

import (
	"bytes"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// StructuredMessageBuilder creates the structured message for the AI.
type StructuredMessageBuilder struct {
	Manager *Manager
}

// NewStructuredMessageBuilder initializes a new message builder.
func NewStructuredMessageBuilder(manager *Manager) *StructuredMessageBuilder {
	return &StructuredMessageBuilder{Manager: manager}
}

// BuildMessage constructs the full message string.
func (b *StructuredMessageBuilder) BuildMessage(userInput string) string {
	var buf bytes.Buffer

	// Build sections in your desired order
	b.appendSection(&buf, "PROMPTS", b.buildPromptsSection)
	b.appendSection(&buf, "REPO-MAP", b.buildRepoMapSection)
	b.appendSection(&buf, "FILES", b.buildFilesSection)
	b.appendSection(&buf, "OLD-PANE-AND-CONVO", b.buildOldSessionSection)
	b.appendSection(&buf, "CURRENT-SESSION-PANES-AND-CONVO", func() string {
		return b.buildCurrentSessionSection(userInput)
	})

	return buf.String()
}

// appendSection is a helper to build and append a section to the buffer.
func (b *StructuredMessageBuilder) appendSection(buf *bytes.Buffer, title string, builderFunc func() string) {
	content := builderFunc()
	if content != "" {
		buf.WriteString(fmt.Sprintf("---%s----\n", title))
		buf.WriteString(content)
		buf.WriteString(fmt.Sprintf("----END-OF-%s----\n\n", title))
	}
}

// buildPromptsSection will contain system prompts.
func (b *StructuredMessageBuilder) buildPromptsSection() string {
	// Get the appropriate prompt based on current mode
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

// buildRepoMapSection builds the repo map section with status.
func (b *StructuredMessageBuilder) buildRepoMapSection() string {
	repoMapState := b.Manager.ContextTracker.CurrentState.RepoMap
	switch repoMapState.Status {
	case StatusNew:
		return fmt.Sprintf("[NEW]\n%s", repoMapState.Content)
	case StatusUpdated:
		return fmt.Sprintf("[UPDATED REPOMAP]\n%s", repoMapState.Content)
	case StatusUnchanged:
		return fmt.Sprintf("[UNCHANGED since message %d]", repoMapState.LastChanged)
	default:
		return ""
	}
}

// buildFilesSection builds the files section with sub-blocks for each file.
func (b *StructuredMessageBuilder) buildFilesSection() string {
	var content bytes.Buffer
	var paths []string
	for path := range b.Manager.ContextTracker.CurrentState.Files {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	for _, path := range paths {
		state := b.Manager.ContextTracker.CurrentState.Files[path]
		switch state.Status {
		case StatusNew:
			content.WriteString(fmt.Sprintf("file: %s [NEW] (last modified: %s)\n", path, state.Timestamp.Format("15:04:05")))
			content.WriteString(state.Content + "\n\n")
		case StatusUpdated:
			content.WriteString(fmt.Sprintf("file: %s [UPDATED] (last modified: %s)\n", path, state.Timestamp.Format("15:04:05")))
			content.WriteString(state.Content + "\n\n")
		case StatusUnchanged:
			content.WriteString(fmt.Sprintf("file: %s [UNCHANGED since message %d] (last modified: %s)\n\n", path, state.LastChanged, state.Timestamp.Format("15:04:05")))
		case StatusRemoved:
			content.WriteString(fmt.Sprintf("file: %s [REMOVED] (removed at: %s)\n\n", path, state.RemovedAt.Format("15:04:05")))
		}
	}
	return strings.TrimSuffix(content.String(), "\n\n")
}

// buildOldSessionSection builds the section for restored session data.
func (b *StructuredMessageBuilder) buildOldSessionSection() string {
	if b.Manager.OldSession == nil {
		return ""
	}

	var content bytes.Buffer
	oldSession := b.Manager.OldSession

	// Format old session data with date/time grouping
	content.WriteString(fmt.Sprintf("### date time %s\n", oldSession.SavedAt.Format("15:04:05")))

	// Show old panes
	for id, state := range oldSession.Panes {
		content.WriteString(fmt.Sprintf("====pane: %s (tmuxai_exec_pane) [UPDATED] (last updated: %s)\n", id, state.Timestamp.Format("15:04:05")))
		content.WriteString(state.Content + "\n")
		content.WriteString(fmt.Sprintf("====end of pane %s====\n", id))
		content.WriteString("--\n")
	}

	// Show old chat history
	content.WriteString("last chat:\n")
	reUserInput := regexp.MustCompile(`(?s)user : (.*)\nai :`)
	for _, msg := range oldSession.Conversation {
		var role, messageText string
		if msg.FromUser {
			role = "user"
			// Extract user input from structured content
			matches := reUserInput.FindStringSubmatch(msg.Content)
			if len(matches) > 1 {
				messageText = strings.TrimSpace(matches[1])
			}
		} else {
			role = "ai"
			// Parse AI response to get user-facing message
			parsedResponse, err := b.Manager.parseAIResponse(msg.Content)
			if err == nil && parsedResponse.Message != "" {
				messageText = parsedResponse.Message
			}
		}

		if strings.TrimSpace(messageText) != "" {
			content.WriteString(fmt.Sprintf("%s\n", role))
			content.WriteString(fmt.Sprintf("%s\n", messageText))
		}
	}

	content.WriteString(fmt.Sprintf("### end of date time %s\n", oldSession.SavedAt.Format("15:04:05")))
	return content.String()
}

// buildCurrentSessionSection constructs the block for the live, current session.
func (b *StructuredMessageBuilder) buildCurrentSessionSection(userInput string) string {
	var content bytes.Buffer

	
	// Add Panes with proper formatting
	for id, state := range b.Manager.ContextTracker.CurrentState.Panes {
		switch state.Status {
		case StatusNew:
			content.WriteString(fmt.Sprintf("====pane: %s (tmuxai_exec_pane) [UPDATED] (last updated: %s)\n", id, state.Timestamp.Format("15:04:05")))
			content.WriteString("___NEW-CONTENT___\n")
			content.WriteString(state.Content + "\n")
			content.WriteString("____END-OF-NEW-CONTENT____\n")
			content.WriteString(fmt.Sprintf("====end of pane %s====\n", id))
		case StatusUpdated:
			content.WriteString(fmt.Sprintf("====pane: %s (tmuxai_exec_pane) [UPDATED] (last updated: %s)\n", id, state.Timestamp.Format("15:04:05")))
			// Show previous content first, then new content
			if state.PreviousContent != "" {
				content.WriteString(state.PreviousContent + "\n")
			}
			content.WriteString("___NEW-CONTENT___\n")
			// Try to show only the new part if possible
			if state.PreviousContent != "" && strings.HasPrefix(state.Content, state.PreviousContent) {
				newPart := strings.TrimPrefix(state.Content, state.PreviousContent)
				content.WriteString(newPart + "\n")
			} else {
				content.WriteString(state.Content + "\n")
			}
			content.WriteString("____END-OF-NEW-CONTENT____\n")
			content.WriteString(fmt.Sprintf("====end of pane %s====\n", id))
		case StatusUnchanged:
			content.WriteString(fmt.Sprintf("====pane: %s (tmuxai_exec_pane) [UPDATED] (last updated: %s)\n", id, state.Timestamp.Format("15:04:05")))
			content.WriteString("___NEW-CONTENT___\n")
			content.WriteString("--- nothing new ---\n")
			content.WriteString("____END-OF-NEW-CONTENT____\n")
			content.WriteString(fmt.Sprintf("====end of pane %s====\n", id))
		case StatusRemoved:
			// Don't show removed panes in current session
			continue
		}
		content.WriteString("--\n")
	}

	// Add Current Chat History
	content.WriteString("\ncurrent chat\n\n")
	// Show conversation history with clean format
	reUserInput := regexp.MustCompile(`(?s)user : (.*)\nai :`)
	for _, msg := range b.Manager.Messages {
		if msg.FromUser {
			content.WriteString("user :\n")
			// Extract clean user input from structured message if needed
			matches := reUserInput.FindStringSubmatch(msg.Content)
			if len(matches) > 1 {
				userInput := strings.TrimSpace(matches[1])
				content.WriteString(userInput + "\n")
			} else {
				// Fallback in case regex fails
				content.WriteString(msg.Content + "\n")
			}
		} else {
			content.WriteString("ai :\n")
			// Parse AI response to get clean message
			parsedResponse, err := b.Manager.parseAIResponse(msg.Content)
			if err == nil && parsedResponse.Message != "" {
				content.WriteString(parsedResponse.Message + "\n")
			} else {
				content.WriteString(msg.Content + "\n")
			}
		}
		content.WriteString("\n")
	}
	
	// Add the current user input
	content.WriteString("user : " + userInput + "\n")
	content.WriteString("ai :\n\n")
	
	content.WriteString("..\n..\n..\n\n")

	return content.String()
}
