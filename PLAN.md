# TmuxAI-Aider Orchestration Plan

## I. Executive Summary

The primary goal is to enhance `tmuxai` to act as an intelligent orchestrator for `aider`, a command-line AI coding assistant. In this model, `tmuxai` will manage the high-level
workflow—understanding project structure, managing multiple tmux panes, and running verification commands—while delegating the specific task of file editing to `aider`.

This plan preserves `tmuxai`'s core, non-intrusive philosophy while enabling a powerful, agentic coding workflow. `tmuxai` becomes the "maestro," and `aider` becomes the specialized "file
surgeon."

---

## II. Detailed Workflow

The envisioned end-to-end user experience is as follows:

1.  **Initialization**: The user starts `tmuxai` in a project's root directory with agentic mode enabled (`tmuxai --agentic`).
2.  **Project Comprehension**: The user asks `tmuxai` to understand the project (e.g., "Check out this project and let me know when you're ready").
3.  **Autonomous Exploration**: `tmuxai` explores the project by:
    *   Running commands like `ls -la` and `tree` to understand the file structure.
    *   Reading key files like `README.md`, `go.mod`, `package.json`, etc., to grasp the project's purpose and dependencies. It should be capable of reading multiple files in a single turn to
build context efficiently.
4.  **Task Delegation**: The user provides a high-level task, such as "Add a new endpoint to handle user profiles."
5.  **Planning & Confirmation**: `tmuxai` analyzes the request, potentially reads more specific source files, and forms a plan. It then determines which files need to be created or modified.
6.  **Aider Invocation**: For file modifications, `tmuxai` will construct and execute a precise, **non-interactive** `aider` command.
    *   **Editing**: `aider --yes --message "Here are the detailed changes for file X and file Y..." path/to/fileX path/to/fileY ; echo TMUXAI_CMD_END_CODE something like that to let tmuxai know that aider is running and wait for its eddit .`
    *   **File Creation**: `tmuxai` first creates an empty file (`touch new_feature.go`) and then instructs `aider` to populate it using the same non-interactive method.
7.  **Verification**: After the `aider --yes` command finishes, `tmuxai`'s enhanced execution tracking (using an end-of-command marker) will detect that the command has completed. It can then proceed to run build commands, tests, or linters to verify the changes.
8.  **Iteration**: The user can continue the conversation, asking for further refinements or new tasks, with the full history preserved across sessions.

---

## III. Analysis of Current TmuxAI Capabilities

### What TmuxAI Has (Strengths for this Plan)

1.  **Agentic Mode & Multi-Pane Control**:
    *   The `--agentic` flag and `agentic_mode` config (`config/config.go`) already exist.
    *   The system prompt for agentic mode (`internal/prompts.go`) supports targeting specific panes via `<ExecCommand pane_id="%ID">`, which is crucial for orchestrating commands across a
workspace.
    *   The ability to create new panes with `<CreateExecPane>` fits the workflow of spawning `aider` or verification processes in dedicated panes.

2.  **Enhanced Command Execution & Waiting**:
    *   **Problem**: The current `Prepare Mode`'s method of injecting a custom shell prompt is fragile and often conflicts with user-defined prompts from tools like `oh-my-posh` or `starship`.
    *   **New Solution**: A universal, marker-based system will be implemented to reliably detect command completion and capture exit codes without modifying user prompts.
    *   **Implementation (Non-Agentic Mode)**: When `tmuxai` executes a command, it will automatically append a suffix like `; echo "TMUXAI_CMD_END_CODE=$?"`. The Go backend will then watch for the `TMUXAI_CMD_END_CODE=` marker to determine when the command has finished, and it will parse the exit code. This replaces the need for the `/prepare` command.
    *   **Implementation (Agentic Mode)**: The AI's system prompt will be updated. It will be instructed to intelligently append the `; echo "TMUXAI_CMD_END_CODE=$?"` marker to commands it identifies as long-running or critical (e.g., `sudo apt update`, `go build`, `aider --yes`). For simple, quick commands (like `ls`), it can omit the marker for a faster, asynchronous execution. This gives the AI more granular control over the execution flow.

3.  **File Reading Framework**:
    *   A `<ReadFile>` tool exists and is processed in `internal/read_file.go`.
    *   It includes important safeguards like file size limits, directory checks, and binary file detection.
    *   The AI can now request reading multiple files by providing a space-separated list of paths in a single `<ReadFile>` tag.

4.  **Chat History Persistence (User Input)**:
    *   `internal/chat.go` shows that user command-line input history is persisted to `~/.config/tmuxai/history`. This provides a good user experience for recalling past commands.

5.  **Context Management**:
    *   The `squashHistory` feature in `internal/squash.go` automatically manages context size, which is vital for long-running, complex orchestration tasks to prevent exceeding token limits.

### What TmuxAI Needs (Gaps to Bridge)

1.  **Efficient Multi-File Reading**:
    *   **Status: Implemented.** The `<ReadFile>` tool has been enhanced to accept multiple, space-separated file paths within a single tag (e.g., `<ReadFile>file1.go file2.go</ReadFile>`). This was achieved by updating `internal/process_response.go` to split the file paths from the tag's content.

2.  **Persistent, Directory-Scoped Session History**:
    *   **Problem**: The full conversation state (`m.Messages` in `internal/manager.go`) is currently stored in memory and is lost when `tmuxai` exits. This prevents the continuation of complex, multi-day coding tasks.
    *   **Required Change**: Implement a persistent session history mechanism.
        *   **Storage**: When `tmuxai --agentic` is run, it will record the current working directory. The conversation history (`ChatMessage` slice) will be saved to a JSON file (e.g., `history.agentic.json`) inside a `.tmuxai_sessions` directory within that project's folder. This keeps session data alongside the project it belongs to.
        *   **Session Management Command**: Introduce a new `/session` command.
            *   When `tmuxai` starts in a directory with existing sessions, it will notify the user.
            *   The `/session` command will allow the user to list and load a previous conversation, restoring the full context from all panes and also tmuxai chat.
        *   **AI-Generated Titles**: After 3-4 conversational turns, `tmuxai` will use its underlying AI model to generate a concise, descriptive title for the session (e.g., "Refactoring the user authentication module"). This title will be stored with the session data and displayed when listing sessions. Short or inconclusive conversations will receive a default, timestamp-based title. The title will be updated as the session progresses.
        *   **Automatic Save**: The session history will be saved automatically upon exiting `tmuxai` or if the tmux pane dies .. it will save on real time to avoid context loss.

3.  **Pane-Context-Aware File Reading**:
    *   **Decision**: This is no longer required. The current file reading logic, which resolves paths relative to `tmuxai`'s own working directory, is sufficient for the planned workflow, as the user will typically launch `tmuxai` from the project root.

