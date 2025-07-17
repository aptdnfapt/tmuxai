package internal

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"

	"github.com/fatih/color"
)


type CLIInterface struct {
	manager     *Manager
	initMessage string
}

func NewCLIInterface(manager *Manager) *CLIInterface {
	return &CLIInterface{
		manager:     manager,
		initMessage: "",
	}
}

// Start starts the CLI interface
func (c *CLIInterface) Start(initMessage string) error {
	c.printWelcomeMessage()

	if initMessage != "" {
		fmt.Println(c.manager.GetPrompt() + initMessage)
		c.processInput(initMessage)
	}

	for {
		line, err := ShowPromptEditor()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error running prompt editor: %v\n", err)
			return err
		}

		// User cancelled input by pressing Esc.
		if line == "" {
			continue
		}

		// After the editor closes, print the prompt and the entered command
		// so it appears correctly in the terminal's scrollback history.
		userColor := color.New(color.FgCyan, color.Bold)
		colonColor := color.New(color.FgYellow, color.Bold)
		fmt.Println(userColor.Sprint("User") + colonColor.Sprint(" : ") + line)

		// Check for exit commands.
		trimmed := strings.TrimSpace(line)
		if trimmed == "exit" || trimmed == "quit" || trimmed == "/exit" {
			c.manager.SaveSession()
			return nil
		}

		c.processInput(line)
	}
}

// printWelcomeMessage prints a welcome message
func (c *CLIInterface) printWelcomeMessage() {
	fmt.Println()
	fmt.Println("Type '/help' for a list of commands, '/exit' to quit")
	fmt.Println()
}

func (c *CLIInterface) processInput(input string) {
	if c.manager.IsMessageSubcommand(input) {
		c.manager.ProcessSubCommand(input)
		return
	}

	// Set up signal handling for Ctrl+C
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)

	// Set up a notification channel
	done := make(chan struct{})

	// Create a cancellable context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Launch a goroutine just for handling the interrupt
	go func() {
		select {
		case <-sigChan:
			cancel()
			c.manager.Status = ""
			c.manager.WatchMode = false
		case <-done:
		}
	}()

	// Run the message processing in the main thread
	c.manager.Status = "running"
	c.manager.ProcessUserMessage(ctx, input)
	c.manager.Status = ""

	close(done)

	signal.Stop(sigChan)
}

