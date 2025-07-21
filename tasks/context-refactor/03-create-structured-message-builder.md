# Task 3: Create Structured Message Builder

## 🎯 **Goal**
Create a new function to build structured messages with the new format including timestamps and sections.

## 📋 **What to Achieve**
- Create `buildStructuredMessage()` function
- Implement the new message format with sections
- Add current time section
- Add basic section structure (without change detection yet)

## 🔧 **How to Achieve**

### **Step 1: Create New File**
Create a new file `internal/structured_message.go` with this content:

```go
package internal

import (
    "fmt"
    "strings"
    "time"
)

// buildStructuredMessage creates a message in the new structured format
func (m *Manager) buildStructuredMessage(userInput string) string {
    var sections []string
    
    // Current time section
    currentTime := time.Now()
    timeSection := fmt.Sprintf("----CURRENT-TIME----\nDate: %s - Time: %s\n----END-OF-CURRENT-TIME----",
        currentTime.Format("January 2, 2006"),
        currentTime.Format("15:04:05"))
    sections = append(sections, timeSection)
    
    // Prompts section (always include system prompts)
    promptsSection := m.buildPromptsSection()
    sections = append(sections, promptsSection)
    
    // Repo map section
    repoMapSection := m.buildRepoMapSection()
    if repoMapSection != "" {
        sections = append(sections, repoMapSection)
    }
    
    // Files section
    filesSection := m.buildFilesSection()
    if filesSection != "" {
        sections = append(sections, filesSection)
    }
    
    // Panes section
    panesSection := m.buildPanesSection()
    if panesSection != "" {
        sections = append(sections, panesSection)
    }
    
    // Old session data section (if exists)
    oldSessionSection := m.buildOldSessionSection()
    if oldSessionSection != "" {
        sections = append(sections, oldSessionSection)
    }
    
    // Conversation section
    conversationSection := m.buildConversationSection(userInput)
    sections = append(sections, conversationSection)
    
    return strings.Join(sections, "\n\n")
}

// buildPromptsSection creates the prompts section
func (m *Manager) buildPromptsSection() string {
    // For now, just indicate prompts are handled separately
    // The actual prompt will be sent as system message
    return "----PROMPTS----\n[System prompts handled separately]\n----END-OF-PROMPTS----"
}

// buildRepoMapSection creates the repo map section
func (m *Manager) buildRepoMapSection() string {
    repoMapContext := m.getRepoMapContext()
    if repoMapContext == "" {
        return ""
    }
    
    return fmt.Sprintf("----REPO-MAP----\n[UPDATED]\n%s\n----END-OF-REPO-MAP----", 
        strings.TrimSpace(repoMapContext))
}

// buildFilesSection creates the files section
func (m *Manager) buildFilesSection() string {
    if len(m.ReadFiles) == 0 {
        return ""
    }
    
    var fileEntries []string
    for _, filePath := range m.ReadFiles {
        // For now, mark all as unchanged - change detection comes later
        fileEntries = append(fileEntries, 
            fmt.Sprintf("file: %s [UNCHANGED since message 1]", filePath))
    }
    
    if len(fileEntries) == 0 {
        return ""
    }
    
    return fmt.Sprintf("----FILES----\n%s\n----END-OF-FILES----", 
        strings.Join(fileEntries, "\n"))
}

// buildPanesSection creates the panes section
func (m *Manager) buildPanesSection() string {
    currentTmuxWindow := m.GetTmuxPanesInXml(m.Config)
    if currentTmuxWindow == "" {
        return ""
    }
    
    return fmt.Sprintf("----PANES----\n[UPDATED] (last updated: %s)\n%s\n----END-OF-PANES----",
        time.Now().Format("15:04:05"),
        strings.TrimSpace(currentTmuxWindow))
}

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
    
    // Add old panes
    for paneID, state := range oldSession.Panes {
        sessionEntries = append(sessionEntries, 
            fmt.Sprintf("pane: %s [OLD SESSION] (from: %s)\n%s", 
                paneID, 
                state.Timestamp.Format("January 2, 3:04 PM"),
                state.Content))
    }
    
    // Add old conversation history
    if len(oldSession.Conversation) > 0 {
        sessionEntries = append(sessionEntries, "\n[OLD CONVERSATION HISTORY]")
        for _, msg := range oldSession.Conversation {
            if msg.FromUser {
                sessionEntries = append(sessionEntries, fmt.Sprintf("User: %s", msg.Content))
            } else {
                sessionEntries = append(sessionEntries, fmt.Sprintf("AI: %s", msg.Content))
            }
        }
    }
    
    return fmt.Sprintf("----OLD-SESSION-DATA----\n%s\n----END-OF-OLD-SESSION-DATA----", 
        strings.Join(sessionEntries, "\n"))
}

// buildConversationSection creates the conversation section
func (m *Manager) buildConversationSection(userInput string) string {
    var conversationEntries []string
    
    // Add existing conversation history
    for _, msg := range m.Messages {
        if msg.FromUser {
            conversationEntries = append(conversationEntries, fmt.Sprintf("User: %s", msg.Content))
        } else {
            conversationEntries = append(conversationEntries, fmt.Sprintf("AI: %s", msg.Content))
        }
    }
    
    // Add current user input
    conversationEntries = append(conversationEntries, fmt.Sprintf("User: %s", userInput))
    
    return fmt.Sprintf("----CONVERSATION----\n%s\n----END-OF-CONVERSATION----", 
        strings.Join(conversationEntries, "\n"))
}
```

### **Step 2: Add Helper Method to Manager**
Add this method to `internal/manager.go`:

```go
// BuildStructuredMessage creates a structured message for the AI
func (m *Manager) BuildStructuredMessage(userInput string) string {
    m.IncrementMessageCounter()
    return m.buildStructuredMessage(userInput)
}
```

## ✅ **Success Criteria**

### **Verification Steps:**
1. **Compile Check**: Run `go build .` - should compile without errors
2. **File Creation**: Verify `internal/structured_message.go` exists
3. **Method Access**: Verify `BuildStructuredMessage()` is accessible from Manager
4. **Format Check**: Verify the output has the correct section structure

### **Expected Results:**
- [ ] File compiles successfully
- [ ] New file `internal/structured_message.go` is created
- [ ] `buildStructuredMessage()` function is defined
- [ ] All section builder functions are defined
- [ ] `BuildStructuredMessage()` method is added to Manager
- [ ] Output includes all required sections with proper delimiters

### **Test Command:**
```bash
cd /path/to/tmuxai
go build .
echo "✅ Structured message builder created successfully if no errors above"
```

### **Manual Test:**
You can add this temporary test to verify the structure:
```go
// Temporary test in main.go - remove after verification
manager, _ := NewManager(cfg, false)
testMessage := manager.BuildStructuredMessage("test input")
fmt.Println("Generated message structure:")
fmt.Println(testMessage)

// Check for required sections
requiredSections := []string{
    "----CURRENT-TIME----",
    "----PROMPTS----", 
    "----CONVERSATION----"
}
for _, section := range requiredSections {
    if !strings.Contains(testMessage, section) {
        log.Fatalf("Missing section: %s", section)
    }
}
fmt.Println("✅ All required sections present")
```

## 📝 **Notes**
- This creates the basic structure without change detection
- Change detection will be added in subsequent tasks
- The prompts section is placeholder - actual prompts handled separately
- Files and panes sections show basic format but no smart change tracking yet
- Old session section handles restored session data

## 🔗 **Next Task**
After this is complete, move to `04-integrate-structured-messages.md`