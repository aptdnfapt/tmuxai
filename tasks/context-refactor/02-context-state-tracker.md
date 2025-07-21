# Plan 02: Context State Tracker

**Objective:** Create a new `ContextStateTracker` component to manage the `ContextState` and detect changes between messages. This involves creating a new file, `internal/context_tracker.go`, and adding the tracker to the `Manager` struct.

---

## 🔧 **Technical Implementation**

### **1. Create New File: `internal/context_tracker.go`**

Create a new file to house the logic for the context state tracker.

```go
package internal

import (
	"crypto/sha256"
	"encoding/hex"
	"time"
)

// ContextStateTracker manages the state of the context sent to the AI.
type ContextStateTracker struct {
	CurrentState *ContextState
	MessageCount int
}

// NewContextStateTracker initializes a new tracker.
func NewContextStateTracker() *ContextStateTracker {
	return &ContextStateTracker{
		CurrentState: &ContextState{
			Files: make(map[string]SectionState),
			Panes: make(map[string]SectionState),
		},
	}
}

// UpdateRepoMap checks if the repo map has changed and updates its state.
func (t *ContextStateTracker) UpdateRepoMap(content string) {
	t.updateSection(&t.CurrentState.RepoMap, content)
}

// UpdateFile checks if a file's content has changed and updates its state.
func (t *ContextStateTracker) UpdateFile(path, content string) {
	fileState := t.CurrentState.Files[path]
	t.updateSection(&fileState, content)
	t.CurrentState.Files[path] = fileState
}

// UpdatePane checks if a pane's content has changed and updates its state.
func (t *ContextStateTracker) UpdatePane(paneID, content string) {
	paneState := t.CurrentState.Panes[paneID]
	t.updateSection(&paneState, content)
	t.CurrentState.Panes[paneID] = paneState
}

// IncrementMessageCount increments the message counter.
func (t *ContextStateTracker) IncrementMessageCount() {
	t.MessageCount++
	t.CurrentState.LastUpdate = t.MessageCount
	t.CurrentState.CurrentTime = time.Now()
}

// Helper function to update a section's state.
func (t *ContextStateTracker) updateSection(state *SectionState, newContent string) {
	newHash := hash(newContent)
	if state.Hash == "" { // First time seeing this item
		state.Status = StatusNew
	} else if state.Hash != newHash {
		state.Status = StatusUpdated
	} else {
		state.Status = StatusUnchanged
	}

	state.Content = newContent
	state.Hash = newHash
	state.Timestamp = time.Now()
	state.LastChanged = t.MessageCount
}

// hash computes the SHA256 hash of a string.
func hash(s string) string {
	h := sha256.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}

```

### **2. Integrate Tracker into the `Manager`**

Modify the `Manager` struct in `internal/manager.go` to include the new `ContextStateTracker`.

**File to Modify:** `internal/manager.go`

**Add the `ContextTracker` field to the `Manager` struct:**

```go
// ... existing Manager fields
	RepoMap          *RepoMapHandler
	ContextTracker   *ContextStateTracker // Add this line
}
```

**Initialize the tracker in the `NewManager` function:**

```go
// In NewManager, after the manager is initialized:
	manager.ContextTracker = NewContextStateTracker()

	// Session loading logic
// ...
```

---

## ✅ **Validation**

- The project should compile successfully after these changes (`go build .`).
- The new `ContextStateTracker` is initialized but not yet used. No functional changes are expected.

## ➡️ **Next Step**

Proceed to `03-structured-message-builder.md` to create the component that will build the new, efficient message format using the state tracked by this new component.
