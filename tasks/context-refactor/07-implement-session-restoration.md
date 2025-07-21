# Task 7: Implement Session Restoration Handler

## 🎯 **Goal**
Add support for loading old session data via `/session` command and displaying it as reference material in the OLD-SESSION-DATA section.

## 📋 **What to Achieve**
- Handle `/session` command to load old session data
- Store old session data in ContextState.OldSession
- Display old session data as reference in structured messages
- Handle conflicts between current and old session pane/file IDs
- Preserve old conversation history

## 🔧 **How to Achieve**

### **Step 1: Create Session Restoration Functions**
Create a new file `internal/session_restoration.go`:

```go
package internal

import (
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"
    "strings"
    "time"
    
    "github.com/alvinunreal/tmuxai/logger"
)

// SessionMetadata represents metadata about a saved session
type SessionMetadata struct {
    SessionName string    `json:"session_name"`
    SavedAt     time.Time `json:"saved_at"`
    FilePath    string    `json:"file_path"`
    MessageCount int      `json:"message_count"`
    ProjectPath string    `json:"project_path"`
}

// loadSessionData loads session data from a session file
func (m *Manager) loadSessionData(sessionPath string) (*OldSessionData, error) {
    // Read session file
    data, err := os.ReadFile(sessionPath)
    if err != nil {
        return nil, fmt.Errorf("failed to read session file: %w", err)
    }
    
    // Parse session data (assuming it's stored as JSON)
    var sessionData struct {
        Messages    []ChatMessage `json:"messages"`
        ReadFiles   []string      `json:"read_files"`
        SessionPath string        `json:"session_path"`
        SavedAt     time.Time     `json:"saved_at"`
    }
    
    if err := json.Unmarshal(data, &sessionData); err != nil {
        return nil, fmt.Errorf("failed to parse session file: %w", err)
    }
    
    // Extract session name from file path
    sessionName := filepath.Base(sessionPath)
    sessionName = strings.TrimSuffix(sessionName, filepath.Ext(sessionName))
    
    // Convert to OldSessionData format
    oldSession := &OldSessionData{
        SessionName:  sessionName,
        SavedAt:      sessionData.SavedAt,
        Files:        make(map[string]SectionState),
        Panes:        make(map[string]SectionState),
        Conversation: sessionData.Messages,
    }
    
    // Process read files from old session
    for _, filePath := range sessionData.ReadFiles {
        // Create a placeholder state for old files
        oldSession.Files[filePath] = SectionState{
            Content:     "[File content from old session - use current version if needed]",
            LastChanged: 0, // Old session
            Hash:        "",
            Status:      StatusOldSession,
            Timestamp:   sessionData.SavedAt,
        }
    }
    
    // Extract pane information from conversation history
    m.extractPaneInfoFromConversation(oldSession)
    
    return oldSession, nil
}

// extractPaneInfoFromConversation extracts pane information from old conversation
func (m *Manager) extractPaneInfoFromConversation(oldSession *OldSessionData) {
    // Look through conversation for pane content
    // This is a simplified extraction - in reality, you might want more sophisticated parsing
    
    for _, msg := range oldSession.Conversation {
        if !msg.FromUser {
            continue // Only look at user messages which might contain pane context
        }
        
        // Look for pane references in the content
        content := msg.Content
        
        // Simple extraction: look for patterns like "pane: %1" or similar
        // This is a basic implementation - you might want more sophisticated parsing
        if strings.Contains(content, "pane:") || strings.Contains(content, "%") {
            // Extract potential pane content
            lines := strings.Split(content, "\n")
            var currentPaneID string
            var paneContent []string
            
            for _, line := range lines {
                if strings.Contains(line, "pane:") && strings.Contains(line, "%") {
                    // Save previous pane if exists
                    if currentPaneID != "" && len(paneContent) > 0 {
                        oldSession.Panes[currentPaneID] = SectionState{
                            Content:     strings.Join(paneContent, "\n"),
                            LastChanged: 0, // Old session
                            Hash:        "",
                            Status:      StatusOldSession,
                            Timestamp:   oldSession.SavedAt,
                        }
                    }
                    
                    // Start new pane
                    if strings.Contains(line, "%") {
                        parts := strings.Fields(line)
                        for _, part := range parts {
                            if strings.HasPrefix(part, "%") {
                                currentPaneID = part
                                paneContent = []string{}
                                break
                            }
                        }
                    }
                } else if currentPaneID != "" {
                    paneContent = append(paneContent, line)
                }
            }
            
            // Save last pane
            if currentPaneID != "" && len(paneContent) > 0 {
                oldSession.Panes[currentPaneID] = SectionState{
                    Content:     strings.Join(paneContent, "\n"),
                    LastChanged: 0, // Old session
                    Hash:        "",
                    Status:      StatusOldSession,
                    Timestamp:   oldSession.SavedAt,
                }
            }
        }
    }
}

// restoreSessionAsReference loads an old session as reference material
func (m *Manager) restoreSessionAsReference(sessionPath string) error {
    oldSession, err := m.loadSessionData(sessionPath)
    if err != nil {
        return fmt.Errorf("failed to load session data: %w", err)
    }
    
    // Store as reference in current context
    m.ContextState.OldSession = oldSession
    
    logger.Info("Loaded old session as reference: %s (saved: %s)", 
        oldSession.SessionName, oldSession.SavedAt.Format("2006-01-02 15:04:05"))
    
    m.Println(fmt.Sprintf("Loaded session '%s' as reference material (saved: %s)", 
        oldSession.SessionName, oldSession.SavedAt.Format("January 2, 2006 3:04 PM")))
    
    return nil
}

// clearOldSessionReference removes old session reference
func (m *Manager) clearOldSessionReference() {
    if m.ContextState.OldSession != nil {
        sessionName := m.ContextState.OldSession.SessionName
        m.ContextState.OldSession = nil
        m.Println(fmt.Sprintf("Cleared old session reference: %s", sessionName))
    }
}

// getAvailableSessions returns list of available session files
func (m *Manager) getAvailableSessions() ([]SessionMetadata, error) {
    // Look for session files in .tmuxai directory
    tmuxaiDir := ".tmuxai"
    if _, err := os.Stat(tmuxaiDir); os.IsNotExist(err) {
        return []SessionMetadata{}, nil
    }
    
    files, err := os.ReadDir(tmuxaiDir)
    if err != nil {
        return nil, fmt.Errorf("failed to read .tmuxai directory: %w", err)
    }
    
    var sessions []SessionMetadata
    
    for _, file := range files {
        if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
            continue
        }
        
        filePath := filepath.Join(tmuxaiDir, file.Name())
        
        // Get file info
        fileInfo, err := file.Info()
        if err != nil {
            continue
        }
        
        // Try to extract session metadata
        sessionName := strings.TrimSuffix(file.Name(), ".json")
        
        sessions = append(sessions, SessionMetadata{
            SessionName: sessionName,
            SavedAt:     fileInfo.ModTime(),
            FilePath:    filePath,
            ProjectPath: ".", // Current directory
        })
    }
    
    return sessions, nil
}

// handleSessionCommand processes /session command
func (m *Manager) handleSessionCommand(args []string) error {
    if len(args) == 0 {
        // List available sessions
        sessions, err := m.getAvailableSessions()
        if err != nil {
            return fmt.Errorf("failed to get available sessions: %w", err)
        }
        
        if len(sessions) == 0 {
            m.Println("No saved sessions found in .tmuxai directory")
            return nil
        }
        
        m.Println("Available sessions:")
        for i, session := range sessions {
            m.Println(fmt.Sprintf("%d. %s (saved: %s)", 
                i+1, session.SessionName, session.SavedAt.Format("January 2, 2006 3:04 PM")))
        }
        
        m.Println("\nUse '/session <number>' to load a session as reference")
        m.Println("Use '/session clear' to clear current session reference")
        
        return nil
    }
    
    // Handle specific commands
    command := args[0]
    
    switch command {
    case "clear":
        m.clearOldSessionReference()
        return nil
        
    default:
        // Try to parse as session number or name
        sessions, err := m.getAvailableSessions()
        if err != nil {
            return fmt.Errorf("failed to get available sessions: %w", err)
        }
        
        if len(sessions) == 0 {
            return fmt.Errorf("no sessions available")
        }
        
        // Try to parse as number
        var selectedSession *SessionMetadata
        
        // Simple number parsing
        if len(command) == 1 && command >= "1" && command <= "9" {
            sessionNum := int(command[0] - '0')
            if sessionNum > 0 && sessionNum <= len(sessions) {
                selectedSession = &sessions[sessionNum-1]
            }
        }
        
        // Try to find by name
        if selectedSession == nil {
            for _, session := range sessions {
                if session.SessionName == command {
                    selectedSession = &session
                    break
                }
            }
        }
        
        if selectedSession == nil {
            return fmt.Errorf("session not found: %s", command)
        }
        
        // Load the session
        return m.restoreSessionAsReference(selectedSession.FilePath)
    }
}
```

