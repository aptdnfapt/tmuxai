# TmuxAI: A Deep Dive into the Intelligent Terminal Assistant

## 1. Executive Summary & Core Philosophy

TmuxAI is a sophisticated, AI-powered terminal assistant designed to integrate seamlessly into a developer's workflow within a `tmux` session. It is not merely a conversational AI for the command line; it is a context-aware "pair programmer" that observes the user's terminal environment, understands the state of their work across multiple panes, and provides assistance by executing commands, reading files, and managing the workspace.

**Core Philosophy:** The project's design is explicitly modeled on human collaboration. It aims to replicate the experience of having a knowledgeable colleague sitting next to you, observing your screen, and offering help. This philosophy is manifested in three key actions:

1.  **Observe:** TmuxAI reads and understands the visible content of all `tmux` panes within the current window, building a rich, real-time context of the user's activities.
2.  **Communicate:** It uses a dedicated chat pane for all user interactions, keeping the conversation separate from the user's primary work panes.
3.  **Act:** With the user's permission, it can execute commands, send keystrokes, and paste content into designated panes, directly manipulating the terminal environment to accomplish tasks.

This approach ensures that TmuxAI is a powerful assistant that enhances, rather than disrupts, the user's established terminal-based workflow.

## 2. System Architecture

TmuxAI is a Go application with a modular architecture designed for clarity, extensibility, and maintainability. The system's components are logically separated, with a central `Manager` coordinating their activities.

```
┌──────────────────┐      ┌─────────────────────┐      ┌─────────────────┐
│                  │      │                     │      │                 │
│   CLI Interface  ├─────▶│       Manager       ├─────▶│    AI Client    │
│ (cli/, chat.go)  │      │    (manager.go)     │      │  (ai_client.go) │
│                  │      │                     │      │                 │
└──────────────────┘      └─────────┬───────────┘      └─────────────────┘
                                    │
           ┌────────────────────────┼────────────────────────┐
           │                        │                        │
           ▼                        ▼                        ▼
┌──────────────────┐      ┌──────────────────┐      ┌──────────────────┐
│                  │      │                  │      │                  │
│  Context System  │      │  Tmux Integration│      │   Session/Config │
│(tracker, builder)│      │    (system/*)    │      │ (session, config)│
│                  │      │                  │      │                  │
└──────────────────┘      └──────────────────┘      └──────────────────┘
```

### 2.1. Core Components

*   **`main.go` & `cli/cli.go`**: The application entry point. It initializes the logger and uses the `cobra` library to define the command-line interface, parse flags (`--agentic`, `--restore`, `--file`, `--layout`), and handle initial user input. It is responsible for creating and starting the `Manager`.

*   **`internal/manager.go`**: The heart of the application. The `Manager` struct is the central coordinator. It holds the application's state, including the configuration, the AI client, pane details, message history, and the new context management system. It orchestrates the entire data flow, from processing user input to executing AI-generated actions.

*   **`internal/ai_client.go`**: This component is responsible for all communication with the backend Large Language Model (LLM). It formats requests into the JSON structure expected by OpenAI-compatible APIs (like OpenRouter, the default), sends HTTP requests, and parses the responses. It includes robust error handling and debug logging for API interactions.

*   **`internal/chat.go` & `internal/prompt_editor.go`**: These files manage the user-facing interface. `chat.go` contains the main loop for reading user input, while `prompt_editor.go` uses the `bubbletea` library to provide a pleasant, multiline text area for composing prompts, with familiar keybindings for submitting or canceling.

*   **`system/` package**: This directory contains all the low-level logic for interacting with the `tmux` environment and the underlying operating system.
    *   `tmux.go`: A crucial wrapper around the `tmux` command-line tool. It provides Go functions for actions like capturing pane content, sending commands/keystrokes, creating/killing panes, and getting details about the current session layout.
    *   `utils.go`: Contains various helper functions, including a sophisticated `HighlightCode` function using the `chroma` library for syntax highlighting in the terminal, and functions to determine OS details and process arguments.
    *   `cosmetics.go`: Handles the formatting of AI messages, applying ANSI color codes to render markdown-style inline code and code blocks beautifully in the terminal.

