package internal

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/alvinunreal/tmuxai/logger"
	"github.com/alvinunreal/tmuxai/system"
)

// GetAvailablePane finds an available pane or creates a new one if none are available
func (m *Manager) GetAvailablePane() system.TmuxPaneDetails {
	panes, _ := m.GetTmuxPanes()
	for _, pane := range panes {
		if !pane.IsTmuxAiPane {
			logger.Info("Found available pane: %s", pane.Id)
			return pane
		}
	}

	return system.TmuxPaneDetails{}
}

func (m *Manager) InitExecPane() {
	availablePane := m.GetAvailablePane()
	if availablePane.Id == "" {
		system.TmuxCreateNewPane(m.PaneId)
		availablePane = m.GetAvailablePane()
	}
	m.ExecPane = &availablePane
}

func (m *Manager) CreateNewExecPane() {
	system.TmuxCreateNewPane(m.PaneId)
	// The new pane might take a moment to register, so we get all panes again
	availablePane := m.GetAvailablePane()
	if availablePane.Id != "" {
		m.ExecPane = &availablePane
		m.Println(fmt.Sprintf("Created new exec pane %s. It is now the primary exec pane.", m.ExecPane.Id))
	} else {
		m.Println("Error: Failed to create or find a new exec pane.")
	}
}

func (m *Manager) PreparePane(targetPane *system.TmuxPaneDetails) {
	targetPane.Refresh(m.GetMaxCaptureLines())
	if targetPane.IsPrepared {
		m.Println(fmt.Sprintf("Pane %s is already prepared.", targetPane.Id))
		return
	}

	// This now simply flags the pane for marker-based execution. No more prompt injection.
	targetPane.IsPrepared = true
	m.Println(fmt.Sprintf("Pane %s is now prepared for synchronous command execution.", targetPane.Id))
}

func (m *Manager) PrepareExecPane() {
	if m.ExecPane == nil || m.ExecPane.Id == "" {
		m.Println("No exec pane initialized to prepare.")
		return
	}
	m.PreparePane(m.ExecPane)
}

func (m *Manager) ExecWaitCapture(targetPane *system.TmuxPaneDetails) (CommandExecHistory, error) {
	const endMarker = "TMUXAI:EXITCODE"
	// This regex ensures we match the marker only when it appears on its own line,
	// preventing a match on the command prompt itself.
	re := regexp.MustCompile(`^` + endMarker + `:(-?\d+)$`)

	m.Println("") // Newline for the animation

	animChars := []string{"⋯", "⋱", "⋮", "⋰"}
	animIndex := 0
	for m.Status != "" {
		fmt.Printf("\r%s%s ", m.GetPrompt(), animChars[animIndex])
		animIndex = (animIndex + 1) % len(animChars)
		time.Sleep(500 * time.Millisecond)
		targetPane.Refresh(m.GetMaxCaptureLines())

		// Scan from the bottom of the content for the marker for efficiency
		lines := strings.Split(targetPane.Content, "\n")
		for i := len(lines) - 1; i >= 0; i-- {
			trimmedLine := strings.TrimSpace(lines[i])
			if re.MatchString(trimmedLine) {
				fmt.Print("\r\033[K") // Clear animation line

				// Parse exit code from the marker line. Handles negative codes.
				matches := re.FindStringSubmatch(trimmedLine)
				code := -1 // Default code if parsing fails
				if len(matches) > 1 {
					parsedCode, err := strconv.Atoi(matches[1])
					if err == nil {
						code = parsedCode
					}
				}

				// The output is everything *before* the marker line in the pane.
				// We reconstruct the content without the marker line.
				var outputBuilder strings.Builder
				for j, contentLine := range lines {
					if i == j { // This is the marker line, skip it
						continue
					}
					outputBuilder.WriteString(contentLine)
					outputBuilder.WriteString("\n")
				}

				return CommandExecHistory{
					Output: strings.TrimSpace(outputBuilder.String()),
					Code:   code,
				}, nil
			}
		}
	}

	fmt.Print("\r\033[K") // Clear animation on exit
	return CommandExecHistory{}, fmt.Errorf("operation cancelled")
}
