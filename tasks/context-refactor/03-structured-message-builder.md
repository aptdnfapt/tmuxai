# Plan 03: Structured Message Builder

**Objective:** Create a `StructuredMessageBuilder` to construct the new, efficient message format using the state from the `ContextStateTracker`. This will replace the current ad-hoc message construction in `ProcessUserMessage`.

---

## 
**Technical Implementation**

### **1. Create New File: `internal/message_builder.go`**

This file will contain the logic for building the structured message.

```go
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

	// Static context sections that are always present
	b.appendSection(&buf, "PROMPTS", b.buildPromptsSection)
	b.appendSection(&buf, "REPO-MAP", b.buildRepoMapSection)
	b.appendSection(&buf, "FILES", b.buildFilesSection)

	// Historical context from a restored session
	b.appendSection(&buf, "OLD-SESSION-DATA", b.buildOldSessionSection)

	// Live context from the current, active session
	b.appendSection(&buf, "CURRENT-SESSION-DATA", func() string {
		return b.buildCurrentSessionSection(userInput)
	})

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

// buildPromptsSection will contain system prompts.
func (b *StructuredMessageBuilder) buildPromptsSection() string {
	// This will be implemented in a later step. For now, it returns an empty string.
	return ""
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

// buildFilesSection builds the files section with sub-blocks for each file.
func (b *StructuredMessageBuilder) buildFilesSection() string {
	var content bytes.Buffer
	for path, state := range b.Manager.ContextTracker.CurrentState.Files {
		var header, footer, fileBlock string
		switch state.Status {
		case StatusNew, StatusUpdated:
			header = fmt.Sprintf("--- file: %s [%s] (last modified: %s) ---\n", path, state.Status, state.Timestamp.Format(time.RFC1123))
			footer = fmt.Sprintf("\n--- end of file: %s ---\n\n", path)
			fileBlock = header + state.Content + footer
		case StatusUnchanged:
			// For UNCHANGED files, we still send the full content. The status is metadata for the AI.
			header = fmt.Sprintf("--- file: %s [%s since message %d] (last modified: %s) ---\n", path, state.Status, state.LastChanged, state.Timestamp.Format(time.RFC1123))
			footer = fmt.Sprintf("\n--- end of file: %s ---\n\n", path)
			fileBlock = header + state.Content + footer
		case StatusRemoved:
			fileBlock = fmt.Sprintf("--- file: %s [%s] (removed at: %s) ---\n\n", path, state.Status, state.RemovedAt.Format(time.RFC1123))
		}
		content.WriteString(fileBlock)
	}
	// Trim the final trailing newline for cleaner output.
	return strings.TrimSuffix(content.String(), "\n")
}

// buildOldSessionSection builds the section for restored session data, ensuring no nested context is included.
func (b *StructuredMessageBuilder) buildOldSessionSection() string {
	if b.Manager.OldSession == nil {
		return ""
	}

	var content bytes.Buffer
	oldSession := b.Manager.OldSession
	content.WriteString(fmt.Sprintf("[Restored from: \"%s\" - saved: %s]\n\n", oldSession.SessionName, oldSession.SavedAt.Format(time.RFC1123)))

	// Show the final state of old panes.
	content.WriteString("### [PREVIOUS PANES]\n")
	for id, state := range oldSession.Panes {
		content.WriteString(fmt.Sprintf("====pane: %s====\n%s\n====end of pane %s====\n", id, state.Content, id))
	}

	// Show a clean, parsed version of the old conversation history to avoid sending nested context.
	content.WriteString("\n### [PREVIOUS CHAT HISTORY]\n")
	for _, msg := range oldSession.Conversation {
		var role, messageText string
		if msg.FromUser {
			role = "User"
			// In the implementation, this would involve regex to extract the final user input
			// from the [CURRENT CHAT HISTORY] block of the saved message.
			messageText = "/* Extracted user input from saved message content */"
		} else {
			role = "AI"
			// In the implementation, this would call a parser on the saved AI response to
			// get only the user-facing message, stripping out tool calls.
			messageText = "/* Extracted AI message from saved response content */"
		}

		if strings.TrimSpace(messageText) != "" {
			content.WriteString(fmt.Sprintf("%s: %s\n", role, messageText))
		}
	}
	return content.String()
}

// buildCurrentSessionSection constructs the block for the live, current session.
func (b *StructuredMessageBuilder) buildCurrentSessionSection(userInput string) string {
	var content bytes.Buffer

	// Add current time
	content.WriteString(fmt.Sprintf("[CURRENT TIME: %s]\n\n", b.Manager.ContextTracker.CurrentState.CurrentTime.Format(time.RFC1123)))

	// Add Panes
	for id, state := range b.Manager.ContextTracker.CurrentState.Panes {
		switch state.Status {
		case StatusNew:
			content.WriteString(fmt.Sprintf("====pane: %s [%s]====\n___NEW-CONTENT___\n%s\n____END-OF-NEW-CONTENT____\n====end of pane %s====\n", id, state.Status, state.Content, id))
		case StatusUpdated:
			var paneContent string
			// Best-effort diff for appended content
			if state.PreviousContent != "" && strings.HasPrefix(state.Content, state.PreviousContent) {
				newPart := strings.TrimPrefix(state.Content, state.PreviousContent)
				paneContent = fmt.Sprintf("%s___NEW-CONTENT___\n%s\n____END-OF-NEW-CONTENT____", state.PreviousContent, newPart)
			} else {
				// Can't diff cleanly, send the whole thing with a clear NEW content block
				paneContent = fmt.Sprintf("%s___NEW-CONTENT___\n%s\n____END-OF-NEW-CONTENT____", state.PreviousContent, state.Content)
			}
			content.WriteString(fmt.Sprintf("====pane: %s [%s]====\n%s\n====end of pane %s====\n", id, state.Status, paneContent, id))
		case StatusUnchanged:
			content.WriteString(fmt.Sprintf("====pane: %s [%s since message %d]====\n", id, state.Status, state.LastChanged))
		case StatusRemoved:
			content.WriteString(fmt.Sprintf("====pane: %s [%s]====\n", id, state.Status))
		}
	}

	// Add Current Conversation
	content.WriteString("\n[CURRENT CHAT HISTORY]\n")
	// This part needs access to the current message history in the manager
	for _, msg := range b.Manager.Messages {
		role := "AI"
		if msg.FromUser {
			role = "User"
		}
		content.WriteString(fmt.Sprintf("%s: %s\n", role, msg.Content))
	}
	// Add the latest user input
	content.WriteString(fmt.Sprintf("User: %s\n", userInput))

	return content.String()
}
```

### **2. Update `Manager` to Use the Builder**

Modify `internal/manager.go` to use the new `StructuredMessageBuilder`.

**File to Modify:** `internal/manager.go`

**Add the `MessageBuilder` to the `Manager` struct:**

```go
// ... existing Manager fields
	ContextTracker   *ContextStateTracker
	MessageBuilder   *StructuredMessageBuilder // Add this line
}
```

**Initialize the builder in `NewManager`:**

```go
// In NewManager, after initializing the ContextTracker:
	manager.MessageBuilder = NewStructuredMessageBuilder(manager)

	// Session loading logic
// ...
```

---

## 
**Validation**

- The project should compile successfully (`go build .`).
- The `StructuredMessageBuilder` is created but not yet used to construct the final message. No functional changes are expected.

## 
**Next Step**

Proceed to `04-integration-and-refactoring.md` to replace the old message creation logic with the new builder and fully integrate the context tracking system.