### 2.2. Context Management Sub-system

This is one of the most advanced features of TmuxAI. It's designed to solve the problem of token waste and context limitations in long conversations.

*   **`internal/types.go`**: Defines the foundational data structures for the new context system, including `ContextState`, `SectionState`, `OldSessionData`, and the `ItemStatus` enum (`[NEW]`, `[UPDATED]`, etc.). These structures are the blueprint for the application's "memory".

*   **`internal/context_tracker.go`**: The "brain" of the context system. The `ContextStateTracker` is responsible for maintaining the state of all context items (panes, files, repo map). Before each turn, it captures the current state of the terminal and compares it to its stored version using SHA256 hashes to efficiently detect changes.

*   **`internal/message_builder.go`**: The "voice" of the context system. The `StructuredMessageBuilder` takes the state information from the `ContextTracker` and constructs the final, highly structured message payload to be sent to the AI. It assembles the different sections (`----FILES----`, `----CURRENT-SESSION-DATA----`, etc.) and applies the correct status markers and timestamps.

*   **`internal/prompts.go`**: This file contains the all-important system prompts. The `baseSystemPrompt` is particularly critical as it contains a detailed explanation of the structured context format, teaching the AI how to interpret the state-aware messages it receives.

### 2.3. Configuration and Session Management

*   **`config/config.go`**: Manages application configuration using the `viper` library. It defines the `Config` struct and handles loading settings from a YAML file (`~/.config/tmuxai/config.yaml`), environment variables (prefixed with `TMUXAI_`), and provides sensible defaults.

*   **`internal/session.go`**: Implements session persistence. It allows the conversation history, list of read files, and pane states to be saved to a JSON file within a project-specific `.tmuxai/` directory. It also handles listing and restoring previous sessions, ensuring that context is not lost between runs.

*   **`internal/session_list.go`**: Provides the TUI for the `/session` command. It uses `bubbletea` to create an interactive list that allows the user to browse and select a session to restore.

## 3. Deep Dive: Key Features & Implementation

### 3.1. The Structured Context System: A Revolution in Efficiency

The cornerstone of TmuxAI's intelligence and efficiency is its state-aware context management system. This system was designed to address the fundamental limitations of stateless LLM interactions: high token usage, slow response times, and limited conversation length due to redundant context.

**The Problem:** A naive terminal assistant would send the entire visible content of all panes and read files with every single user message. In a 10-message conversation, the same unchanged file content would be sent 10 times, wasting thousands of tokens and filling the context window with duplicated data.

**The TmuxAI Solution:**

The implementation, detailed in the `tasks/context-refactor` markdown files, revolves around tracking state and sending only deltas, guided by the `ContextStateTracker` and `StructuredMessageBuilder`.

**Step-by-Step Data Flow:**

1.  **State Capture (`process_message.go`)**: When the user sends a message, `ProcessUserMessage` first calls the `ContextTracker`. The tracker captures the current content of every pane (`system.TmuxCapturePane`) and re-reads every file currently in context (`os.ReadFile`).

2.  **Change Detection (`context_tracker.go`)**: For each context item (a pane, a file, etc.), the `updateSection` function is called. It computes a SHA256 hash of the new content.
    *   If the item has no previous hash, its status is set to `StatusNew`.
    *   If the hash differs from the stored hash, its status becomes `StatusUpdated`. The old content is preserved in `PreviousContent` to enable diffing.
    *   If the hashes match, the status is `StatusUnchanged`. No content update is needed.

