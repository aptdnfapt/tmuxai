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
	"time"
)

// StructuredMessageBuilder creates the structured message for the AI.
type StructuredMessageBuilder struct {
	Tracker *ContextStateTracker
}

// NewStructuredMessageBuilder initializes a new message builder.
func NewStructuredMessageBuilder(tracker *ContextStateTracker) *StructuredMessageBuilder {
	return &StructuredMessageBuilder{Tracker: tracker}
}

// BuildMessage constructs the full message string.
func (b *StructuredMessageBuilder) BuildMessage(userInput string) string {
	var buf bytes.Buffer

	b.appendSection(&buf, "CURRENT-TIME", b.buildTimeSection)
	b.appendSection(&buf, "PROMPTS", b.buildPromptsSection)
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
	return fmt.Sprintf("Date: %s\n", b.Tracker.CurrentState.CurrentTime.Format(time.RFC1123)) 
}

// buildPromptsSection builds the prompts section.
func (b *StructuredMessageBuilder) buildPromptsSection() string {
	// This will be implemented in a later step. For now, it returns an empty string.
	return ""
}

// buildRepoMapSection builds the repo map section with status.
func (b *StructuredMessageBuilder) buildRepoMapSection() string {
	repoMapState := b.Tracker.CurrentState.RepoMap
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
	for path, state := range b.Tracker.CurrentState.Files {
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
	for id, state := range b.Tracker.CurrentState.Panes {
		switch state.Status {
		case StatusNew, StatusUpdated:
			content.WriteString(fmt.Sprintf("pane: %s [%s] (last updated: %s)\n%s\n", id, state.Status, state.Timestamp.Format(time.Kitchen), state.Content))
		case StatusUnchanged:
			content.WriteString(fmt.Sprintf("pane: %s [%s since message %d]\n", id, state.Status, state.LastChanged))
		case StatusRemoved:
			content.WriteString(fmt.Sprintf("pane: %s [%s]\n", id, state.Status))
		}
	}
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
	manager.MessageBuilder = NewStructuredMessageBuilder(manager.ContextTracker)

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
