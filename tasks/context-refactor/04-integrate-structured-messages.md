# Task 4: Integrate Structured Messages into ProcessUserMessage

## 🎯 **Goal**
Replace the old context building in `ProcessUserMessage` with the new structured message format.

## 📋 **What to Achieve**
- Replace old context concatenation with structured message building
- Maintain all existing functionality
- Use new message format for AI communication
- Keep conversation history clean

## 🔧 **How to Achieve**

### **Step 1: Update ProcessUserMessage Function**
In `internal/process_message.go`, find the `ProcessUserMessage` function (around line 17) and replace the context building section.

**Find this code (around lines 33-44):**
```go
repoMapContext := m.getRepoMapContext()

currentTmuxWindow := m.GetTmuxPanesInXml(m.Config)
execPaneEnv := ""
if !m.ExecPane.IsSubShell {
    execPaneEnv = fmt.Sprintf("Keep in mind, you are working within the shell: %s and OS: %s", m.ExecPane.Shell, m.ExecPane.OS)
}
currentMessage := ChatMessage{
    Content:   repoMapContext + currentTmuxWindow + "\n\n" + execPaneEnv + "\n\n" + message,
    FromUser:  true,
    Timestamp: time.Now(),
}
```

**Replace it with:**
```go
// Build structured message with new format
structuredContent := m.BuildStructuredMessage(message)

currentMessage := ChatMessage{
    Content:   structuredContent,
    FromUser:  true,
    Timestamp: time.Now(),
}
```

### **Step 2: Update Message History Handling**
The conversation history is now handled within the structured message, so we need to adjust how we build the history array.

**Find this code (around lines 46-59):**
```go
// build current chat history
var history []ChatMessage
switch {
case m.WatchMode:
    history = []ChatMessage{m.watchPrompt()}
case m.GetAgenticMode():
    history = []ChatMessage{m.agenticPrompt()}
default:
    history = []ChatMessage{m.chatAssistantPrompt(m.ExecPane.IsPrepared)}
}

history = append(history, m.Messages...)

sending := append(history, currentMessage)
```

**Replace it with:**
```go
// build current chat history - only system prompt, conversation is in structured message
var history []ChatMessage
switch {
case m.WatchMode:
    history = []ChatMessage{m.watchPrompt()}
case m.GetAgenticMode():
    history = []ChatMessage{m.agenticPrompt()}
default:
    history = []ChatMessage{m.chatAssistantPrompt(m.ExecPane.IsPrepared)}
}

// Note: m.Messages is now included in the structured message conversation section
// So we only send system prompt + current structured message
sending := append(history, currentMessage)
```

### **Step 3: Update System Prompts to Explain New Format**
Add this method to `internal/structured_message.go`:

```go
// getStructuredFormatExplanation returns explanation for AI about the new format
func (m *Manager) getStructuredFormatExplanation() string {
    return `
IMPORTANT: You receive context in structured sections:

CURRENT-TIME: Shows current date and time
PROMPTS: Your instructions (always read this)
REPO-MAP: Project structure (check for [UPDATED] tag)
FILES: File contents you've read (check for [UPDATED] tag)
PANES: Current terminal panes (check for [UPDATED] tag)
OLD-SESSION-DATA: Restored session reference (if present)
CONVERSATION: Our chat history

When you see [UNCHANGED since message X], refer to that message's content.
When you see [UPDATED], focus on the new content.
When you see [REMOVED], that item no longer exists.
When you see [OLD SESSION], that's reference material from a restored session.
`
}
```

### **Step 4: Update System Prompts**
In `internal/prompts.go`, find the `baseSystemPrompt()` function and add the format explanation:

**Find the end of the baseSystemPrompt function (around line 41):**
```go
DO NOT WRITE MORE TEXT AFTER THE TOOL CALLS IN A RESPONSE. You can wait until the next response to summarize the actions you've done.
`
```

**Add before the closing backtick:**
```go
DO NOT WRITE MORE TEXT AFTER THE TOOL CALLS IN A RESPONSE. You can wait until the next response to summarize the actions you've done.

CONTEXT FORMAT:
You receive context in structured sections with clear delimiters. Pay attention to [UPDATED], [UNCHANGED], [REMOVED], and [OLD SESSION] markers to understand what has changed since previous messages.
`
```

### **Step 5: Add Debug Output (Optional)**
Add this temporary debug output to see the new format in action. In `internal/process_message.go`, after building the structured message:

```go
// Temporary debug output - remove after testing
if m.Config.Debug {
    logger.Debug("Structured message format:\n%s", structuredContent)
}
```

## ✅ **Success Criteria**

### **Verification Steps:**
1. **Compile Check**: Run `go build .` - should compile without errors
2. **Message Format**: Verify AI receives structured messages
3. **Functionality**: Verify all existing features still work
4. **Conversation**: Verify conversation history is preserved
5. **Sections**: Verify all sections appear in AI messages

### **Expected Results:**
- [ ] File compiles successfully
- [ ] `ProcessUserMessage` uses `BuildStructuredMessage()`
- [ ] Old context concatenation is removed
- [ ] AI receives messages in new structured format
- [ ] Conversation history is included in CONVERSATION section
- [ ] System prompts explain the new format
- [ ] All existing functionality works (commands, file reading, etc.)

### **Test Commands:**
```bash
cd /path/to/tmuxai
go build .
echo "✅ Structured messages integrated successfully if no errors above"

# Test with actual TmuxAI run
./tmuxai --agentic
# Send a test message and verify the format
```

### **Manual Verification:**
1. **Start TmuxAI**: `./tmuxai --agentic`
2. **Send test message**: "What's in this project?"
3. **Check debug output**: Look for structured sections in debug logs
4. **Verify AI response**: AI should understand and respond appropriately
5. **Test file reading**: Try `<ReadFile>README.md</ReadFile>` and verify it appears in FILES section

### **Expected Message Structure:**
```
----CURRENT-TIME----
Date: July 21, 2024 - Time: 14:35:22
----END-OF-CURRENT-TIME----

----PROMPTS----
[System prompts handled separately]
----END-OF-PROMPTS----

----REPO-MAP----
[UPDATED]
[repo map content]
----END-OF-REPO-MAP----

----PANES----
[UPDATED] (last updated: 14:35:20)
[pane content]
----END-OF-PANES----

----CONVERSATION----
User: What's in this project?
----END-OF-CONVERSATION----
```

## 📝 **Notes**
- This is a major change that affects core message processing
- Test thoroughly to ensure no functionality is broken
- The conversation history is now embedded in the structured message
- Debug output helps verify the new format is working
- System prompts now explain the format to the AI

## 🔗 **Next Task**
After this is complete, move to `05-implement-change-detection.md`