### **Step 2: Update Chat Command Handler**
In `internal/chat_command.go`, find the command handling section and add session command support.

**Find the command switch statement and add:**
```go
case "session":
    args := []string{}
    if len(parts) > 1 {
        args = parts[1:]
    }
    
    if err := m.handleSessionCommand(args); err != nil {
        m.Println(fmt.Sprintf("Session command error: %v", err))
    }
    return true
```

### **Step 3: Update buildOldSessionSection**
Replace the existing `buildOldSessionSection` in `internal/structured_message.go`:

```go
// buildOldSessionSection creates the old session section (if exists)
func (m *Manager) buildOldSessionSection() string {
    if m.ContextState.OldSession == nil {
        return ""
    }
    
    oldSession := m.ContextState.OldSession
    var sessionEntries []string
    
    sessionEntries = append(sessionEntries, 
        fmt.Sprintf("[Restored from: \"%s\" - saved: %s]", 
            oldSession.SessionName, 
            oldSession.SavedAt.Format("January 2, 2006 3:04 PM")))
    
    // Add old files (if any)
    if len(oldSession.Files) > 0 {
        sessionEntries = append(sessionEntries, "\n[OLD SESSION FILES]")
        for filePath, state := range oldSession.Files {
            sessionEntries = append(sessionEntries, 
                fmt.Sprintf("file: %s [OLD SESSION] (from: %s)", 
                    filePath, 
                    state.Timestamp.Format("January 2, 3:04 PM")))
        }
    }
    
    // Add old panes (if any)
    if len(oldSession.Panes) > 0 {
        sessionEntries = append(sessionEntries, "\n[OLD SESSION PANES]")
        for paneID, state := range oldSession.Panes {
            sessionEntries = append(sessionEntries, 
                fmt.Sprintf("pane: %s [OLD SESSION] (from: %s)", 
                    paneID, 
                    state.Timestamp.Format("January 2, 3:04 PM")))
            
            // Include a preview of the content (first few lines)
            lines := strings.Split(state.Content, "\n")
            previewLines := 3
            if len(lines) > previewLines {
                preview := strings.Join(lines[:previewLines], "\n")
                sessionEntries = append(sessionEntries, fmt.Sprintf("%s\n[... %d more lines]", preview, len(lines)-previewLines))
            } else {
                sessionEntries = append(sessionEntries, state.Content)
            }
        }
    }
    
    // Add conversation history summary
    if len(oldSession.Conversation) > 0 {
        sessionEntries = append(sessionEntries, "\n[OLD CONVERSATION HISTORY]")
        
        // Show last few messages as summary
        maxMessages := 5
        startIdx := 0
        if len(oldSession.Conversation) > maxMessages {
            startIdx = len(oldSession.Conversation) - maxMessages
            sessionEntries = append(sessionEntries, fmt.Sprintf("[... %d earlier messages]", startIdx))
        }
        
        for i := startIdx; i < len(oldSession.Conversation); i++ {
            msg := oldSession.Conversation[i]
            if msg.FromUser {
                sessionEntries = append(sessionEntries, fmt.Sprintf("User: %s", msg.Content))
            } else {
                // Truncate long AI responses
                content := msg.Content
                if len(content) > 200 {
                    content = content[:200] + "..."
                }
                sessionEntries = append(sessionEntries, fmt.Sprintf("AI: %s", content))
            }
        }
    }
    
    return fmt.Sprintf("----OLD-SESSION-DATA----\n%s\n----END-OF-OLD-SESSION-DATA----", 
        strings.Join(sessionEntries, "\n"))
}
```

