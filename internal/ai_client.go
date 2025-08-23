package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/alvinunreal/tmuxai/config"
	"github.com/alvinunreal/tmuxai/logger"
)

// AiClient represents an AI client for interacting with OpenRouter API
type AiClient struct {
	config *config.OpenRouterConfig
	client *http.Client
}

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatCompletionRequest represents a request to the chat completion API
type ChatCompletionRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

// ChatCompletionChoice represents a choice in the chat completion response
type ChatCompletionChoice struct {
	Index   int     `json:"index"`
	Message Message `json:"message"`
}

// ChatCompletionResponse represents a response from the chat completion API
type ChatCompletionResponse struct {
	ID      string                 `json:"id"`
	Object  string                 `json:"object"`
	Created int64                  `json:"created"`
	Choices []ChatCompletionChoice `json:"choices"`
}

func NewAiClient(cfg *config.OpenRouterConfig) *AiClient {
	return &AiClient{
		config: cfg,
		client: &http.Client{},
	}
}

// GetResponseFromChatMessages gets a response from the AI based on chat messages
func (c *AiClient) GetResponseFromChatMessages(ctx context.Context, chatMessages []ChatMessage, model string) (string, error) {
	// Convert chat messages to AI client format
	aiMessages := []Message{}

	for i, msg := range chatMessages {
		var role string

		if i == 0 && !msg.FromUser {
			role = "system"
		} else if msg.FromUser {
			role = "user"
		} else {
			role = "assistant"
		}

		aiMessages = append(aiMessages, Message{
			Role:    role,
			Content: msg.Content,
		})
	}

	logger.Info("Sending %d messages to AI", len(aiMessages))

	// Get response from AI
	response, err := c.ChatCompletion(ctx, aiMessages, model)
	if err != nil {
		return "", err
	}

	return response, nil
}

// ChatCompletion sends a chat completion request to the OpenRouter API
func (c *AiClient) ChatCompletion(ctx context.Context, messages []Message, model string) (string, error) {
	reqBody := ChatCompletionRequest{
		Model:    model,
		Messages: messages,
	}

	reqJSON, err := json.Marshal(reqBody)
	if err != nil {
		logger.Error("Failed to marshal request: %v", err)
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Remove trailing slash from BaseURL if present: https://github.com/alvinunreal/tmuxai/issues/13
	baseURL := strings.TrimSuffix(c.config.BaseURL, "/")
	url := baseURL + "/chat/completions"

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqJSON))
	if err != nil {
		logger.Error("Failed to create request: %v", err)
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.config.APIKey)

	req.Header.Set("HTTP-Referer", "https://github.com/alvinunreal/tmuxai")
	req.Header.Set("X-Title", "TmuxAI")

	// Log the request details for debugging before sending
	logger.Debug("Sending API request to: %s with model: %s", url, model)

	// Send the request
	resp, err := c.client.Do(req)
	if err != nil {
		if ctx.Err() == context.Canceled {
			return "", fmt.Errorf("request canceled: %w", ctx.Err())
		}
		logger.Error("Failed to send request: %v", err)
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("Failed to read response: %v", err)
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Log the raw response for debugging
	logger.Debug("API response status: %d, response size: %d bytes", resp.StatusCode, len(body))

	// Check for errors
	if resp.StatusCode != http.StatusOK {
		logger.Error("API returned error: %s", body)
		return "", fmt.Errorf("API returned error: %s", body)
	}

	// Parse the response
	var completionResp ChatCompletionResponse
	if err := json.Unmarshal(body, &completionResp); err != nil {
		logger.Error("Failed to unmarshal response: %v, body: %s", err, body)
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Return the response content
	if len(completionResp.Choices) > 0 {
		responseContent := completionResp.Choices[0].Message.Content
		logger.Debug("Received AI response (%d characters): %s", len(responseContent), responseContent)
		return responseContent, nil
	}

	// Enhanced error for no completion choices
	logger.Error("No completion choices returned. Raw response: %s", string(body))
	return "", fmt.Errorf("no completion choices returned (model: %s, status: %d)", model, resp.StatusCode)
}

func debugChatMessages(chatMessages []ChatMessage, response string, cfg *config.Config) {
	timestamp := time.Now().Format("20060102-150405")

	// Use configured debug directory if available, otherwise use default
	debugDir := ""
	if cfg.DebugDir != "" {
		// Expand ~ to home directory
		if strings.HasPrefix(cfg.DebugDir, "~/") {
			homeDir, err := os.UserHomeDir()
			if err == nil {
				debugDir = filepath.Join(homeDir, cfg.DebugDir[2:])
			} else {
				// Fall back to default if we can't get home directory
				configDir, _ := config.GetConfigDir()
				debugDir = fmt.Sprintf("%s/debug", configDir)
			}
		} else {
			debugDir = cfg.DebugDir
		}
	} else {
		configDir, _ := config.GetConfigDir()
		debugDir = fmt.Sprintf("%s/debug", configDir)
	}

	// Create debug directory if it doesn't exist
	if _, err := os.Stat(debugDir); os.IsNotExist(err) {
		os.Mkdir(debugDir, 0755)
	}

	debugFileName := fmt.Sprintf("%s/debug-%s.txt", debugDir, timestamp)

	file, err := os.Create(debugFileName)
	if err != nil {
		logger.Error("Failed to create debug file: %v", err)
		return
	}
	defer file.Close()

	// Include conversation history and current structured context for better debugging
	// (Excluding system prompt as it's always the same)
	if len(chatMessages) > 0 {
		file.WriteString("==================    SENT REQUEST ==================\n")
		
		// Second Block: Conversation History (excluding system prompt)
		if len(chatMessages) > 1 {
			file.WriteString("\n-------------------- SECOND BLOCK: CONVERSATION HISTORY --------------------\n")
			startIdx := 1
			// Skip system prompt if it's the first message
			if !chatMessages[0].FromUser {
				startIdx = 1
			} else {
				startIdx = 0
			}
			
			for i := startIdx; i < len(chatMessages)-1; i++ {
				msg := chatMessages[i]
				role := "assistant"
				if msg.FromUser {
					role = "user"
				}
				timeStr := msg.Timestamp.Format(time.RFC3339)
				file.WriteString(fmt.Sprintf("\nMessage %d (%s) at %s:\n%s\n", i, role, timeStr, msg.Content))
			}
		}
		
		// Third Block: Current Structured Context
		if len(chatMessages) > 1 {
			file.WriteString("\n-------------------- THIRD BLOCK: CURRENT STRUCTURED CONTEXT --------------------\n")
			msg := chatMessages[len(chatMessages)-1]
			timeStr := msg.Timestamp.Format(time.RFC3339)
			file.WriteString(fmt.Sprintf("\nRole: user at %s\nContent:\n%s\n", timeStr, msg.Content))
		}
	}

	file.WriteString("==================    RECEIVED RESPONSE ==================\n\n")
	file.WriteString(response)
	file.WriteString("\n\n==================    END DEBUG ==================\n")
}
