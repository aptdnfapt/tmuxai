package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/alvinunreal/tmuxai/logger"
)

const sessionDir = ".tmuxai"

// SessionData holds all the data that needs to be persisted for a session.
type SessionData struct {
	Title         string               `json:"title"`
	Messages      []ChatMessage        `json:"messages"`
	ExecHistory   []CommandExecHistory `json:"exec_history"`
	ReadFiles     []string             `json:"read_files"` // List of absolute paths of files read
	PreparedPanes map[string]bool      `json:"prepared_panes"`
	Panes         map[string]string    `json:"panes"`
	Timestamp     time.Time            `json:"timestamp"`
}

// ensureSessionDir creates the .tmuxai directory if it doesn't exist.
// It also ensures .tmuxai is in .gitignore if the project is a git repo.
func ensureSessionDir() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	sessionPath := filepath.Join(cwd, sessionDir)
	if err := os.MkdirAll(sessionPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create session directory: %w", err)
	}

	// Check for .git directory to determine if it's a git repository
	gitPath := filepath.Join(cwd, ".git")
	if _, err := os.Stat(gitPath); err == nil {
		// It's a git repo, ensure .gitignore has .tmuxai/
		gitignorePath := filepath.Join(cwd, ".gitignore")
		content, err := os.ReadFile(gitignorePath)
		if err != nil && !os.IsNotExist(err) {
			logger.Error("Could not read .gitignore: %v", err)
		}

		if !strings.Contains(string(content), sessionDir+"/") {
			f, err := os.OpenFile(gitignorePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				logger.Error("Could not open .gitignore to append: %v", err)
			} else {
				defer f.Close()
				if _, err := f.WriteString("\n" + sessionDir + "/\n"); err != nil {
					logger.Error("Could not write to .gitignore: %v", err)
				} else {
					logger.Info("Added '%s/' to .gitignore", sessionDir)
				}
			}
		}
	}

	return sessionPath, nil
}

// SessionInfo contains minimal information about a session for listing.
type SessionInfo struct {
	FilePath  string
	Title     string
	Timestamp time.Time
}

// ListSessions finds all session files and returns their information, sorted by most recent.
func ListSessions() ([]SessionInfo, error) {
	sessionPath, err := ensureSessionDir()
	if err != nil {
		return nil, err
	}

	files, err := os.ReadDir(sessionPath)
	if err != nil {
		return nil, fmt.Errorf("could not read session directory: %w", err)
	}

	var sessions []SessionInfo

	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".json" {
			filePath := filepath.Join(sessionPath, file.Name())
			data, err := os.ReadFile(filePath)
			if err != nil {
				logger.Error("Failed to read session file %s: %v", filePath, err)
				continue
			}

			var sessionData SessionData
			if err := json.Unmarshal(data, &sessionData); err != nil {
				logger.Error("Failed to parse session file %s: %v", filePath, err)
				continue
			}

			sessions = append(sessions, SessionInfo{
				FilePath:  filePath,
				Title:     sessionData.Title,
				Timestamp: sessionData.Timestamp,
			})
		}
	}

	// Sort sessions by timestamp, descending (most recent first)
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].Timestamp.After(sessions[j].Timestamp)
	})

	return sessions, nil
}

// findLatestSession finds the most recently modified session file in the session directory.
func findLatestSession() (string, error) {
	sessionPath, err := ensureSessionDir()
	if err != nil {
		return "", err
	}

	files, err := os.ReadDir(sessionPath)
	if err != nil {
		return "", fmt.Errorf("could not read session directory: %w", err)
	}

	var latestFile os.FileInfo
	var latestTime time.Time

	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".json" {
			info, err := file.Info()
			if err != nil {
				continue
			}
			if latestFile == nil || info.ModTime().After(latestTime) {
				latestFile = info
				latestTime = info.ModTime()
			}
		}
	}

	if latestFile == nil {
		return "", fmt.Errorf("no session files found")
	}

	return filepath.Join(sessionPath, latestFile.Name()), nil
}