3.  **Message Construction (`message_builder.go`)**: The `BuildMessage` function assembles the final payload.
    *   It iterates through the state provided by the tracker.
    *   For `[UPDATED]` panes, it cleverly identifies only the new content that has been appended since the last turn and wraps it in `___NEW-CONTENT___` blocks. This "rolling diff" is extremely token-efficient.
    *   For `[UNCHANGED]` items, it still sends the full content but includes a marker like `[UNCHANGED since message 3]`. This is crucial: the AI is stateless and needs the full context, but the marker provides metadata for the AI to understand the timeline.
    *   If a session was restored, the historical context is neatly packaged into a read-only `----OLD-SESSION-DATA----` block, preventing confusion with the live session.

4.  **AI Instruction (`prompts.go`)**: The `baseSystemPrompt` explicitly teaches the AI how to read this format. It explains the meaning of each section header, status marker, and the `___NEW-CONTENT___` block. This instruction is vital for the AI to correctly interpret the state-aware context.

This entire system transforms TmuxAI from a simple chatbot into an assistant with a persistent, evolving understanding of the user's workspace, all while being incredibly efficient with API costs.

### 3.2. Operation Modes: Tailoring AI Behavior

TmuxAI offers several operation modes, each governed by a different system prompt fragment from `internal/prompts.go`, which alters the AI's behavior and capabilities.

*   **Observe Mode (Default):** The standard mode. The AI observes all panes but can only execute commands in a single, designated `ExecPane`. It uses a countdown timer (`internal/countdown.go`) after asynchronous commands to wait for output.

*   **Prepare Mode:** An enhancement for synchronous command execution. When a user runs `/prepare`, the pane is flagged internally (`m.PreparedPanes`). In `process_message.go`, when an `<ExecCommand>` is detected for a prepared pane, the application *itself* appends a unique marker command (e.g., `; echo "tmuxai waiting for command id: 12345 exitcode:$?"`). The `ExecWaitCapture` function (`internal/exec_pane.go`) then polls the pane content, waiting for this specific marker to appear. This is far more reliable than a fixed timer and captures the command's true exit code.

*   **Agentic Mode:** The most powerful mode. Enabled by `agentic_mode: true` or the `--agentic` flag. The `agenticPrompt` provides the AI with a more extensive set of tools and rules. The key difference is the ability to target *any* pane by its ID (e.g., `<ExecCommand pane_id="%2">`). The prompt also instructs the AI to decide for itself whether to wait for a command to complete by using the `wait="true"` attribute in its tool call.

*   **Watch Mode:** A proactive monitoring mode, activated with `/watch <prompt>`. The `startWatchMode` function (`internal/process_message.go`) enters a loop where it periodically captures pane content and sends it to the AI with the specialized `watchPrompt`. This prompt instructs the AI to be concise, comment only when valuable, and use the `<NoComment>1</NoComment>` tag to remain silent if nothing noteworthy has occurred.

### 3.3. AI Interaction: XML-Based Tool Calling

TmuxAI communicates actions to and from the AI using a custom XML-like tag format. This is a robust implementation of "function calling" that works across any text-based LLM.

*   **Response Parsing (`internal/process_response.go`):** The `parseAIResponse` function is a sophisticated parser that uses regular expressions to find and extract action tags from the AI's response string. It can handle attributes (like `pane_id` and `wait`), multiline content, and multiple instances of the same tag. It cleanly separates the user-facing message from the machine-readable tool calls.

*   **Tool Schema (`internal/prompts.go`):** The agentic and chat assistant prompts clearly define the "schema" for these tools. They provide examples of how to use `<ExecCommand>`, `<TmuxSendKeys>`, `<PasteMultilineContent>`, and `<ReadFile>`, ensuring the AI generates valid and predictable tool calls.

*   **Guideline Enforcement (`internal/process_message.go`):** After parsing, the `aiFollowedGuidelines` function acts as a validator. It checks for common AI mistakes, such as using multiple mutually exclusive tags (e.g., `<RequestAccomplished>` and `<ExecCommand>` in the same response). If the response is invalid, it sends a corrective message back to the AI, prompting it to try again. This self-correction loop significantly improves reliability.

### 3.4. Seamless External Tool Integration (Aider)

