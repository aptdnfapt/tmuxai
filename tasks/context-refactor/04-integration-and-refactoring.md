# Plan 04: Integration and Refactoring

**Objective:** Integrate the new context tracking and message building components into the main message processing loop. This involves refactoring `ProcessUserMessage` in `internal/process_message.go` to use the new system.

---

## 🔧 **Technical Implementation**

### **1. File to Modify:** `internal/process_message.go`

### **2. Refactor `ProcessUserMessage`**

The goal is to replace the current manual context string concatenation with calls to the new `ContextStateTracker` and `StructuredMessageBuilder`.

**Current (Old) Logic to be Replaced:**

```go
// This entire block will be replaced.
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

**New Logic:**

```go
// Add this new logic at the beginning of ProcessUserMessage.

	// 1. Increment message count for the new turn.
	m.ContextTracker.IncrementMessageCount()

	// 2. Update context state from all sources.
	// Update Repo Map
	repoMapContext, _ := m.RepoMap.GetMap()
	m.ContextTracker.UpdateRepoMap(repoMapContext)

	// Update Panes
	panes, _ := m.GetTmuxPanes()
	for _, pane := range panes {
		paneContent, _ := system.TmuxCapturePane(pane.Id, m.GetMaxCaptureLines())
		m.ContextTracker.UpdatePane(pane.Id, paneContent)
	}

	// Update Read Files (a placeholder for now, will be fully implemented later)
	for _, filePath := range m.ReadFiles {
		content, err := os.ReadFile(filePath)
		if err == nil {
			m.ContextTracker.UpdateFile(filePath, string(content))
		}
	}

	// 3. Build the new structured message.
	structuredMessage := m.MessageBuilder.BuildMessage(message)

	// 4. Create the ChatMessage with the new structured content.
	currentMessage := ChatMessage{
		Content:   structuredMessage,
		FromUser:  true,
		Timestamp: time.Now(),
	}

	// The rest of the function (history building, AI call, response parsing) remains the same for now.
```

### **3. Remove Redundant `getRepoMapContext` Call**

Since the repo map is now handled by the `ContextStateTracker`, the initial call to `m.getRepoMapContext()` at the top of `ProcessUserMessage` should be removed as it's now redundant.

---

## ✅ **Validation**

- After refactoring, the application should compile and run (`go build . && ./tmuxai`).
- The messages sent to the AI should now be in the new structured format.
- **Crucially**, you will need to check the `debug.log` file (with `debug: true` in your config) to verify that the payload sent to the AI matches the new structured format.
- The AI's behavior might change due to the new format. This is expected and will be addressed in the next step by updating the system prompts.

## ➡️ **Next Step**

Proceed to `05-system-prompt-updates.md` to modify the system prompts to instruct the AI on how to understand and use the new structured message format.

```