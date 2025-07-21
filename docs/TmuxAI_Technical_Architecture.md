# TmuxAI Technical Architecture

## System Architecture Overview

TmuxAI is built with a modular architecture that separates concerns and allows for flexible configuration. The system is written in Go and uses several key components to provide its functionality.

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│                 │     │                 │     │                 │
│  CLI Interface  │────▶│     Manager     │────▶│    AI Client    │
│                 │     │                 │     │                 │
└─────────────────┘     └────────┬────────┘     └─────────────────┘
                                 │
                                 ▼
                        ┌─────────────────┐
                        │                 │
                        │  Tmux System    │
                        │  Integration    │
                        │                 │
                        └─────────────────┘
```

## Core Components

### 1. Manager (`internal/manager.go`)

The Manager is the central component that coordinates all activities. It:

- Manages the tmux session and panes
- Handles user interactions
- Processes messages to and from the AI
- Executes commands in tmux panes
- Manages the chat history and context

Key structures and methods:
- `Manager` struct: Contains configuration, AI client, pane details, and message history
- `NewManager()`: Initializes the manager with configuration
- `Start()`: Starts the manager and CLI interface
- `ProcessUserMessage()`: Processes user messages and sends them to the AI
- `ContextTracker` and `MessageBuilder`: These components work together to track the state of the terminal and build efficient, structured messages that only send updated content to the AI.

### 2. AI Client (`internal/ai_client.go`)

The AI Client handles communication with the AI service (OpenRouter by default). It:

- Formats messages for the AI service
- Sends requests to the AI service
- Processes responses from the AI service

Key structures and methods:
- `AiClient` struct: Contains configuration for the AI service
- `GetResponseFromChatMessages()`: Sends chat messages to the AI and gets a response
- `ChatCompletion()`: Handles the HTTP request to the AI service

### 3. CLI Interface (`internal/chat.go`)

The CLI Interface provides the user interface for interacting with TmuxAI. It:

- Displays the prompt for user input
- Processes user commands
- Displays AI responses

Key structures and methods:
- `CLIInterface` struct: Contains a reference to the manager
- `Start()`: Starts the CLI interface
- `processInput()`: Processes user input and commands

### 4. Message Processing (`internal/process_message.go`)

This component processes user messages and sends them to the AI. It:

- Captures context from tmux panes
- Formats messages for the AI
- Processes AI responses
- Executes commands based on AI responses

Key methods:
- `ProcessUserMessage()`: Main method for processing user messages
- `aiFollowedGuidelines()`: Validates AI responses against guidelines
- `startWatchMode()`: Starts the watch mode loop

### 5. Response Processing (`internal/process_response.go`)

This component parses and processes AI responses. It:

- Extracts commands, keystrokes, and other actions from AI responses
- Validates responses against guidelines
- Formats responses for display

Key methods:
- `parseAIResponse()`: Parses the AI response into structured data
- `isTrue()`: Helper for parsing boolean values
- `collapseBlankLines()`: Formats response text

### 6. Configuration (`config/config.go`)

The Configuration component handles loading and managing configuration. It:

- Loads configuration from file
- Applies environment variables
- Provides default values

Key structures and methods:
- `Config` struct: Contains all configuration options
- `Load()`: Loads configuration from file and environment variables
- `DefaultConfig()`: Provides default configuration values

### 7. System Integration (`system/`)

The System Integration components handle interaction with the tmux system. They:

- Execute commands in tmux panes
- Capture output from tmux panes
- Create and manage tmux sessions and panes

## Data Flow

1. **User Input → Processing**:
   - User enters a message in the CLI interface.
   - The `Manager`'s `ContextStateTracker` captures the current state of all panes and tracked files.
   - The tracker compares the new state with the previous state to identify changes.
   - The `StructuredMessageBuilder` constructs a message payload containing only the changed content, with appropriate status markers (`[UPDATED]`, `[UNCHANGED]`, etc.).

2. **AI Request → Response**:
   - Manager sends the formatted message to the AI client
   - AI client sends the request to the AI service
   - AI service processes the request and returns a response
   - AI client passes the response back to the manager

3. **Response → Action**:
   - The `Manager` parses the AI response, extracting commands and other actions.
   - It executes the actions in the appropriate tmux panes.
   - On the next user input, the data flow starts again from step 1, capturing the new state resulting from the AI's actions.

## Key Interfaces

### 1. AI Response Format

AI responses are parsed using XML-like tags to extract structured data:

```xml
<ExecCommand pane_id="%1" wait="true">ls -la</ExecCommand>
<TmuxSendKeys pane_id="%2">echo "Hello, world!"</TmuxSendKeys>
<PasteMultilineContent pane_id="%3">
This is
multiline
content
</PasteMultilineContent>
<ReadFile>README.md</ReadFile>
<RequestAccomplished>true</RequestAccomplished>
```

These tags are parsed into the `AIResponse` struct:

```go
type AIResponse struct {
    Message                string
    SendKeys               []SendKeysInfo
    ExecCommand            []ExecCommandInfo
    PasteMultilineContent  []PasteInfo
    ReadFile               []ReadFileInfo
    RequestAccomplished    bool
    ExecPaneSeemsBusy      bool
    WaitingForUserResponse bool
    NoComment              bool
    CreateExecPane         bool
}
```

### 2. Configuration Interface

Configuration is loaded from a YAML file and environment variables:

```yaml
debug: false
agentic_mode: false
editor: nano
max_capture_lines: 200
max_context_size: 20000
wait_interval: 5
send_keys_confirm: true
paste_multiline_confirm: true
exec_confirm: true
read_file_confirm: true
multi_file_read: false
max_read_file_size: 250000
whitelist_patterns: []
blacklist_patterns: []
openrouter:
  api_key: "your-api-key-here"
  model: "google/gemini-2.5-flash-preview"
  base_url: "https://openrouter.ai/api/v1"
