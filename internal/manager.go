package internal

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/alvinunreal/tmuxai/config"
	"github.com/alvinunreal/tmuxai/logger"
	"github.com/alvinunreal/tmuxai/system"
	"github.com/fatih/color"
)

// Manager represents the TmuxAI manager agent
type Manager struct {
	Config           *config.Config
	AiClient         *AiClient
	Status           string // running, waiting, done
	PaneId           string
	ExecPane         *system.TmuxPaneDetails
	Messages         []ChatMessage
	ExecHistory      []CommandExecHistory
	ReadFiles        []string
	WatchMode        bool
	OS               string
	SessionOverrides map[string]interface{} // session-only config overrides
	PreparedPanes    map[string]bool
	LastExecPaneID   string
	SessionPath      string
	isRestore        bool
	RepoMap          *RepoMapHandler
	ContextTracker   *ContextStateTracker
	MessageBuilder   *StructuredMessageBuilder
	OldSession       *OldSessionData
}

// NewManager creates a new manager agent
func NewManager(cfg *config.Config, isRestore bool) (*Manager, error) {
	if cfg.OpenRouter.APIKey == "" {
		fmt.Println("OpenRouter API key is required. Set it in the config file or as an environment variable: TMUXAI_OPENROUTER_API_KEY")
		return nil, fmt.Errorf("OpenRouter API key is required")
	}

	paneId, err := system.TmuxCurrentPaneId()
	if err != nil {
		// If we're not in a tmux session, start a new session and execute the same command
		paneId, err = system.TmuxCreateSession()
		if err != nil {
			return nil, fmt.Errorf("system.TmuxCreateSession failed: %w", err)
		}
		args := strings.Join(os.Args[1:], " ")

		system.TmuxSendCommandToPane(paneId, "tmuxai "+args, true)
		// shell initialization may take some time
		time.Sleep(1 * time.Second)
		system.TmuxSendCommandToPane(paneId, "Enter", false)
		err = system.TmuxAttachSession(paneId)
		if err != nil {
			return nil, fmt.Errorf("system.TmuxAttachSession failed: %w", err)
		}
		os.Exit(0)
	}

	aiClient := NewAiClient(&cfg.OpenRouter)
	osName := system.GetOSDetails()

	manager := &Manager{
		Config:           cfg,
		AiClient:         aiClient,
		PaneId:           paneId,
		Messages:         []ChatMessage{},
		ExecPane:         &system.TmuxPaneDetails{},
		OS:               osName,
		SessionOverrides: make(map[string]interface{}),
		PreparedPanes:    make(map[string]bool),
		LastExecPaneID:   "",
		ReadFiles:        []string{},
		SessionPath:      "",
		isRestore:        isRestore,
	}

	manager.ContextTracker = NewContextStateTracker()
	manager.MessageBuilder = NewStructuredMessageBuilder(manager)

	// Initialize RepoMap if in agentic mode
	if manager.GetAgenticMode() {
		manager.RepoMap = NewRepoMapHandler()
	}

	// Session loading logic
	if manager.isRestore {
		latestSession, err := findLatestSession()
		if err != nil {
			// It's not an error if no session is found, just log it.
			logger.Info("Restore flag is set, but no previous session found: %v", err)
		} else {
			if err := manager.LoadSession(latestSession); err != nil {
				// Also not a fatal error, just log and continue with a new session.
				logger.Error("Failed to load session %s: %v", latestSession, err)
			}
		}
	}

	manager.InitExecPane()
	return manager, nil
}

// Start starts the manager agent
func (m *Manager) Start(initMessage string) error {
	cliInterface := NewCLIInterface(m)
	if initMessage != "" {
		logger.Info("Initial task provided: %s", initMessage)
	}
	if err := cliInterface.Start(initMessage); err != nil {
		logger.Error("Failed to start CLI interface: %v", err)
		return err
	}

	return nil
}

func (m *Manager) Println(msg string) {
	fmt.Println(m.GetPrompt() + msg)
}

// PrintContextUsage shows current token usage as a subtle progress bar
func (m *Manager) PrintContextUsage() {
	var totalTokens int
	for _, msg := range m.Messages {
		totalTokens += system.EstimateTokenCount(msg.Content)
	}

	usagePercent := 0.0
	if m.GetMaxContextSize() > 0 {
		usagePercent = float64(totalTokens) / float64(m.GetMaxContextSize()) * 100
	}

	dimColor := color.New(color.FgHiBlack)
	formatter := system.NewInfoFormatter()
	fmt.Printf("%s %s\n",
		dimColor.Sprintf("%d tokens", totalTokens),
		dimColor.Sprintf("[%s]", formatter.FormatProgressBar(usagePercent, 10)),
	)
}

func (m *Manager) GetAIChatPrompt() string {
	tmuxaiColor := color.New(color.FgGreen, color.Bold)
	colonColor := color.New(color.FgYellow, color.Bold)
	return tmuxaiColor.Sprint("TmuxAI") + colonColor.Sprint(" : ")
}

func (m *Manager) GetConfig() *config.Config {
	return m.Config
}

// getPrompt returns the prompt string with color
func (m *Manager) GetPrompt() string {
	tmuxaiColor := color.New(color.FgGreen, color.Bold)
	arrowColor := color.New(color.FgYellow, color.Bold)
	stateColor := color.New(color.FgMagenta, color.Bold)

	var stateSymbol string
	switch m.Status {
	case "running":
		stateSymbol = "▶"
	case "waiting":
		stateSymbol = "?"
	case "done":
		stateSymbol = "✓"
	default:
		stateSymbol = ""
	}
	if m.WatchMode {
		stateSymbol = "∞"
	}

	prompt := tmuxaiColor.Sprint("TmuxAI")
	if stateSymbol != "" {
		prompt += " " + stateColor.Sprint("["+stateSymbol+"]")
	}
	prompt += arrowColor.Sprint(" » ")
	return prompt
}
