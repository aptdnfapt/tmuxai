# Task 2: Add Context State to Manager

## 🎯 **Goal**
Add the new context state tracking system to the Manager struct and initialize it properly.

## 📋 **What to Achieve**
- Add ContextState field to Manager struct
- Initialize context state in NewManager()
- Add message counter tracking
- Prepare for structured message building

## 🔧 **How to Achieve**

### **Step 1: Update Manager Struct**
In `internal/manager.go`, find the Manager struct (around line 15) and add the new field:

```go
type Manager struct {
    Config           *config.Config
    AiClient         *AiClient
    Status           string // running, waiting, done
    PaneId           string
    ExecPane         *system.TmuxPaneDetails
    Messages         []ChatMessage
    ExecHistory      []CommandExecHistory
    ReadFiles        []string
    WatchMode        bool
    OS               string
    SessionOverrides map[string]interface{} // session-only config overrides
    PreparedPanes    map[string]bool
    LastExecPaneID   string
    SessionPath      string
    isRestore        bool
    RepoMap          *RepoMapHandler
    
    // NEW: Context state tracking
    ContextState     *ContextState          // NEW: Add this line
    MessageCounter   int                    // NEW: Add this line
}
```

### **Step 2: Initialize Context State**
In `internal/manager.go`, find the `NewManager` function (around line 35) and add initialization after line 77:

```go
// After this existing line:
// isRestore:        isRestore,

// ADD these lines:
ContextState:     NewContextState(),
MessageCounter:   0,
```

### **Step 3: Add Context State Helper Methods**
Add these methods at the end of `internal/manager.go`:

```go
// Context state management methods
func (m *Manager) IncrementMessageCounter() int {
    m.MessageCounter++
    m.ContextState.LastUpdate = m.MessageCounter
    m.ContextState.CurrentTime = time.Now()
    return m.MessageCounter
}

func (m *Manager) GetCurrentMessageNumber() int {
    return m.MessageCounter
}

func (m *Manager) UpdateFileState(filePath, content string) {
    if m.ContextState.Files == nil {
        m.ContextState.Files = make(map[string]SectionState)
    }
    
    state := m.ContextState.Files[filePath]
    state.UpdateContent(content, m.MessageCounter)
    m.ContextState.Files[filePath] = state
}

func (m *Manager) UpdatePaneState(paneID, content string) {
    if m.ContextState.Panes == nil {
        m.ContextState.Panes = make(map[string]SectionState)
    }
    
    state := m.ContextState.Panes[paneID]
    state.UpdateContent(content, m.MessageCounter)
    m.ContextState.Panes[paneID] = state
}

func (m *Manager) MarkFileAsRemoved(filePath string) {
    if state, exists := m.ContextState.Files[filePath]; exists {
        state.MarkAsRemoved(m.MessageCounter)
        m.ContextState.Files[filePath] = state
    }
}

func (m *Manager) MarkPaneAsRemoved(paneID string) {
    if state, exists := m.ContextState.Panes[paneID]; exists {
        state.MarkAsRemoved(m.MessageCounter)
        m.ContextState.Panes[paneID] = state
    }
}
```

### **Step 4: Add Required Import**
Make sure `internal/manager.go` has the time import at the top:

```go
import (
    "fmt"
    "os"
    "strings"
    "time"    // Make sure this exists
    
    "github.com/alvinunreal/tmuxai/config"
    "github.com/alvinunreal/tmuxai/logger"
    "github.com/alvinunreal/tmuxai/system"
    "github.com/fatih/color"
)
```

## ✅ **Success Criteria**

### **Verification Steps:**
1. **Compile Check**: Run `go build .` - should compile without errors
2. **Manager Creation**: Verify NewManager() creates ContextState properly
3. **Helper Methods**: Verify context state methods are accessible
4. **Initialization**: Verify ContextState is not nil after NewManager()

### **Expected Results:**
- [ ] File compiles successfully
- [ ] Manager struct has `ContextState *ContextState` field
- [ ] Manager struct has `MessageCounter int` field
- [ ] NewManager() initializes ContextState with `NewContextState()`
- [ ] NewManager() initializes MessageCounter to 0
- [ ] Helper methods are defined and accessible
- [ ] All imports are correct

### **Test Command:**
```bash
cd /path/to/tmuxai
go build .
echo "✅ Manager context state added successfully if no errors above"
```

### **Quick Test:**
You can add this temporary test in main.go to verify:
```go
// Temporary test - remove after verification
manager, err := NewManager(cfg, false)
if err != nil {
    log.Fatal(err)
}
if manager.ContextState == nil {
    log.Fatal("ContextState not initialized")
}
if manager.MessageCounter != 0 {
    log.Fatal("MessageCounter not initialized to 0")
}
fmt.Println("✅ Context state properly initialized")
```

## 📝 **Notes**
- This task only adds the infrastructure, no business logic yet
- The helper methods will be used by subsequent tasks
- MessageCounter tracks which message number we're on
- ContextState.CurrentTime will be updated on each message

## 🔗 **Next Task**
After this is complete, move to `03-create-structured-message-builder.md`