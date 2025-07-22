# Plan 05: System Prompt Updates

**Objective:** Update the system prompts in `internal/prompts.go` to explain the new structured message format to the AI. This is crucial for the AI to understand and correctly interpret the time-stamped, state-aware context.

---

## 🔧 **Technical Implementation**

### **1. File to Modify:** `internal/prompts.go`

### **2. Update `baseSystemPrompt()`**

Modify the `baseSystemPrompt` function to include a detailed explanation of the new structured format. This explanation should be added to the existing base prompt.

**New Text to Add to the Prompt:**

```text

==== STRUCTURED CONTEXT FORMAT ====
You will receive context in a structured format with headers and status markers. Here is how to interpret it:

- **`----SECTION----` / `----END-OF-SECTION----`**: These delimit a context section (e.g., REPO-MAP, FILES, PANES).
- **`[UPDATED]`**: This section or item has new content since the last message.
- **`[UNCHANGED since message X]`**: The content for this section or item has not changed since message number X. You should refer to your memory of that message.
- **`[REMOVED]`**: This item (e.g., a file or pane) has been deleted or closed.
- **`[NEW]`**: This is the first time you are seeing this item.
- **Timestamps**: Pay attention to the `(last modified: ...)` and `(last updated: ...)` timestamps to understand the timeline of events.
- **`----CURRENT-TIME----`**: This section at the top of each message tells you the current time of the user's system.
- **`----NEW-CONTENT----`**: In an `[UPDATED]` pane, this block contains **only** the new lines that have appeared since the last message. The previous content is not repeated.
- **`----OLD-SESSION-DATA----`**: If you are in a restored session, this section provides the context (panes and conversation) from the previously saved state.

Your task is to use this structured information to maintain a coherent understanding of the user's environment over time, without needing the full context repeated in every message. By referencing `[UNCHANGED]` markers, you can reduce redundant processing and focus only on what's new or `[UPDATED]`.

```

**Implementation:**

In `baseSystemPrompt()`, append this new text to the `basePrompt` string.

```go
func (m *Manager) baseSystemPrompt() string {
	basePrompt := `... existing prompt text ...`

    // Add the new explanation here.
    structuredContextExplanation := `
==== STRUCTURED CONTEXT FORMAT ====
... (paste the text from above) ...
`
    basePrompt += structuredContextExplanation

	if m.Config.Prompts.BaseSystem != "" {
		basePrompt = m.Config.Prompts.BaseSystem
	}
	return basePrompt
}
```

---

## ✅ **Validation**

- The application should compile and run.
- With the updated prompt, the AI should now be able to understand the structured messages. 
- Observe the AI's responses in a conversation. It should start to acknowledge the new format, for example, by not asking for the content of a file that is marked as `[UNCHANGED]`.
- Check `debug.log` to ensure the new prompt is being included in the payload sent to the AI.

## ➡️ **Next Step**

Proceed to `06-final-testing-and-cleanup.md` to conduct final testing, remove any obsolete code, and ensure the new system is working reliably.
