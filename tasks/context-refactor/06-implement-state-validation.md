# Task 6: Implement State Validation System

## 🎯 **Goal**
Add state validation to detect file deletions, tmux session changes, and other edge cases before building each message.

## 📋 **What to Achieve**
- Validate file existence before each message
- Detect tmux session termination or changes
- Handle edge cases gracefully
- Mark removed items appropriately
- Prevent context corruption

## 🔧 **How to Achieve**

### **Step 1: Create State Validation Functions**
Create a new file `internal/state_validation.go`:

```go
package internal

import (
    "fmt"
    "os"
    "strings"
    "time"
    
    "github.com/alvinunreal/tmuxai/logger"
    "github.com/alvinunreal/tmuxai/system"
)

// StateChange represents a detected change in system state
type StateChange struct {
    Type        string    `json:"type"`         // file_removed, file_added, session_lost, etc.
    Item        string    `json:"item"`         // file path, pane ID, etc.
    Message     string    `json:"message"`      // human readable description
    Timestamp   time.Time `json:"timestamp"`    // when detected
    MessageNum  int       `json:"message_num"`  // current message number
}

// validateSystemState checks for changes in files, panes, and tmux session
func (m *Manager) validateSystemState() []StateChange {
    var changes []StateChange
    
    // Validate file existence
    fileChanges := m.validateFileExistence()
    changes = append(changes, fileChanges...)
    
    // Validate tmux session
    sessionChanges := m.validateTmuxSession()
    changes = append(changes, sessionChanges...)
    
    // Validate repo structure (detect new files)
    repoChanges := m.validateRepoStructure()
    changes = append(changes, repoChanges...)
    
    return changes
}

// validateFileExistence checks if tracked files still exist
func (m *Manager) validateFileExistence() []StateChange {
    var changes []StateChange
    
    for filePath, state := range m.ContextState.Files {
        // Skip already removed files
        if state.Status == StatusRemoved {
            continue
        }
        
        // Check if file still exists
        if _, err := os.Stat(filePath); os.IsNotExist(err) {
            changes = append(changes, StateChange{
                Type:       "file_removed",
                Item:       filePath,
                Message:    fmt.Sprintf("File %s no longer exists", filePath),
                Timestamp:  time.Now(),
                MessageNum: m.MessageCounter,
            })
            
            // Mark as removed in context state
            m.MarkFileAsRemoved(filePath)
            logger.Info("File removed from filesystem: %s", filePath)
        }
    }
    
    return changes
}

// validateTmuxSession checks if tmux session is still valid
func (m *Manager) validateTmuxSession() []StateChange {
    var changes []StateChange
    
    // Check if we can still get current pane ID
    currentPaneID, err := system.TmuxCurrentPaneId()
    if err != nil {
        changes = append(changes, StateChange{
            Type:       "session_lost",
            Item:       m.PaneId,
            Message:    "Tmux session terminated or lost",
            Timestamp:  time.Now(),
            MessageNum: m.MessageCounter,
        })
        
        // Mark all panes as removed
        for paneID := range m.ContextState.Panes {
            m.MarkPaneAsRemoved(paneID)
        }
        
        logger.Error("Tmux session lost: %v", err)
        return changes
    }
    
    // Check if we're in a different session
    if currentPaneID != m.PaneId {
        changes = append(changes, StateChange{
            Type:       "session_changed",
            Item:       fmt.Sprintf("%s -> %s", m.PaneId, currentPaneID),
            Message:    "Tmux session changed",
            Timestamp:  time.Now(),
            MessageNum: m.MessageCounter,
        })
        
        logger.Info("Tmux session changed from %s to %s", m.PaneId, currentPaneID)
        // Update to new session
        m.PaneId = currentPaneID
    }
    
    return changes
}

// validateRepoStructure detects new files in the repository
func (m *Manager) validateRepoStructure() []StateChange {
    var changes []StateChange
    
    // Only check if we have repo map enabled
    if m.RepoMap == nil || !m.RepoMap.IsEnabled() {
        return changes
    }
    
    // Get current repo map
    currentRepoMap, err := m.RepoMap.GetMap()
    if err != nil {
        return changes
    }
    
    // Simple check: if repo map content changed significantly, there might be new files
    if m.ContextState.RepoMap.Content != "" && m.ContextState.RepoMap.IsChanged(currentRepoMap) {
        changes = append(changes, StateChange{
            Type:       "repo_structure_changed",
            Item:       "repository",
            Message:    "Repository structure changed (files added/removed/moved)",
            Timestamp:  time.Now(),
            MessageNum: m.MessageCounter,
        })
        
        logger.Info("Repository structure changed")
    }
    
    return changes
}

// handleStateChanges processes detected state changes
func (m *Manager) handleStateChanges(changes []StateChange) error {
    if len(changes) == 0 {
        return nil
    }
    
    for _, change := range changes {
        switch change.Type {
        case "session_lost":
            return fmt.Errorf("tmux session lost - cannot continue")
            
        case "session_changed":
            m.Println(fmt.Sprintf("Notice: Tmux session changed to %s", change.Item))
            
        case "file_removed":
            logger.Info("File removed: %s", change.Item)
            // File already marked as removed in validateFileExistence
            
        case "repo_structure_changed":
            logger.Info("Repository structure changed")
            // Repo map will be updated in next change detection cycle
            
        default:
            logger.Info("State change detected: %s - %s", change.Type, change.Message)
        }
    }
    
    return nil
}

// validateContextIntegrity performs a comprehensive check of context state
func (m *Manager) validateContextIntegrity() error {
    // Check if ContextState is properly initialized
    if m.ContextState == nil {
        return fmt.Errorf("context state not initialized")
    }
    
    if m.ContextState.Files == nil {
        m.ContextState.Files = make(map[string]SectionState)
    }
    
    if m.ContextState.Panes == nil {
        m.ContextState.Panes = make(map[string]SectionState)
    }
    
    // Validate message counter
    if m.MessageCounter < 0 {
        m.MessageCounter = 0
    }
    
    return nil
}
```

