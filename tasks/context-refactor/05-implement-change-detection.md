# Task 5: Implement Change Detection System

## 🎯 **Goal**
Add smart change detection to only send updated content instead of duplicating everything in each message.

## 📋 **What to Achieve**
- Detect when files, panes, and repo map actually change
- Only send changed content with [UPDATED] markers
- Mark unchanged content with [UNCHANGED since message X]
- Implement content hashing for change detection

## 🔧 **How to Achieve**

### **Step 1: Create Change Detection Functions**
Add these functions to `internal/structured_message.go`:

```go
// detectRepoMapChanges checks if repo map has changed
func (m *Manager) detectRepoMapChanges() (bool, string) {
    currentRepoMap := m.getRepoMapContext()
    if currentRepoMap == "" {
        return false, ""
    }
    
    // Check if we have previous repo map state
    if m.ContextState.RepoMap.Content == "" {
        // First time - mark as updated
        m.ContextState.RepoMap.UpdateContent(currentRepoMap, m.MessageCounter)
        return true, currentRepoMap
    }
    
    // Check if content changed
    if m.ContextState.RepoMap.IsChanged(currentRepoMap) {
        m.ContextState.RepoMap.UpdateContent(currentRepoMap, m.MessageCounter)
        return true, currentRepoMap
    }
    
    return false, ""
}

// detectPaneChanges checks which panes have changed
func (m *Manager) detectPaneChanges() map[string]PaneChangeInfo {
    changes := make(map[string]PaneChangeInfo)
    
    // Get current pane states
    panes, err := m.GetTmuxPanes()
    if err != nil {
        return changes
    }
    
    for _, pane := range panes {
        currentContent, err := system.TmuxCapturePane(pane.Id, m.GetMaxCaptureLines())
        if err != nil {
            continue
        }
        
        // Check if we have previous state for this pane
        previousState, exists := m.ContextState.Panes[pane.Id]
        
        if !exists {
            // New pane
            m.UpdatePaneState(pane.Id, currentContent)
            changes[pane.Id] = PaneChangeInfo{
                Status:  "NEW",
                Content: currentContent,
                PaneDetails: pane,
            }
        } else if previousState.IsChanged(currentContent) {
            // Pane content changed
            m.UpdatePaneState(pane.Id, currentContent)
            changes[pane.Id] = PaneChangeInfo{
                Status:  "UPDATED",
                Content: currentContent,
                PaneDetails: pane,
            }
        } else {
            // Pane unchanged
            changes[pane.Id] = PaneChangeInfo{
                Status: "UNCHANGED",
                LastChanged: previousState.LastChanged,
                PaneDetails: pane,
            }
        }
    }
    
    // Check for removed panes
    for paneID, state := range m.ContextState.Panes {
        if state.Status == StatusRemoved {
            continue // Already marked as removed
        }
        
        // Check if pane still exists
        paneExists := false
        for _, pane := range panes {
            if pane.Id == paneID {
                paneExists = true
                break
            }
        }
        
        if !paneExists {
            m.MarkPaneAsRemoved(paneID)
            changes[paneID] = PaneChangeInfo{
                Status: "REMOVED",
                RemovedAt: m.MessageCounter,
            }
        }
    }
    
    return changes
}

// detectFileChanges checks which files have changed (for files we've read)
func (m *Manager) detectFileChanges() map[string]FileChangeInfo {
    changes := make(map[string]FileChangeInfo)
    
    for _, filePath := range m.ReadFiles {
        // Check if file still exists
        if _, err := os.Stat(filePath); os.IsNotExist(err) {
            // File was removed
            m.MarkFileAsRemoved(filePath)
            changes[filePath] = FileChangeInfo{
                Status: "REMOVED",
                RemovedAt: m.MessageCounter,
            }
            continue
        }
        
        // File exists - check if content changed
        content, err := os.ReadFile(filePath)
        if err != nil {
            continue
        }
        
        contentStr := string(content)
        previousState, exists := m.ContextState.Files[filePath]
        
        if !exists {
            // First time tracking this file
            m.UpdateFileState(filePath, contentStr)
            changes[filePath] = FileChangeInfo{
                Status: "NEW",
                Content: contentStr,
            }
        } else if previousState.IsChanged(contentStr) {
            // File content changed
            m.UpdateFileState(filePath, contentStr)
            changes[filePath] = FileChangeInfo{
                Status: "UPDATED", 
                Content: contentStr,
            }
        } else {
            // File unchanged
            changes[filePath] = FileChangeInfo{
                Status: "UNCHANGED",
                LastChanged: previousState.LastChanged,
            }
        }
    }
    
    return changes
}
```

### **Step 2: Add Change Info Types**
Add these types to `internal/types.go`:

```go
// Change detection types
type PaneChangeInfo struct {
    Status      string                    `json:"status"`      // NEW, UPDATED, UNCHANGED, REMOVED
    Content     string                    `json:"content"`     // Current content (if updated/new)
    LastChanged int                       `json:"last_changed"` // Message number when last changed
    RemovedAt   int                       `json:"removed_at"`   // Message number when removed
    PaneDetails system.TmuxPaneDetails    `json:"pane_details"`
}

type FileChangeInfo struct {
    Status      string `json:"status"`       // NEW, UPDATED, UNCHANGED, REMOVED  
    Content     string `json:"content"`      // Current content (if updated/new)
    LastChanged int    `json:"last_changed"` // Message number when last changed
    RemovedAt   int    `json:"removed_at"`   // Message number when removed
}
```

