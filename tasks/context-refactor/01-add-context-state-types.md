# Task 1: Add Context State Data Structures

## 🎯 **Goal**
Add new data structures to `internal/types.go` to support the structured context system with timestamps and state tracking.

## 📋 **What to Achieve**
- Add new types for context state management
- Add timestamp tracking for all context items
- Add status markers for items (ACTIVE, REMOVED, OLD_SESSION)
- Add old session data structures

## 🔧 **How to Achieve**

### **Step 1: Add to `internal/types.go`**
Add these new type definitions at the end of the file:

```go
// Context state management types
type ItemStatus string

const (
    StatusActive     ItemStatus = "ACTIVE"
    StatusRemoved    ItemStatus = "REMOVED" 
    StatusOldSession ItemStatus = "OLD_SESSION"
)

type SectionState struct {
    Content       string      `json:"content"`
    LastChanged   int         `json:"last_changed"`    // which message number
    Hash          string      `json:"hash"`            // content hash for change detection
    Size          int         `json:"size"`            // content size in tokens
    Status        ItemStatus  `json:"status"`          // ACTIVE, REMOVED, OLD_SESSION
    Timestamp     time.Time   `json:"timestamp"`       // when last updated
    RemovedAt     time.Time   `json:"removed_at"`      // when removed (if removed)
}

type ContextState struct {
    RepoMap        SectionState                    `json:"repo_map"`
    Files          map[string]SectionState         `json:"files"`          // filepath -> state
    Panes          map[string]SectionState         `json:"panes"`          // paneID -> state
    Prompts        SectionState                    `json:"prompts"`
    OldSession     *OldSessionData                 `json:"old_session"`    // restored session data
    LastUpdate     int                             `json:"last_update"`    // message number
    CurrentTime    time.Time                       `json:"current_time"`   // current timestamp
}

type OldSessionData struct {
    SessionName   string                          `json:"session_name"`
    SavedAt       time.Time                       `json:"saved_at"`
    Files         map[string]SectionState         `json:"files"`
    Panes         map[string]SectionState         `json:"panes"`
    Conversation  []ChatMessage                   `json:"conversation"`
}
```

### **Step 2: Add Helper Methods**
Add these helper methods at the end of `internal/types.go`:

```go
// Helper methods for ContextState
func NewContextState() *ContextState {
    return &ContextState{
        Files:       make(map[string]SectionState),
        Panes:       make(map[string]SectionState),
        CurrentTime: time.Now(),
        LastUpdate:  0,
    }
}

func (s *SectionState) IsChanged(newContent string) bool {
    newHash := fmt.Sprintf("%x", sha256.Sum256([]byte(newContent)))
    return s.Hash != newHash
}

func (s *SectionState) UpdateContent(content string, messageNum int) {
    s.Content = content
    s.Hash = fmt.Sprintf("%x", sha256.Sum256([]byte(content)))
    s.LastChanged = messageNum
    s.Timestamp = time.Now()
    s.Status = StatusActive
}

func (s *SectionState) MarkAsRemoved(messageNum int) {
    s.Status = StatusRemoved
    s.RemovedAt = time.Now()
    s.LastChanged = messageNum
}
```

### **Step 3: Add Required Import**
Add this import at the top of `internal/types.go`:

```go
import (
    "crypto/sha256"
    "fmt"
    "time"
)
```

## ✅ **Success Criteria**

### **Verification Steps:**
1. **Compile Check**: Run `go build .` - should compile without errors
2. **Type Check**: Verify all new types are properly defined
3. **Import Check**: Ensure `crypto/sha256` import is added
4. **Method Check**: Verify helper methods are accessible

### **Expected Results:**
- [ ] File compiles successfully
- [ ] New types `ContextState`, `SectionState`, `OldSessionData` are defined
- [ ] Status constants `StatusActive`, `StatusRemoved`, `StatusOldSession` are defined
- [ ] Helper methods `NewContextState()`, `IsChanged()`, `UpdateContent()`, `MarkAsRemoved()` work
- [ ] All JSON tags are properly set for serialization

### **Test Command:**
```bash
cd /path/to/tmuxai
go build .
echo "✅ Types added successfully if no errors above"
```

## 📝 **Notes**
- This task only adds data structures, no business logic
- These types will be used by subsequent tasks
- The `crypto/sha256` import is needed for content change detection
- JSON tags enable session persistence (future feature)

## 🔗 **Next Task**
After this is complete, move to `02-add-context-state-to-manager.md`