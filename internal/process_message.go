package internal

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/alvinunreal/tmuxai/logger"
	"github.com/alvinunreal/tmuxai/system"
	"github.com/briandowns/spinner"
)

// Main function to process regular user messages
// Returns true if the request was accomplished and no further processing should happen
func (m *Manager) ProcessUserMessage(ctx context.Context, message string) bool {
	// Check if context management is needed before sending
	if m.needSquash() {
		m.Println("Exceeded context size, squashing history...")
		m.squashHistory()
	}

	s := spinner.New(spinner.CharSets[26], 100*time.Millisecond)
	s.Start()

	// check for status change before processing
	if m.Status == "" {
		s.Stop()
		return false
	}

	// 2. Update context state from all sources.
	// Update Repo Map
	m.RepoMap.GetMap()

	// Update Panes
	panes, _ := m.GetTmuxPanes()
	for _, pane := range panes {
		if pane.IsTmuxAiPane {
			continue
		}
		system.TmuxCapturePane(pane.Id, m.GetMaxCaptureLines())
	}

	// Update Read Files (a placeholder for now, will be fully implemented later)
	for _, filePath := range m.ReadFiles {
		_, err := os.ReadFile(filePath)
		if err == nil {
			// File content is now read fresh each time in the message builder
		}
	}

	// 3. Build the new structured message for sending to AI.
	structuredMessage := m.MessageBuilder.BuildMessage(message)

	// 4. Create the ChatMessage with the full structured content for sending to AI
	currentMessage := ChatMessage{
		Content:   structuredMessage, // Send the full structured message to AI
		FromUser:  true,
		Timestamp: time.Now(),
	}

	// For conversation history, we'll store only the actual user input
	historyUserMessage := ChatMessage{
		Content:   message, // Store only the actual user input
		FromUser:  true,
		Timestamp: time.Now(),
	}

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

	// Use currentMessage (with full structured content) for sending to AI
	sending := append(history, currentMessage)

	response, err := m.AiClient.GetResponseFromChatMessages(ctx, sending, m.GetOpenRouterModel())
	if err != nil {
		s.Stop()
		m.Status = ""

		if ctx.Err() == context.Canceled {
			return false
		}

		// Log both to console and debug file to capture error context
		errMsg := "Failed to get response from AI: " + err.Error()
		fmt.Println(errMsg)

		// Debug the failed request even when there's an error
		if m.Config.Debug {
			debugChatMessages(append(history, currentMessage), "ERROR: "+err.Error(), m.Config)
		}

		return false
	}

	// check for status change again
	if m.Status == "" {
		s.Stop()
		return false
	}

	r, err := m.parseAIResponse(response)
	if err != nil {
		s.Stop()
		m.Status = ""

		// Log both to console and debug file
		errMsg := "Failed to parse AI response: " + err.Error()
		fmt.Println(errMsg)

		// Debug the failed parsing even when there's an error
		if m.Config.Debug {
			debugChatMessages(append(history, currentMessage), "PARSE ERROR: "+response, m.Config)
		}

		return false
	}

	if m.Config.Debug {
		debugChatMessages(append(history, currentMessage), response, m.Config)
	}

	logger.Debug("AIResponse: %s", r.String())

	s.Stop()

	// We'll create response messages inline where needed, no need to create it here

	if r.CreateExecPane {
		m.CreateNewExecPane()
	}

	// did AI follow our guidelines?
	guidelineError, validResponse := m.aiFollowedGuidelines(r)
	if !validResponse {
		m.Println("AI didn't follow guidelines, trying again...")
		// Store only the actual user input and AI message, not the full structured content
		aiMsg := ChatMessage{
			Content:   r.Message,
			FromUser:  false,
			Timestamp: time.Now(),
		}
		m.Messages = append(m.Messages, historyUserMessage, aiMsg)
		return m.ProcessUserMessage(ctx, guidelineError)

	}

	// colorize code blocks in the response
	if r.Message != "" {
		fmt.Println(m.GetAIChatPrompt() + system.Cosmetics(r.Message))
		m.PrintContextUsage()
	}

	// observe/prepared mode
	for _, execCommand := range r.ExecCommand {
		var targetPane *system.TmuxPaneDetails
		if execCommand.PaneID != "" {
			// Find the pane details for this ID.
			panes, _ := m.GetTmuxPanes()
			found := false

			// Normalize the pane ID from the AI, which might be missing or have extra '%' prefixes.
			// This handles cases like '%2', '%%2', or just '2'.
			normalizedPaneID := "%" + strings.TrimLeft(execCommand.PaneID, "%")

			for i, p := range panes {
				if p.Id == normalizedPaneID { // Compare against the normalized ID
					targetPane = &panes[i]
					found = true
					break
				}
			}
			if !found {
				// List available panes to help user understand what panes are available
				availablePanes := []string{}
				for _, p := range panes {
					if m.GetAgenticMode() || p.IsTmuxAiExecPane {
						availablePanes = append(availablePanes, p.Id)
					}
				}
				m.Println(fmt.Sprintf("Error: Could not find target pane with ID %s. Available panes: %s", execCommand.PaneID, strings.Join(availablePanes, ", ")))
				continue
			}
		} else {
			// Default to the primary exec pane
			targetPane = m.ExecPane
		}

		code, _ := system.HighlightCode("sh", execCommand.Command)
		m.Println(code)

		isSafe := false
		command := execCommand.Command
		confirmPrompt := "Execute this command?"
		if m.GetAgenticMode() {
			confirmPrompt = fmt.Sprintf("Execute this command in pane %s?", targetPane.Id)
		}

		if m.GetExecConfirm() {
			isSafe, command = m.confirmedToExec(execCommand.Command, confirmPrompt, true)
		} else {
			isSafe = true
		}
		if isSafe {
			m.Println(fmt.Sprintf("Executing in pane %s: %s", targetPane.Id, command))
			m.LastExecPaneID = targetPane.Id

			targetPane.Refresh(m.GetMaxCaptureLines())
			const endMarkerPrefix = "tmuxai waiting for command id"
			originalCommand := command
			commandToRun := command
			shouldWait := false

			// Agentic mode: AI decides whether to wait
			if m.GetAgenticMode() {
				shouldWait = execCommand.Wait
			} else { // Normal mode: check if the pane is prepared.
				if targetPane.IsPrepared {
					shouldWait = true
				}
			}

			if shouldWait {
				commandID := fmt.Sprintf("%05d", rand.Intn(100000))
				// A synchronous command was requested. First, add the history for the *current* turn.
				if !r.ExecPaneSeemsBusy && !r.NoComment {
					// Store only the actual user input and AI message, not the full structured content
					aiMsg := ChatMessage{
						Content:   r.Message,
						FromUser:  false,
						Timestamp: time.Now(),
					}
					m.Messages = append(m.Messages, historyUserMessage, aiMsg)
				}

				var exitCodeVar string
				if targetPane.Shell == "fish" {
					exitCodeVar = "$status"
				} else {
					exitCodeVar = "$?"
				}
				markerCommand := fmt.Sprintf(`; echo "%s: %s exitcode:%s"`, endMarkerPrefix, commandID, exitCodeVar)
				commandToRun += markerCommand

				// Execute the command and wait for it to complete.
				system.TmuxSendCommandToPane(targetPane.Id, commandToRun, true)
				result, err := m.ExecWaitCapture(targetPane, commandID)
				if err != nil {
					m.Println(fmt.Sprintf("Command cancelled or failed to wait: %v", err))
					m.Status = ""
					return false
				}
				result.Command = originalCommand // Fill in the command
				m.ExecHistory = append(m.ExecHistory, result)
				logger.Debug("Synchronous command finished. Code: %d", result.Code)

				// Now that the command is done, start the next turn by re-processing with the updated context.
				accomplished := m.ProcessUserMessage(ctx, "Ok, that command finished. Here is the updated pane content, what is the next step?")
				return accomplished // Return immediately to prevent any further processing of the stale AI response.
			} else {
				// Fire-and-forget for unprepared panes or agentic commands without the marker.
				system.TmuxSendCommandToPane(targetPane.Id, commandToRun, true)
				time.Sleep(1 * time.Second)
			}
		} else {
			m.Status = ""
			return false
		}
	}

	// Process SendKeys
	if len(r.SendKeys) > 0 {
		// Group keys by pane for confirmation
		keysByPane := make(map[string][]string)
		paneOrder := []string{} // Preserve order
		for _, sk := range r.SendKeys {
			paneID := sk.PaneID
			if paneID == "" {
				paneID = m.ExecPane.Id // Default to primary exec pane
			}
			if _, exists := keysByPane[paneID]; !exists {
				paneOrder = append(paneOrder, paneID)
			}
			keysByPane[paneID] = append(keysByPane[paneID], sk.Keys)
		}

		// Confirm and execute for each pane
		for _, paneID := range paneOrder {
			keys := keysByPane[paneID]

			// Normalize the pane ID from the AI, which might have extra '%' prefixes.
			// This handles cases like '%2' or '%%2'.
			normalizedPaneID := "%" + strings.TrimLeft(paneID, "%")

			keysPreview := fmt.Sprintf("Keys to send to pane %s:\n", paneID)
			for i, key := range keys {
				code, _ := system.HighlightCode("txt", key)
				if i == len(keys)-1 {
					keysPreview += code
				} else {
					keysPreview += code + "\n"
				}
			}
			m.Println(keysPreview)

			confirmMessage := fmt.Sprintf("Send these keys to pane %s?", paneID)
			if len(keys) == 1 {
				confirmMessage = fmt.Sprintf("Send this key to pane %s?", paneID)
			}

			allConfirmed := true
			if m.GetSendKeysConfirm() {
				allConfirmed, _ = m.confirmedToExec("keys shown above", confirmMessage, false) // No edit for keys
				if !allConfirmed {
					m.Status = ""
					return false // Abort all further actions
				}
			}

			// Send each key with delay
			for _, sendKey := range keys {
				m.Println(fmt.Sprintf("Sending to %s: %s", paneID, sendKey))
				system.TmuxSendCommandToPane(normalizedPaneID, sendKey, false)
				time.Sleep(1 * time.Second)
			}
		}
	}

	// This block handles state changes and asynchronous actions.
	// Synchronous actions (waitable ExecCommand) have already returned.

	// Process PasteMultilineContent
	if len(r.PasteMultilineContent) > 0 {
		for _, pc := range r.PasteMultilineContent {
			targetPaneID := pc.PaneID
			if targetPaneID == "" {
				targetPaneID = m.ExecPane.Id // Default to primary exec pane
			}

			// Normalize the pane ID from the AI before using it.
			// This handles cases like '%2' or '%%2'.
			normalizedPaneID := "%" + strings.TrimLeft(targetPaneID, "%")

			code, _ := system.HighlightCode("txt", pc.Content)
			m.Println(fmt.Sprintf("Content to paste into pane %s:", targetPaneID))
			fmt.Println(code)

			isSafe := false
			if m.GetPasteMultilineConfirm() {
				isSafe, _ = m.confirmedToExec(pc.Content, fmt.Sprintf("Paste this content into pane %s?", targetPaneID), false)
			} else {
				isSafe = true
			}

			if isSafe {
				m.Println("Pasting...")
				system.TmuxPasteToPane(normalizedPaneID, pc.Content)
			} else {
				m.Status = ""
				return false
			}
		}
	}

	// Process ReadFile requests
	if len(r.ReadFile) > 0 {
		// A ReadFile request is synchronous and requires a new turn.
		if !r.ExecPaneSeemsBusy && !r.NoComment {
			// Store only the actual user input and AI message, not the full structured content
			aiMsg := ChatMessage{
				Content:   r.Message,
				FromUser:  false,
				Timestamp: time.Now(),
			}
			m.Messages = append(m.Messages, historyUserMessage, aiMsg)
		}

		// 1. Validate all files first
		type validFile struct {
			Info    ReadFileInfo
			OsInfo  os.FileInfo
			AbsPath string
		}
		var filesToRead []validFile
		var totalBytes int64
		var fileListForPrompt []string

		filesToProcess := r.ReadFile
		if len(filesToProcess) > 1 && !m.GetMultiFileRead() {
			m.Println("AI requested to read multiple files, but 'multi_file_read' is disabled. Reading only the first file.")
			filesToProcess = filesToProcess[:1]
		}

		for _, fileInfo := range filesToProcess {
			osInfo, absPath, err := m.validateReadFile(fileInfo.FilePath)
			if err != nil {
				m.Println(fmt.Sprintf("Skipping file %s: %v", fileInfo.FilePath, err))
				continue
			}
			filesToRead = append(filesToRead, validFile{Info: fileInfo, OsInfo: osInfo, AbsPath: absPath})
			totalBytes += osInfo.Size()
			fileListForPrompt = append(fileListForPrompt, fmt.Sprintf("%s (%d bytes)", fileInfo.FilePath, osInfo.Size()))
		}

		if len(filesToRead) == 0 {
			m.Println("No valid files found to read.")
			return false // No files read, so no new turn.
		}

		// 2. Ask for confirmation for the batch
		if m.GetReadFileConfirm() {
			fmt.Printf("Read %d file(s)? (%d bytes total)\n", len(filesToRead), totalBytes)
			for _, fileLine := range fileListForPrompt {
				fmt.Printf(" - %s\n", fileLine)
			}
			confirmed, _ := m.confirmedToExec("", "Read files?", false)
			if !confirmed {
				m.Println("File reading cancelled by user.")
				return false
			}
		}

		// 3. Read confirmed files and update context tracker
		var fileNamesForPrompt []string
		filesAdded := 0
		for _, file := range filesToRead {
			content, err := os.ReadFile(file.AbsPath)
			if err != nil {
				m.Println(fmt.Sprintf("Error reading file %s: %v", file.Info.FilePath, err))
				continue
			}
			logger.Info("Read file: %s (%d bytes)", file.AbsPath, len(content))

			// Add to session's read file list if not already there
			isAlreadyRead := false
			for _, path := range m.ReadFiles {
				if path == file.AbsPath {
					isAlreadyRead = true
					break
				}
			}
			if !isAlreadyRead {
				m.ReadFiles = append(m.ReadFiles, file.AbsPath)
			}

			fileNamesForPrompt = append(fileNamesForPrompt, file.Info.FilePath)
			filesAdded++
		}

		// 4. If files were read, start a new turn with a simple confirmation message.
		if filesAdded > 0 {
			m.Println(fmt.Sprintf("Successfully read %d file(s) and added to context for this turn.", filesAdded))

			// Re-process immediately. The prompt does NOT contain the file content,
			// as it's now correctly in the ----FILES---- context block.
			nextPrompt := fmt.Sprintf("I have read the file(s): %s. What is the next step?", strings.Join(fileNamesForPrompt, ", "))
			accomplished := m.ProcessUserMessage(ctx, nextPrompt)
			return accomplished
		}
	}

	// Handle final state changes
	if r.RequestAccomplished {
		if !r.ExecPaneSeemsBusy && !r.NoComment {
			// Store only the actual user input and AI message, not the full structured content
			aiMsg := ChatMessage{
				Content:   r.Message,
				FromUser:  false,
				Timestamp: time.Now(),
			}
			m.Messages = append(m.Messages, historyUserMessage, aiMsg)
		}
		m.Status = ""
		return true
	}

	if r.WaitingForUserResponse {
		if !r.ExecPaneSeemsBusy && !r.NoComment {
			// Store only the actual user input and AI message, not the full structured content
			aiMsg := ChatMessage{
				Content:   r.Message,
				FromUser:  false,
				Timestamp: time.Now(),
			}
			m.Messages = append(m.Messages, historyUserMessage, aiMsg)
		}
		m.Status = "waiting"
		return false
	}

	// watch mode only
	if r.NoComment {
		// Do not append to history for NoComment
		return false
	}

	// This block handles asynchronous actions (SendKeys, Paste, or async ExecCommand).
	isAsyncAction := len(r.SendKeys) > 0 || len(r.PasteMultilineContent) > 0
	if len(r.ExecCommand) > 0 {
		isAsyncAction = true // Any exec command reaching here is async
	}

	if isAsyncAction || r.ExecPaneSeemsBusy {
		// For async actions, we append history, do a countdown, and then re-process.
		if !r.ExecPaneSeemsBusy && !r.NoComment {
			// Store only the actual user input and AI message, not the full structured content
			aiMsg := ChatMessage{
				Content:   r.Message,
				FromUser:  false,
				Timestamp: time.Now(),
			}
			m.Messages = append(m.Messages, historyUserMessage, aiMsg)
		}

		m.Countdown(m.GetWaitInterval())

		// If the AI didn't finish or ask us to wait, continue the loop.
		if !r.RequestAccomplished && !r.WaitingForUserResponse {
			accomplished := m.ProcessUserMessage(ctx, "Ok, that's done. Here is the current pane content, what's next?")
			if accomplished {
				return true
			}
		}
	}

	return false
}

