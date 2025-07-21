# Plan 06: Final Testing and Cleanup

**Objective:** Conduct thorough testing of the new context management system, remove any obsolete code, and ensure the implementation is robust and reliable.

---

## 🔧 **Technical Implementation**

### **1. Remove Obsolete Functions**

Now that the new system is in place, some old functions are no longer needed.

**File to Modify:** `internal/manager.go`

- **Remove `getRepoMapContext()`:** This function is now redundant. The repo map is fetched and updated within `ProcessUserMessage` as part of the new context tracking flow. Ensure it is completely removed from the `manager.go` file.

**File to Modify:** `internal/process_message.go`

- **Remove `GetTmuxPanesInXml()` call and related logic:** The logic that formats pane content into an XML-like string is now replaced by the `StructuredMessageBuilder`. The call to `m.GetTmuxPanesInXml(m.Config)` inside `ProcessUserMessage` should have already been removed during the refactoring in plan 04, but double-check that it and any related string concatenation are gone.

### **2. Code Cleanup**

Review the changes made in the previous steps and perform any necessary cleanup:

- **Remove commented-out code:** Delete any blocks of old code that were commented out during the refactoring process.
- **Ensure consistent formatting:** Run `go fmt ./...` to ensure all new and modified code adheres to Go's formatting standards.
- **Check for unused imports:** Make sure there are no unused imports in the modified files.

### **3. Comprehensive Testing**

This is the most critical part of this plan. Test the following scenarios thoroughly:

- **Initial Message:** Verify that the first message sends the full context with all sections marked as `[NEW]`.
- **No Changes:** Send a second message without making any changes to files or panes. Verify that the context sections are marked as `[UNCHANGED since message 1]`.
- **File Updates:** Modify a file that is part of the context. Send a message and verify that the file is marked as `[UPDATED]` and its new content is included.
- **Pane Updates:** Run a command in a pane. Send a message and verify that the pane is marked as `[UPDATED]` with its new content.
- **File Deletion:** Delete a file that was in the context. The system should ideally detect this (though full implementation of this might require further work on the tracker). For now, ensure it doesn't crash.
- **Long Conversations:** Have a long conversation to ensure the `[UNCHANGED since message X]` references work correctly over many turns.
- **Debug Logs:** Keep `debug: true` enabled and monitor `debug.log` to confirm the payloads are correct for each scenario.

---

## ✅ **Validation**

- The application is stable and performs as expected in all test scenarios.
- Token usage per message (after the first one) should be significantly reduced. You can verify this by observing the token counts printed in the console (if that feature is still present) or by checking the payloads in `debug.log`.
- The AI's responses should be consistent and contextually aware, correctly using the information from the structured messages.

## 🏆 **Project Completion**

Once all tests pass and the cleanup is complete, the context refactor project is finished. The new system should provide a more efficient and cost-effective experience.
