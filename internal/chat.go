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
	messageToProcess := input
	isCommand := c.manager.IsMessageSubcommand(input)

	userColor := color.New(color.FgCyan, color.Bold)
	colonColor := color.New(color.FgYellow, color.Bold)

	if isCommand {
		var shouldProcessMessage bool
		messageToProcess, shouldProcessMessage = c.manager.ProcessSubCommand(input)
		if !shouldProcessMessage {
			return // It was a normal command (like /info), don't proceed.
		}
		// If a command returns a message to process (i.e., from /edit), print it now.
		fmt.Println(userColor.Sprint("User") + colonColor.Sprint(" : ") + messageToProcess)
	} else {
		// This is a regular message, not a command. Print it.
		fmt.Println(userColor.Sprint("User") + colonColor.Sprint(" : ") + input)
	}

	// At this point, messageToProcess is either the original input (if not a command)
	// or the content from the editor (if it was /edit).
	if strings.TrimSpace(messageToProcess) == "" {
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
	c.manager.ProcessUserMessage(ctx, messageToProcess)
	c.manager.Status = ""

	close(done)

	signal.Stop(sigChan)
}
