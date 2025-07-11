package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/alvinunreal/tmuxai/logger"
)

const sessionDir = ".tmuxai"

// SessionData holds all the data that needs to be persisted for a session.
type SessionData struct {
	Title       string               `json:"title"`
	Messages    []ChatMessage        `json:"messages"`
	ExecHistory []CommandExecHistory `json:"exec_history"`
	ReadFiles   []string             `json:"read_files"` // List of absolute paths of files read
	Timestamp   time.Time            `json:"timestamp"`
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

	m.Messages = sessionData.Messages
	m.ExecHistory = sessionData.ExecHistory
	m.ReadFiles = sessionData.ReadFiles
	m.SessionPath = path

	m.Println(fmt.Sprintf("Restored session: '%s'", sessionData.Title))
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

	sessionData := SessionData{
		Title:       title,
		Messages:    m.Messages,
		ExecHistory: m.ExecHistory,
		ReadFiles:   m.ReadFiles,
		Timestamp:   time.Now(),
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
