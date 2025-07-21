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
		state.PreviousContent = ""
	} else if state.Hash != newHash {
		state.Status = StatusUpdated
		state.PreviousContent = state.Content // Preserve old content
	} else {
		state.Status = StatusUnchanged
	}

	if state.Status != StatusUnchanged {
		state.Content = newContent
		state.Hash = newHash
		state.Timestamp = time.Now()
		state.LastChanged = t.MessageCount
	}
}

// hash computes the SHA256 hash of a string.
func hash(s string) string {
	h := sha256.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}