A key design principle of TmuxAI is its ability to orchestrate external tools without having hardcoded dependencies on them. The integration with `aider` is a prime example of this philosophy.

*   **Configuration, Not Code:** There is no Go code in TmuxAI that mentions "aider". The entire integration is achieved through the `agentic_prompt_file` configuration (`config.example.yaml`). Users point this to a markdown file like `aider-tmux-agentic-prompts/aider_agentic_prompt.md`.

*   **Instruction-Based Orchestration:** This prompt file effectively "teaches" the AI how to be a project manager for `aider`. It instructs the AI to follow a specific workflow:
    1.  First, use TmuxAI's native tools (`<ReadFile>`, `<ExecCommand>tree`) to understand the project.
    2.  Then, formulate a plan.
    3.  Finally, construct and execute a non-interactive `aider` command (`aider --yes --message "..." file1.go file2.go`) to perform the code editing.
    4.  After `aider` finishes, use TmuxAI's tools again to run `go test` or `npm run build` to verify the changes.

This powerful pattern allows TmuxAI to remain a generic, standalone tool while being infinitely extensible. Users can create their own prompt files to integrate TmuxAI with any other CLI tool, from `kubectl` to `docker-compose`, simply by teaching the AI how to use them.

### 3.5. Project-Aware Context with `RepoMap`

In Agentic Mode, TmuxAI can provide the AI with a high-level map of the entire codebase, similar to `aider`'s own repository map feature.

*   **Automatic and Git-Aware (`internal/repomap.go`):** When `NewRepoMapHandler` is initialized, it first finds the root of the Git repository. The feature is only enabled if it's a Git project. It then uses `git ls-files` to get a list of all tracked files, which is more accurate and efficient than a simple directory scan as it respects `.gitignore`.

*   **Symbol Extraction with `ctags`:** The handler calls the `ctags` command-line tool (a prerequisite) on the list of tracked files. `ctags` analyzes the source code and generates a structured (JSON) list of all symbols (functions, classes, types, etc.) in the project.

*   **Efficient Caching:** The generated map and the modification times of all source files are cached in `.tmuxai/repomap.json`. On subsequent runs, the handler quickly checks the file modification times. If nothing has changed, the cached map is used instantly. If any file has been modified, the map is regenerated. This ensures the AI always has an up-to-date overview without expensive re-computation on every message.

This feature provides the AI with invaluable architectural context, allowing it to make more intelligent decisions about which files are relevant to a user's request.

## 4. A User Request: The Full Lifecycle

Let's trace a typical user request to see how these components work together.

**Scenario:** In Agentic Mode, the user asks, "Read `main.go` and then list the contents of the `internal` directory."

1.  **Input (`prompt_editor.go`):** The user types the message into the `bubbletea` text area and hits Enter.
2.  **Command Check (`chat.go`):** `processInput` determines this is not a `/` command and passes the raw string to the manager.
3.  **Context Gathering (`process_message.go`):**
    *   `ContextTracker.IncrementMessageCount()` advances the turn counter.
    *   The tracker captures the current content of all panes. Let's say one pane was updated.
    *   The tracker re-reads any files already in context (none yet).
    *   The tracker gets the latest `RepoMap`.
4.  **State Update (`context_tracker.go`):**
    *   The hash of the updated pane's content is different. Its status is set to `UPDATED`.
    *   The hashes for other panes and the `RepoMap` match. Their status is `UNCHANGED`.
5.  **Message Building (`message_builder.go`):**
    *   `BuildMessage` is called.
    *   It constructs the payload:
        *   `----PROMPTS----` section with the agentic prompt.
        *   `----REPO-MAP----` section with `[UNCHANGED since message X]`.
        *   `----FILES----` section is empty for now.
        *   `----CURRENT-SESSION-DATA----` section contains all panes. The updated pane is marked `[UPDATED]` with its `___NEW-CONTENT___`, and others are marked `[UNCHANGED]`.
        *   The user's message is added to the `[CURRENT CHAT HISTORY]`.
