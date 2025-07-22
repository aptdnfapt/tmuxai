package internal

import (
	"bytes"
	"fmt"
	"strings"
	"time"
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

	b.appendSection(&buf, "CURRENT-TIME", b.buildTimeSection)
	b.appendSection(&buf, "PROMPTS", b.buildPromptsSection)
	b.appendSection(&buf, "OLD-SESSION-DATA", b.buildOldSessionSection)
	b.appendSection(&buf, "REPO-MAP", b.buildRepoMapSection)
	b.appendSection(&buf, "FILES", b.buildFilesSection)
	b.appendSection(&buf, "PANES", b.buildPanesSection)
	b.appendSection(&buf, "CONVERSATION", func() string { return userInput })

	return buf.String()
}

// appendSection is a helper to build and append a section to the buffer.
func (b *StructuredMessageBuilder) appendSection(buf *bytes.Buffer, title string, builderFunc func() string) {
	content := builderFunc()
	if content != "" {
		buf.WriteString(fmt.Sprintf("----%s----\n", title))
		buf.WriteString(content)
		buf.WriteString(fmt.Sprintf("----END-OF-%s----\n\n", title))
	}
}

// buildTimeSection creates the timestamp header.
func (b *StructuredMessageBuilder) buildTimeSection() string {
	return fmt.Sprintf("Date: %s\n", b.Manager.ContextTracker.CurrentState.CurrentTime.Format(time.RFC1123))
}

// buildPromptsSection builds the prompts section.
func (b *StructuredMessageBuilder) buildPromptsSection() string {
	// This will be implemented in a later step. For now, it returns an empty string.
	return ""
}

// buildOldSessionSection builds the section for restored session data.
func (b *StructuredMessageBuilder) buildOldSessionSection() string {
	if b.Manager.OldSession == nil {
		return ""
	}

	var content bytes.Buffer
	oldSession := b.Manager.OldSession

	content.WriteString(fmt.Sprintf("[Restored from: \"%s\" - saved: %s]\n\n", oldSession.SessionName, oldSession.SavedAt.Format(time.RFC1123)))

	// Old Panes
	for id, state := range oldSession.Panes {
		content.WriteString(fmt.Sprintf("pane: %s [OLD SESSION] (from: %s)\n%s\n", id, oldSession.SavedAt.Format(time.Kitchen), state.Content))
	}

	// Old Conversation
	if len(oldSession.Conversation) > 0 {
		content.WriteString("\n[OLD CONVERSATION HISTORY]\n")
		for _, msg := range oldSession.Conversation {
			role := "AI"
			if msg.FromUser {
				role = "User"
			}
			content.WriteString(fmt.Sprintf("%s: \"%s\"\n", role, msg.Content))
		}
	}

	return content.String()
}

// buildRepoMapSection builds the repo map section with status.
func (b *StructuredMessageBuilder) buildRepoMapSection() string {
	repoMapState := b.Manager.ContextTracker.CurrentState.RepoMap
	switch repoMapState.Status {
	case StatusNew, StatusUpdated:
		return fmt.Sprintf("[%s]\n%s", repoMapState.Status, repoMapState.Content)
	case StatusUnchanged:
		return fmt.Sprintf("[%s since message %d]", repoMapState.Status, repoMapState.LastChanged)
	default:
		return ""
	}
}

// buildFilesSection builds the files section with status for each file.
func (b *StructuredMessageBuilder) buildFilesSection() string {
	var content bytes.Buffer
	for path, state := range b.Manager.ContextTracker.CurrentState.Files {
		switch state.Status {
		case StatusNew, StatusUpdated:
			content.WriteString(fmt.Sprintf("file: %s [%s] (last modified: %s)\n%s\n", path, state.Status, state.Timestamp.Format(time.Kitchen), state.Content))
		case StatusUnchanged:
			content.WriteString(fmt.Sprintf("file: %s [%s since message %d]\n", path, state.Status, state.LastChanged))
		case StatusRemoved:
			content.WriteString(fmt.Sprintf("file: %s [%s]\n", path, state.Status))
		}
	}
	return content.String()
}

// buildPanesSection builds the panes section with status for each pane.
func (b *StructuredMessageBuilder) buildPanesSection() string {
	var content bytes.Buffer
	for id, state := range b.Manager.ContextTracker.CurrentState.Panes {
		switch state.Status {
		case StatusNew:
			content.WriteString(fmt.Sprintf("pane: %s [%s] (last updated: %s)\n%s\n", id, state.Status, state.Timestamp.Format(time.Kitchen), state.Content))
		case StatusUpdated:
			var paneContent string
			// Best-effort diff for appended content
			if state.PreviousContent != "" && strings.HasPrefix(state.Content, state.PreviousContent) {
				newPart := strings.TrimSpace(strings.TrimPrefix(state.Content, state.PreviousContent))
				// Only show diff if there is new content
				if newPart != "" {
					paneContent = fmt.Sprintf("----NEW-CONTENT----\n%s\n----END-OF-NEW-CONTENT----", newPart)
				} else {
					// The pane was updated, but our simple diff logic didn't find any new appended content.
					// This can happen with whitespace changes or other minor edits.
					// To avoid sending the whole duplicated pane, we send an empty content. The AI sees [UPDATED] and knows *something* changed.
					paneContent = ""
				}
			} else {
				// Can't diff cleanly (e.g. content removed) or no previous content, just send the whole thing.
				paneContent = state.Content
			}
			content.WriteString(fmt.Sprintf("pane: %s [%s] (last updated: %s)\n%s\n", id, state.Status, state.Timestamp.Format(time.Kitchen), paneContent))
		case StatusUnchanged:
			content.WriteString(fmt.Sprintf("pane: %s [%s since message %d]\n", id, state.Status, state.LastChanged))
		case StatusRemoved:
			content.WriteString(fmt.Sprintf("pane: %s [%s]\n", id, state.Status))
		}
	}
	return content.String()
}
