package internal

import (
	"fmt"
	"time"

	"github.com/alvinunreal/tmuxai/logger"
	"github.com/alvinunreal/tmuxai/system"
)

func (m *Manager) ApplyLayout(layout string) error {
	switch layout {
	case "1x2":
		m.setup1x2Layout()
	default:
		return fmt.Errorf("unknown layout: %s", layout)
	}
	return nil
}

func (m *Manager) setup1x2Layout() {
	// If an exec pane already exists, remove it to make way for the new layout.
	if m.ExecPane != nil && m.ExecPane.Id != "" && m.ExecPane.Id != m.PaneId {
		m.Println(fmt.Sprintf("Removing existing exec pane %s to create new layout...", m.ExecPane.Id))
		if err := system.TmuxKillPane(m.ExecPane.Id); err != nil {
			m.Println(fmt.Sprintf("Error removing existing exec pane: %v", err))
			logger.Error("Error removing existing exec pane for 1x2 layout: %v", err)
			return
		}
		time.Sleep(250 * time.Millisecond) // Give tmux time to process
	}
	// 1. Split current window to create top pane (70%)
	// The -b flag makes the new pane appear *before* the target, which means above it in a vertical split.
	// We target the current chat pane (m.PaneId)
	m.Println("Creating 1x2 layout...")
	mainPaneID, err := system.TmuxSplitWindow(m.PaneId, "-v -b -l 70%")
	if err != nil {
		m.Println(fmt.Sprintf("Error creating main pane: %v", err))
		logger.Error("Error creating main pane for 1x2 layout: %v", err)
		return
	}
	m.Println(fmt.Sprintf("Created main pane %s.", mainPaneID))

	// 2. Split the chat pane horizontally to create the bottom-right pane
	// The current pane (m.PaneId) is now at the bottom. We split it.
	sidePaneID, err := system.TmuxSplitWindow(m.PaneId, "-h")
	if err != nil {
		m.Println(fmt.Sprintf("Error creating side pane: %v", err))
		logger.Error("Error creating side pane for 1x2 layout: %v", err)
		// Maybe try to kill the mainPane to revert? For now, just log.
		return
	}
	m.Println(fmt.Sprintf("Created side exec pane %s.", sidePaneID))

	// 3. Update manager's state
	// The main pane should be the new primary execution pane.
	// We need to refresh the panes list to find the new pane details.
	panes, err := m.GetTmuxPanes()
	if err != nil {
		m.Println("Error getting panes after layout change.")
		logger.Error("Error getting panes after 1x2 layout change: %v", err)
		return
	}

	found := false
	for i := range panes {
		if panes[i].Id == mainPaneID {
			m.ExecPane = &panes[i]
			found = true
			break
		}
	}

	if found {
		m.Println(fmt.Sprintf("Set %s as primary exec pane.", mainPaneID))
	} else {
		m.Println(fmt.Sprintf("Could not find new main pane %s to set as primary exec pane.", mainPaneID))
		logger.Error("Could not find new main pane %s to set as primary exec pane.", mainPaneID)
	}

	m.Println("Layout 1x2 created successfully.")
}