### **Step 3: Update Section Builders to Use Change Detection**
Replace the section builder functions in `internal/structured_message.go`:

```go
// buildRepoMapSection creates the repo map section with change detection
func (m *Manager) buildRepoMapSection() string {
    changed, content := m.detectRepoMapChanges()
    
    if !changed && m.ContextState.RepoMap.Content != "" {
        // Repo map unchanged
        return fmt.Sprintf("----REPO-MAP----\n[UNCHANGED since message %d]\n----END-OF-REPO-MAP----", 
            m.ContextState.RepoMap.LastChanged)
    }
    
    if content == "" {
        return ""
    }
    
    return fmt.Sprintf("----REPO-MAP----\n[UPDATED]\n%s\n----END-OF-REPO-MAP----", 
        strings.TrimSpace(content))
}

// buildFilesSection creates the files section with change detection
func (m *Manager) buildFilesSection() string {
    fileChanges := m.detectFileChanges()
    
    if len(fileChanges) == 0 {
        return ""
    }
    
    var fileEntries []string
    
    for filePath, change := range fileChanges {
        switch change.Status {
        case "NEW":
            fileEntries = append(fileEntries, 
                fmt.Sprintf("file: %s [NEW] (added: %s)", 
                    filePath, time.Now().Format("15:04:05")))
            // Don't include content here - just the reference
            
        case "UPDATED":
            fileEntries = append(fileEntries, 
                fmt.Sprintf("file: %s [UPDATED] (last modified: %s)", 
                    filePath, time.Now().Format("15:04:05")))
            // Don't include content here - just the reference
            
        case "UNCHANGED":
            fileEntries = append(fileEntries, 
                fmt.Sprintf("file: %s [UNCHANGED since message %d]", 
                    filePath, change.LastChanged))
                    
        case "REMOVED":
            fileEntries = append(fileEntries, 
                fmt.Sprintf("file: %s [REMOVED] (removed at: %s)", 
                    filePath, time.Now().Format("15:04:05")))
        }
    }
    
    return fmt.Sprintf("----FILES----\n%s\n----END-OF-FILES----", 
        strings.Join(fileEntries, "\n"))
}

// buildPanesSection creates the panes section with change detection
func (m *Manager) buildPanesSection() string {
    paneChanges := m.detectPaneChanges()
    
    if len(paneChanges) == 0 {
        return ""
    }
    
    var paneEntries []string
    
    for paneID, change := range paneChanges {
        switch change.Status {
        case "NEW":
            paneEntries = append(paneEntries, 
                fmt.Sprintf("pane: %s [NEW] (created: %s)\n%s", 
                    paneID, time.Now().Format("15:04:05"), change.Content))
                    
        case "UPDATED":
            paneEntries = append(paneEntries, 
                fmt.Sprintf("pane: %s [UPDATED] (last updated: %s)\n%s", 
                    paneID, time.Now().Format("15:04:05"), change.Content))
                    
        case "UNCHANGED":
            paneEntries = append(paneEntries, 
                fmt.Sprintf("pane: %s [UNCHANGED since message %d]", 
                    paneID, change.LastChanged))
                    
        case "REMOVED":
            paneEntries = append(paneEntries, 
                fmt.Sprintf("pane: %s [REMOVED] (session ended: %s)", 
                    paneID, time.Now().Format("15:04:05")))
        }
    }
    
    return fmt.Sprintf("----PANES----\n%s\n----END-OF-PANES----", 
        strings.Join(paneEntries, "\n"))
}
```

### **Step 4: Add Required Import**
Add this import to `internal/structured_message.go`:

```go
import (
    "fmt"
    "os"
    "strings"
    "time"
    
    "github.com/alvinunreal/tmuxai/system"
)
```

## ✅ **Success Criteria**

### **Verification Steps:**
1. **Compile Check**: Run `go build .` - should compile without errors
2. **Change Detection**: Verify unchanged content shows [UNCHANGED since message X]
3. **Update Detection**: Verify changed content shows [UPDATED]
4. **New Content**: Verify new files/panes show [NEW]
5. **Removal Detection**: Verify deleted files/ended sessions show [REMOVED]

### **Expected Results:**
- [ ] File compiles successfully
- [ ] Change detection functions are implemented
- [ ] New change info types are defined
- [ ] Section builders use change detection
- [ ] First message shows everything as [UPDATED] or [NEW]
- [ ] Subsequent messages show appropriate change markers
- [ ] Token usage is significantly reduced for unchanged content

### **Test Commands:**
```bash
cd /path/to/tmuxai
go build .
echo "✅ Change detection implemented successfully if no errors above"
```

### **Manual Test Sequence:**
1. **Start TmuxAI**: `./tmuxai --agentic`
2. **First message**: "What's in this project?" - should show [UPDATED] for all sections
3. **Second message**: "List files" - should show [UNCHANGED since message 1] for repo map
4. **Run command**: Change pane content, verify pane shows [UPDATED]
5. **Read file**: `<ReadFile>README.md</ReadFile>`, then ask another question - file should show [UNCHANGED]

### **Expected Token Reduction:**
- **Message 1**: Full context (~1000 tokens)
- **Message 2**: Reduced context (~200-300 tokens) 
- **Message 3+**: Minimal context (~50-100 tokens)

## 📝 **Notes**
- This is the core optimization that reduces token usage
- Change detection uses content hashing for accuracy
- Files section only shows references, not full content (content is in conversation history)
- Panes section shows full content when changed, reference when unchanged
- Removed items are tracked but not cleaned up immediately

## 🔗 **Next Task**
After this is complete, move to `06-implement-state-validation.md`