// LoadSession loads a session from a JSON file.
func (m *Manager) LoadSession(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read session file %s: %w", path, err)
	}

	var sessionData SessionData
	if err := json.Unmarshal(data, &sessionData); err != nil {
		return fmt.Errorf("failed to unmarshal session data from %s: %w", path, err)
	}

	// Store session data for generating summaries later
	// This is a simplified approach for the new context management
	m.SessionOverrides["old_session_title"] = sessionData.Title
	m.SessionOverrides["old_session_timestamp"] = sessionData.Timestamp
	m.SessionOverrides["old_session_messages"] = sessionData.Messages
	m.SessionOverrides["old_session_panes"] = sessionData.Panes
	m.SessionOverrides["old_session_files"] = sessionData.ReadFiles

	// The current session starts fresh.
	m.Messages = []ChatMessage{}
	m.ExecHistory = []CommandExecHistory{}
	m.ReadFiles = []string{}
	m.SessionPath = path // Keep track of the loaded session path to save over it.
	if sessionData.PreparedPanes != nil {
		m.PreparedPanes = sessionData.PreparedPanes // Restore prepared state
	}

	m.Println(fmt.Sprintf("Restored context from session: '%s'", sessionData.Title))
	logger.Info("Session restored from %s", path)
	return nil
}

// SaveSession saves the current session state to a JSON file.
func (m *Manager) SaveSession() error {
	// Don't save empty sessions
	if len(m.Messages) == 0 {
		logger.Info("No messages in history, skipping session save.")
		return nil
	}

	sessionPath, err := ensureSessionDir()
	if err != nil {
		return err
	}

	title := ""
	if m.SessionPath != "" {
		// Existing session, read title from it
		data, err := os.ReadFile(m.SessionPath)
		if err == nil {
			var oldSessionData SessionData
			if json.Unmarshal(data, &oldSessionData) == nil {
				title = oldSessionData.Title
			}
		}
	}

	// If no title, generate one
	if title == "" {
		title, err = m.generateSessionTitle()
		if err != nil {
			logger.Error("Failed to generate session title: %v. Using default.", err)
			title = fmt.Sprintf("Session from %s", time.Now().Format("2006-01-02 15:04"))
		}
	}

	// Capture pane contents at time of saving
	panes, _ := m.GetTmuxPanes()
	paneContents := make(map[string]string)
	for _, pane := range panes {
		paneContents[pane.Id] = pane.Content
	}

	sessionData := SessionData{
		Title:         title,
		Messages:      m.Messages,
		ExecHistory:   m.ExecHistory,
		ReadFiles:     m.ReadFiles,
		PreparedPanes: m.PreparedPanes,
		Panes:         paneContents,
		Timestamp:     time.Now(),
	}

	data, err := json.MarshalIndent(sessionData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal session data: %w", err)
	}

	// Use existing path or create a new one
	savePath := m.SessionPath
	if savePath == "" {
		// Sanitize title for filename
		safeTitle := strings.ReplaceAll(strings.ToLower(title), " ", "_")
		safeTitle = strings.Map(func(r rune) rune {
			if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '_' {
				return r
			}
			return -1
		}, safeTitle)

		if len(safeTitle) > 50 {
			safeTitle = safeTitle[:50]
		}

		timestamp := time.Now().Format("20060102150405")
		filename := fmt.Sprintf("%s_%s.json", timestamp, safeTitle)
		savePath = filepath.Join(sessionPath, filename)
		m.SessionPath = savePath
	}

	logger.Info("Saving session to %s", savePath)
	return os.WriteFile(savePath, data, 0644)
}

// generateSessionTitle asks the AI to create a title for the conversation.
func (m *Manager) generateSessionTitle() (string, error) {
	if len(m.Messages) == 0 {
		return "Empty Session", nil
	}

	var convo strings.Builder
	for _, msg := range m.Messages {
		if msg.FromUser {
			convo.WriteString("User: " + msg.Content + "\n")
		} else {
			convo.WriteString("Assistant: " + msg.Content + "\n")
		}
	}

	prompt := fmt.Sprintf("Based on the following conversation, create a very short, descriptive title (5-7 words max) for this session. Just return the title, nothing else.\n\nCONVERSATION:\n%s", convo.String())

	titleMessage := []ChatMessage{
		{Content: prompt, FromUser: true, Timestamp: time.Now()},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	title, err := m.AiClient.GetResponseFromChatMessages(ctx, titleMessage, m.GetOpenRouterModel())
	if err != nil {
		return "", err
	}

	// Clean up response
	title = strings.Trim(title, "\n \"'")
	return title, nil
}