### **Step 2: Integrate State Validation into ProcessUserMessage**
In `internal/process_message.go`, add state validation at the beginning of `ProcessUserMessage` function.

**Find this code (around line 18):**
```go
// Check if context management is needed before sending
if m.needSquash() {
    m.Println("Exceeded context size, squashing history...")
    m.squashHistory()
}
```

**Add this BEFORE the squash check:**
```go
// Validate system state before processing message
if err := m.validateContextIntegrity(); err != nil {
    m.Println(fmt.Sprintf("Context integrity error: %v", err))
    return false
}

stateChanges := m.validateSystemState()
if err := m.handleStateChanges(stateChanges); err != nil {
    m.Println(fmt.Sprintf("Critical state change: %v", err))
    m.Status = ""
    return false
}
```

### **Step 3: Add State Change Reporting to Structured Messages**
Add this function to `internal/structured_message.go`:

```go
// buildStateChangesSection creates a section for recent state changes
func (m *Manager) buildStateChangesSection() string {
    // Get recent state changes (this is a simple implementation)
    // In a full implementation, you might want to track recent changes
    
    // For now, just validate and report any immediate issues
    stateChanges := m.validateSystemState()
    
    if len(stateChanges) == 0 {
        return ""
    }
    
    var changeEntries []string
    changeEntries = append(changeEntries, "[SYSTEM STATE CHANGES DETECTED]")
    
    for _, change := range stateChanges {
        changeEntries = append(changeEntries, 
            fmt.Sprintf("- %s: %s (detected at %s)", 
                change.Type, change.Message, change.Timestamp.Format("15:04:05")))
    }
    
    return fmt.Sprintf("----STATE-CHANGES----\n%s\n----END-OF-STATE-CHANGES----", 
        strings.Join(changeEntries, "\n"))
}
```

