# TmuxAI Project Documentation

## Project Overview

TmuxAI is an intelligent terminal assistant that integrates with tmux sessions to provide AI-powered assistance directly within the terminal environment. Unlike other CLI AI tools, TmuxAI observes and understands the content of tmux panes, providing contextual assistance without requiring users to change their workflow or interrupt their terminal sessions.

TmuxAI functions as a pair programmer that sits beside the user, watching their terminal environment exactly as they see it. It can understand what the user is working on across multiple panes, help solve problems, and execute commands on the user's behalf in a dedicated execution pane.

## Core Philosophy

TmuxAI's design philosophy mirrors the way humans collaborate at the terminal:

1. **Observes**: Reads the visible content in all panes
2. **Communicates**: Uses a dedicated chat pane for interaction
3. **Acts**: Can execute commands in a separate execution pane (with user permission)

This approach provides powerful AI assistance while respecting the user's existing workflow and maintaining the familiar terminal environment.

## System Architecture

TmuxAI is built in Go and follows a modular architecture:

### Core Components

1. **Manager**: The central component that coordinates all activities, manages the tmux session, and handles user interactions.
2. **AI Client**: Handles communication with the AI service (OpenRouter by default, but configurable for other providers).
3. **CLI Interface**: Provides the user interface for interacting with TmuxAI.
4. **Pane Management**: Handles the creation, monitoring, and interaction with tmux panes.
5. **Message Processing**: Processes user messages and AI responses.

### Key Files and Their Functions

- **main.go**: Entry point that initializes the logger and executes the CLI command.
- **cli/cli.go**: Defines the command-line interface using Cobra.
- **internal/manager.go**: Implements the Manager struct and its methods for managing the TmuxAI session.
- **internal/ai_client.go**: Handles communication with the AI service.
- **internal/chat.go**: Implements the CLI interface for user interaction.
- **internal/process_message.go**: Processes user messages and sends them to the AI.
- **internal/process_response.go**: Parses and processes AI responses.
- **config/config.go**: Handles configuration loading and management.
- **system/**: Contains utilities for interacting with the tmux system.

## Operation Modes

TmuxAI operates in several modes, each with specific behaviors:

### 1. Observe Mode (Default)

In this mode, TmuxAI:
- Captures context from all visible panes
- Processes user requests with this context
- Suggests commands that can be executed with user confirmation
- Uses a countdown timer before checking for new output
- Continues the conversation based on updated pane content

### 2. Prepare Mode

An enhanced execution mode that:
- Appends a special marker to commands for reliable execution tracking
- Waits for command completion by watching for the marker
- Captures the command's exit code for better contextual awareness
- Eliminates the need for fixed wait intervals

### 3. Agentic Mode

The most powerful mode that allows TmuxAI to:
- Create new panes for specific tasks
- Execute commands in any pane by referencing its ID
- Send keystrokes or paste content into any pane
- Perform actions in multiple panes simultaneously

### 4. Watch Mode

A proactive monitoring mode where TmuxAI:
- Continuously monitors terminal activity
- Provides suggestions based on what the user is doing
- Can detect inefficient commands, potential errors, or security issues

## Key Features

### 1. Context-Aware Assistance

TmuxAI understands the current state of all visible panes, including:
- Current command with arguments
- Detected shell type
- User's operating system
- Content of each pane

### 2. Command Execution

TmuxAI can execute commands in a designated execution pane:
- Commands are checked against whitelist/blacklist patterns
- User confirmation is required (unless whitelisted)
- Output is captured and used for further assistance

### 3. File Reading

TmuxAI can read files to understand code and project structure:
- Supports reading single or multiple files
- Provides file content to the AI for context
- Maintains a list of previously read files

### 4. Context Management

To manage token usage with AI models, TmuxAI implements "squashing":
- Summarizes chat history when context grows too large
- Automatically triggers when context reaches 80% of maximum size
- Can be manually triggered with the `/squash` command

### 5. Multi-Pane Interaction (Agentic Mode)

In agentic mode, TmuxAI can:
- Create new panes
- Execute commands in specific panes
- Send keystrokes to panes
- Paste multiline content into panes

### 6. External Tool Integration (Aider)

TmuxAI can work with Aider (a completely separate, standalone command-line AI coding assistant) through configuration:
- TmuxAI and Aider are **entirely separate projects** developed independently
- The connection is established solely through TmuxAI's configuration system
- TmuxAI handles project understanding and exploration
- TmuxAI delegates precise file editing to Aider (which TmuxAI cannot do directly)
- TmuxAI executes non-interactive Aider commands as it would any external program
- TmuxAI verifies changes through tests and builds

## Configuration

TmuxAI is highly configurable through:

1. **Config File**: Located at `~/.config/tmuxai/config.yaml`
2. **Environment Variables**: Using the `TMUXAI_` prefix
3. **Runtime Commands**: Using the `/config` command

Key configuration options include:
- `agentic_mode`: Enables interaction with all panes
- `max_capture_lines`: Controls how much scrollback history is sent as context
- `openrouter.model`: Specifies which AI model to use
- Various confirmation settings for commands, keystrokes, and file operations

## Commands

TmuxAI provides several commands for user interaction:

- `/info`: Display system information and context statistics
- `/clear`: Clear chat history
- `/reset`: Clear chat history and reset all panes
- `/config`: View or modify configuration settings
- `/squash`: Manually trigger context summarization
- `/prepare`: Toggle a pane into 'prepared' mode
- `/watch`: Enable Watch Mode with a specified goal
- `/exit`: Exit TmuxAI

## Future Development

According to the project roadmap (PLAN.md), future developments include:

1. **Enhanced External Tool Orchestration**: Improving how TmuxAI works with external tools like Aider (while maintaining their separation as independent projects) for better code editing capabilities.
2. **Session Persistence**: Addressing the issue of conversation history being lost on exit.
3. **Improved Tool-Calling**: Moving away from XML tag parsing to more robust native tool-calling features provided by modern AI APIs.
4. **Automated Background Processing**: Enhancing background task handling.

## Conclusion

TmuxAI represents a novel approach to AI assistance in the terminal environment, focusing on non-intrusive integration with the user's existing workflow. By observing and understanding the terminal context, TmuxAI provides relevant assistance without requiring the user to switch contexts or learn new tools. 

TmuxAI's ability to work with external tools like Aider (through configuration only, not code integration) further enhances its capabilities. This allows TmuxAI to leverage specialized tools for tasks it cannot perform directly (like code editing) while maintaining its core philosophy of being a helpful pair programmer that works alongside the user.

It's important to emphasize that TmuxAI and Aider remain completely separate, standalone projects developed independently. Their connection is established solely through TmuxAI's configuration system, which instructs TmuxAI on how to execute Aider commands as it would any external program.