### **Step 4: Add Session Command Help**
In `internal/chat_command.go`, update the help command to include session commands:

**Find the help text and add:**
```go
/session                 - List available sessions
/session <number>        - Load session as reference material  
/session <name>          - Load session by name as reference
/session clear           - Clear current session reference
```

## ✅ **Success Criteria**

### **Verification Steps:**
1. **Compile Check**: Run `go build .` - should compile without errors
2. **List Sessions**: `/session` command shows available sessions
3. **Load Session**: `/session 1` loads a session as reference
4. **Display Reference**: Old session appears in OLD-SESSION-DATA section
5. **Clear Reference**: `/session clear` removes old session data
6. **Conflict Handling**: Same pane IDs in current and old sessions are handled properly

### **Expected Results:**
- [ ] File compiles successfully
- [ ] `/session` command lists available sessions
- [ ] `/session <number>` loads session as reference
- [ ] Old session data appears in structured messages
- [ ] Current session remains primary, old session is reference only
- [ ] Pane/file ID conflicts are handled gracefully
- [ ] Old conversation history is preserved and displayed
- [ ] Session clearing works properly

### **Test Commands:**
```bash
cd /path/to/tmuxai
go build .
echo "✅ Session restoration implemented successfully if no errors above"
```

### **Manual Test Sequence:**
1. **Start TmuxAI**: `./tmuxai --agentic`
2. **List sessions**: `/session` - should show available sessions
3. **Load session**: `/session 1` - should load first session
4. **Check structure**: Send a message, verify OLD-SESSION-DATA section appears
5. **Clear session**: `/session clear` - should remove old session reference
6. **Verify clearing**: Send another message, verify OLD-SESSION-DATA section is gone

### **Expected Output Structure:**
```
----OLD-SESSION-DATA----
[Restored from: "feature-work" - saved: July 19, 2024 12:45 PM]

[OLD SESSION FILES]
file: main.go [OLD SESSION] (from: July 19, 12:45 PM)

[OLD SESSION PANES]  
pane: %1 [OLD SESSION] (from: July 19, 12:45 PM)
$ npm test
> test
[... 5 more lines]

[OLD CONVERSATION HISTORY]
User: Run the tests
AI: I'll run the test suite...
----END-OF-OLD-SESSION-DATA----
```

## 📝 **Notes**
- Old sessions are reference material only, not active context
- Current session always takes precedence
- Pane/file ID conflicts are resolved by marking old items as [OLD SESSION]
- Conversation history from old sessions provides context
- Session restoration doesn't affect current tmux session

## 🔗 **Next Task**
After this is complete, move to `08-testing-and-optimization.md`