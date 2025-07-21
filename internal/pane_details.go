package internal

import (
	"github.com/alvinunreal/tmuxai/system"
)

func (m *Manager) GetTmuxPanes() ([]system.TmuxPaneDetails, error) {
	currentPaneId, _ := system.TmuxCurrentPaneId()
	windowTarget, _ := system.TmuxCurrentWindowTarget()
	currentPanes, _ := system.TmuxPanesDetails(windowTarget)

	for i := range currentPanes {
		currentPanes[i].IsTmuxAiPane = currentPanes[i].Id == currentPaneId
		currentPanes[i].IsTmuxAiExecPane = currentPanes[i].Id == m.ExecPane.Id
		currentPanes[i].IsPrepared = m.PreparedPanes[currentPanes[i].Id]
		if currentPanes[i].IsSubShell {
			currentPanes[i].OS = "OS Unknown (subshell)"
		} else {
			currentPanes[i].OS = m.OS
		}

	}
	return currentPanes, nil
}