prompts:
  base_system: ""
  agentic: ""
  agentic_prompt_file: ""
  chat_assistant: ""
  chat_assistant_prepared: ""
  watch: ""
```

## Operation Modes Implementation

### 1. Observe Mode

The default mode is implemented in `ProcessUserMessage()` in `internal/process_message.go`. It:

1. Captures context from all visible panes
2. Sends the context and user message to the AI
3. Parses the AI response for commands
4. Executes commands with user confirmation
5. Uses a countdown timer before checking for new output
6. Continues the conversation with updated context

### 2. Prepare Mode

Prepare mode is implemented by appending a special marker to commands:

```go
commandID := fmt.Sprintf("%05d", rand.Intn(100000))
markerCommand := fmt.Sprintf(`; echo "%s: %s exitcode:%s"`, endMarkerPrefix, commandID, exitCodeVar)
commandToRun += markerCommand
```

The system then waits for this marker to appear in the pane output:

```go
result, err := m.ExecWaitCapture(targetPane, commandID)
```

### 3. Agentic Mode

Agentic mode is implemented by allowing commands to target specific panes:

```go
for _, execCommand := range r.ExecCommand {
    var targetPane *system.TmuxPaneDetails
    if execCommand.PaneID != "" {
        // Find the pane details for this ID
        // ...
    } else {
        // Default to the primary exec pane
        targetPane = m.ExecPane
    }
    // Execute command in the target pane
    // ...
}
```

### 4. Watch Mode

Watch mode is implemented as a loop that continuously monitors panes:

```go
func (m *Manager) startWatchMode(desc string) {
    // ...
    m.Countdown(m.GetWaitInterval())
    // ...
    accomplished := m.ProcessUserMessage(ctx, desc)
    // ...
    if m.Status != "" && m.WatchMode {
        m.startWatchMode("")
    }
}
```

## External Tool Integration: Aider

TmuxAI can work with Aider (a completely separate, standalone command-line AI coding assistant) to provide code editing capabilities that TmuxAI itself cannot perform directly. This is **not a built-in integration** but rather a configuration-based approach that connects these two separate tools through TmuxAI's agentic prompt system:

1. TmuxAI handles project understanding and exploration
2. Aider (as an external tool) handles precise file editing
3. TmuxAI executes non-interactive Aider commands as it would any external program
4. TmuxAI verifies changes through tests and builds

The workflow when using these separate tools together:

1. **Autonomous Project Exploration**: TmuxAI reads files and executes commands to understand the project structure
2. **Problem Investigation**: TmuxAI analyzes files related to errors/issues
3. **Intelligent Planning**: TmuxAI determines which files need modification
4. **External Tool Execution**: TmuxAI executes Aider commands (as an external program) for precise file operations
5. **Verification & Iteration**: TmuxAI runs tests and builds to verify changes

This approach demonstrates how TmuxAI can leverage external specialized tools without requiring any code-level integration between the separate projects.

## Conclusion

TmuxAI's architecture is designed to be modular, flexible, and extensible. The separation of concerns allows for easy maintenance and enhancement of the system. The ability to work with external tools like Aider (through configuration only, not code integration) demonstrates how TmuxAI can be extended to provide additional capabilities while maintaining its core philosophy of being a non-intrusive pair programmer.