6.  **API Call (`ai_client.go`):**
    *   The manager assembles the final list of messages (system prompt + history + current message).
    *   `AiClient.GetResponseFromChatMessages` sends the payload to the LLM.
7.  **Response (`ai_client.go`):** The AI responds with:
    ```
    Okay, I will first read the main.go file and then list the contents of the internal directory.
    <ReadFile>main.go</ReadFile>
    ```
8.  **Parsing (`process_response.go`):**
    *   `parseAIResponse` extracts the `ReadFile` tag and the user-facing message.
9.  **First Action - ReadFile (`process_message.go`):**
    *   The manager sees the `ReadFile` request. It validates the file path and checks permissions (`internal/read_file.go`).
    *   It reads `main.go` and adds its content to the context via `ContextTracker.UpdateFile("main.go", content)`.
    *   Because `ReadFile` is a synchronous action that requires a new turn, it immediately re-triggers `ProcessUserMessage` with a new, simple prompt: "I have read the file(s): main.go. What is the next step?".
10. **Second Turn - Context Gathering:**
    *   The process repeats. This time, when `ContextTracker` runs, it sees `main.go` as a `[NEW]` file.
    *   The `StructuredMessageBuilder` includes the full content of `main.go` in the `----FILES----` section.
11. **Second Turn - API Call:** The new payload, now including the file content, is sent to the AI.
12. **Second Turn - Response:** The AI, having seen the file, now proceeds to the next step:
    ```
    Now that I've reviewed main.go, I'll list the contents of the internal directory.
    <ExecCommand>ls -l internal/</ExecCommand>
    ```
13. **Second Action - ExecCommand (`process_message.go`):**
    *   `parseAIResponse` extracts the `ExecCommand` tag.
    *   The manager confirms the command (if `exec_confirm: true`).
    *   The command is sent to the default exec pane via `system.TmuxSendCommandToPane`.
    *   Since this is an asynchronous action (no `wait="true"`), the manager proceeds to the countdown timer (`internal/countdown.go`).
14. **Waiting and Final Turn:**
    *   After the countdown, the manager re-triggers `ProcessUserMessage` with a generic prompt like "Ok, that's done. Here is the current pane content, what's next?".
    *   The `ContextTracker` will capture the output of the `ls` command in the exec pane, marking it as `UPDATED`. This new output is sent to the AI, completing the full cycle.

## 5. Noteworthy Implementation Details

*   **Confirmation & Security (`internal/confirm.go`):** Before executing potentially dangerous commands, `confirmedToExec` is called. It first checks the command against user-configurable `whitelist_patterns` and `blacklist_patterns` from `config.yaml`. If not automatically allowed, it presents an interactive `[Y]es/No/Edit` prompt using the `readline` library for a good user experience.

*   **Terminal Cosmetics (`system/cosmetics.go`):** AI responses are piped through the `Cosmetics` function, which uses regular expressions to find markdown code blocks and inline code. It then uses the `chroma` library to apply syntax highlighting to code blocks and custom ANSI escape codes to style inline code, making the output far more readable than plain text.

*   **Robust Logging (`logger/logger.go`):** The application features a singleton logger that writes to `~/.config/tmuxai/tmuxai.log`. It supports different levels (Info, Error, Debug). Crucially, if `debug: true` is set, `ai_client.go` calls `debugChatMessages` to write the *entire* request and response payload to a timestamped file in `~/.config/tmuxai/debug/`, which is invaluable for debugging prompts and AI behavior.

*   **Graceful Panics and Shutdown:** The entry point in `main.go` wraps the application start in a function that includes `defer instance.Close()` for the logger. This ensures that even if the application panics or exits unexpectedly, the log buffer is flushed to disk, preserving valuable debugging information.

This comprehensive analysis showcases TmuxAI as a well-architected, highly efficient, and extensible AI assistant that deeply understands and integrates with the developer's terminal environment, setting a high standard for context-aware AI tools.