func (m *Manager) startWatchMode(desc string) {

	// check status
	if m.Status == "" {
		return
	}

	m.Countdown(m.GetWaitInterval())

	// Create a new background context since this is a separate process
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	accomplished := m.ProcessUserMessage(ctx, desc)
	if accomplished {
		m.WatchMode = false
		m.Status = ""
	}

	// we continue running if status is still set
	if m.Status != "" && m.WatchMode {
		m.startWatchMode("")
	}
}

func (m *Manager) aiFollowedGuidelines(r AIResponse) (string, bool) {
	// Count state tags. Rule: Max 1 state tag.
	stateTags := 0
	if r.RequestAccomplished {
		stateTags++
	}
	if r.ExecPaneSeemsBusy {
		stateTags++
	}
	if r.WaitingForUserResponse {
		stateTags++
	}
	if r.NoComment {
		stateTags++
	}

	if stateTags > 1 {
		return "AI Error: Only one of <RequestAccomplished>, <ExecPaneSeemsBusy>, <WaitingForUserResponse>, or <NoComment> can be used at a time.", false
	}

	// Count action tags. Rule: Max 1 main action, can be combined with CreateExecPane.
	mainActionTags := 0
	if len(r.ExecCommand) > 0 {
		mainActionTags++
	}
	if len(r.SendKeys) > 0 {
		mainActionTags++
	}
	if len(r.PasteMultilineContent) > 0 {
		mainActionTags++
	}
	if len(r.ReadFile) > 0 {
		mainActionTags++
	}

	if mainActionTags > 1 {
		return "AI Error: Only one of <ExecCommand>, <TmuxSendKeys>, <PasteMultilineContent>, or <ReadFile> can be used at a time.", false
	}

	// Rule: State tags cannot be mixed with any action tags (including CreateExecPane).
	totalActionTags := mainActionTags
	if r.CreateExecPane {
		totalActionTags++
	}

	if stateTags > 0 && totalActionTags > 0 {
		return "AI Error: State tags (like <RequestAccomplished>) cannot be combined with action tags (like <ExecCommand> or <CreateExecPane>).", false
	}

	// Rule: A response must contain at least one tag.
	if stateTags == 0 && totalActionTags == 0 {
		return "AI Error: The response must contain at least one valid XML tag.", false
	}

	return "", true
}
