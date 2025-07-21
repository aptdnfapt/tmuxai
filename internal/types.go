package internal

import (
	"fmt"
	"time"
)

// Message represents a chat message
type ChatMessage struct {
	Content   string    `json:"content"`
	FromUser  bool      `json:"from_user"`
	Timestamp time.Time `json:"timestamp"`
}

// Parsed only when pane is prepared
type CommandExecHistory struct {
	Command string `json:"command"`
	Output  string `json:"output"`
	Code    int    `json:"code"`
}

type ExecCommandInfo struct {
	Command string `json:"command"`
	PaneID  string `json:"pane_id"`
	Wait    bool   `json:"wait"`
}

type SendKeysInfo struct {
	Keys   string `json:"keys"`
	PaneID string `json:"pane_id"`
}

type PasteInfo struct {
	Content string `json:"content"`
	PaneID  string `json:"pane_id"`
}

type ReadFileInfo struct {
	FilePath string `json:"file_path"`
	PaneID   string `json:"pane_id"`
}

type AIResponse struct {
	Message                string            `json:"message"`
	SendKeys               []SendKeysInfo    `json:"send_keys"`
	ExecCommand            []ExecCommandInfo `json:"exec_command"`
	PasteMultilineContent  []PasteInfo       `json:"paste_multiline_content"`
	ReadFile               []ReadFileInfo    `json:"read_file"`
	RequestAccomplished    bool              `json:"request_accomplished"`
	ExecPaneSeemsBusy      bool              `json:"exec_pane_seems_busy"`
	WaitingForUserResponse bool              `json:"waiting_for_user_response"`
	NoComment              bool              `json:"no_comment"`
	CreateExecPane         bool              `json:"create_exec_pane"`
}

func (ai *AIResponse) String() string {
	var execCommands []string
	for _, cmd := range ai.ExecCommand {
		execCommands = append(execCommands, fmt.Sprintf("{Cmd: %s, PaneID: %s, Wait: %v}", cmd.Command, cmd.PaneID, cmd.Wait))
	}
	var sendKeys []string
	for _, sk := range ai.SendKeys {
		sendKeys = append(sendKeys, fmt.Sprintf("{Keys: %s, PaneID: %s}", sk.Keys, sk.PaneID))
	}
	var pasteContent []string
	for _, pc := range ai.PasteMultilineContent {
		pasteContent = append(pasteContent, fmt.Sprintf("{Content: %s, PaneID: %s}", pc.Content, pc.PaneID))
	}
	var readFiles []string
	for _, rf := range ai.ReadFile {
		readFiles = append(readFiles, fmt.Sprintf("{FilePath: %s, PaneID: %s}", rf.FilePath, rf.PaneID))
	}

	return fmt.Sprintf(`
	Message: %s
	SendKeys: %v
	ExecCommand: %v
	PasteMultilineContent: %v
	ReadFile: %v
	RequestAccomplished: %v
	ExecPaneSeemsBusy: %v
	WaitingForUserResponse: %v
	NoComment: %v
	CreateExecPane: %v
`,
		ai.Message,
		sendKeys,
		execCommands,
		pasteContent,
		readFiles,
		ai.RequestAccomplished,
		ai.ExecPaneSeemsBusy,
		ai.WaitingForUserResponse,
		ai.NoComment,
		ai.CreateExecPane,
	)
}

type ItemStatus string

const (
	StatusActive     ItemStatus = "ACTIVE"
	StatusNew        ItemStatus = "NEW"
	StatusUpdated    ItemStatus = "UPDATED"
	StatusUnchanged  ItemStatus = "UNCHANGED"
	StatusRemoved    ItemStatus = "REMOVED"
	StatusOldSession ItemStatus = "OLD_SESSION"
)

type SectionState struct {
	PreviousContent string
	Content         string
	LastChanged     int // which message number it was last changed
	Hash            string
	Size            int // content size in tokens/bytes
	Status          ItemStatus
	Timestamp       time.Time // when last updated
	RemovedAt       time.Time // when removed (if applicable)
}

type OldSessionData struct {
	SessionName  string
	SavedAt      time.Time
	Files        map[string]SectionState // filepath -> state
	Panes        map[string]SectionState // paneID -> state
	Conversation []ChatMessage
}

type ContextState struct {
	RepoMap     SectionState
	Files       map[string]SectionState // filepath -> state
	Panes       map[string]SectionState // paneID -> state
	Prompts     SectionState
	OldSession  *OldSessionData // pointer to restored session data
	LastUpdate  int             // message number
	CurrentTime time.Time       // current timestamp
}