### **Step 4: Update buildStructuredMessage to Include State Changes**
In `internal/structured_message.go`, update the `buildStructuredMessage` function to include state changes:

**Find this code:**
```go
// Old session data section (if exists)
oldSessionSection := m.buildOldSessionSection()
if oldSessionSection != "" {
    sections = append(sections, oldSessionSection)
}
```

**Add after it:**
```go
// State changes section (if any)
stateChangesSection := m.buildStateChangesSection()
if stateChangesSection != "" {
    sections = append(sections, stateChangesSection)
}
```

### **Step 5: Add Graceful Recovery Methods**
Add these methods to `internal/state_validation.go`:

```go
// recoverFromSessionLoss attempts to recover from tmux session loss
func (m *Manager) recoverFromSessionLoss() error {
    m.Println("Attempting to recover from session loss...")
    
    // Try to create a new session
    newPaneId, err := system.TmuxCreateSession()
    if err != nil {
        return fmt.Errorf("failed to create new session: %w", err)
    }
    
    m.PaneId = newPaneId
    m.Println(fmt.Sprintf("Created new tmux session: %s", newPaneId))
    
    // Clear pane context state
    m.ContextState.Panes = make(map[string]SectionState)
    
    // Reinitialize exec pane
    m.InitExecPane()
    
    return nil
}

// cleanupRemovedItems removes old removed items from context state
func (m *Manager) cleanupRemovedItems() {
    const maxRemovedAge = 10 // Keep removed items for 10 messages
    
    // Clean up old removed files
    for filePath, state := range m.ContextState.Files {
        if state.Status == StatusRemoved && 
           m.MessageCounter - state.LastChanged > maxRemovedAge {
            delete(m.ContextState.Files, filePath)
        }
    }
    
    // Clean up old removed panes
    for paneID, state := range m.ContextState.Panes {
        if state.Status == StatusRemoved && 
           m.MessageCounter - state.LastChanged > maxRemovedAge {
            delete(m.ContextState.Panes, paneID)
        }
    }
}
```

## ✅ **Success Criteria**

### **Verification Steps:**
1. **Compile Check**: Run `go build .` - should compile without errors
2. **File Deletion**: Delete a file that was read, verify it's marked as [REMOVED]
3. **Session Loss**: Kill tmux session, verify graceful handling
4. **New Files**: Add files to repo, verify detection
5. **Context Integrity**: Verify no crashes on edge cases

### **Expected Results:**
- [ ] File compiles successfully
- [ ] State validation runs before each message
- [ ] Deleted files are detected and marked as [REMOVED]
- [ ] Tmux session loss is detected and handled
- [ ] New files in repo are detected
- [ ] Context integrity is maintained
- [ ] Graceful recovery from session loss works
- [ ] Old removed items are cleaned up

### **Test Commands:**
```bash
cd /path/to/tmuxai
go build .
echo "✅ State validation implemented successfully if no errors above"
```

### **Manual Test Sequence:**
1. **Start TmuxAI**: `./tmuxai --agentic`
2. **Read a file**: `<ReadFile>test.txt</ReadFile>`
3. **Delete the file**: `rm test.txt` (in another terminal)
4. **Send message**: Verify file shows as [REMOVED]
5. **Kill session**: `tmux kill-session` (in another terminal)
6. **Send message**: Verify graceful error handling

### **Expected Behavior:**
- **File deletion**: Shows `file: test.txt [REMOVED] (removed at: 14:35:22)`
- **Session loss**: Shows error message and attempts recovery
- **New files**: Shows in repo map updates
- **No crashes**: System handles all edge cases gracefully

## 📝 **Notes**
- State validation prevents context corruption
- Graceful recovery maintains user experience
- Cleanup prevents memory leaks from old removed items
- Session loss detection prevents hanging
- File existence validation prevents stale references

## 🔗 **Next Task**
After this is complete, move to `07-implement-session-restoration.md`