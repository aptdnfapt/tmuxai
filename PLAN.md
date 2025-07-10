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

### What TmuxAI Has

1.  **Agentic Mode & Multi-Pane Control**:
    *   The `--agentic` flag and `agentic_mode` config (`config/config.go`) are fully functional.
    *   The system prompt for agentic mode (`internal/prompts.go`) instructs the AI to target specific panes via `pane_id` attributes, enabling complex workspace orchestration.
    *   The AI can create new panes on demand using `<CreateExecPane>`.

2.  **Reliable, Marker-Based Command Execution**:
    *   A robust, marker-based system (`TMUXAI:EXITCODE:$?`) is implemented to reliably detect command completion and capture exit codes.
    *   In non-agentic "prepared" mode, the marker is appended automatically for synchronous execution.
    *   In agentic mode, the AI is instructed to append the marker to long-running commands, giving it granular control over synchronous vs. asynchronous execution.

3.  **Efficient Multi-File Reading**:
    *   The `<ReadFile>` tool can accept multiple, space-separated file paths in a single tag (e.g., `<ReadFile>file1.go file2.go</ReadFile>`).
    *   The system performs a single batch confirmation for all requested files, rather than asking for each one individually, improving user experience.
    *   Includes essential safeguards against reading directories, oversized files (`max_read_file_size`), and binary files.

4.  **Automatic Context Management**:
    *   The `squashHistory` feature (`internal/squash.go`) automatically summarizes long conversations when they approach the token limit, preventing errors during long-running tasks.

5.  **User Input History**:
    *   User command-line input is persisted to a history file (`~/.config/tmuxai/history`), providing a standard readline experience.

### What TmuxAI Needs

1.  **Persistent, Directory-Scoped Session History**:
    *   **Problem**: The full conversation state (`m.Messages`) is currently stored only in memory and is lost when `tmuxai` exits, preventing the continuation of complex tasks.
    *   **Required Change**: Implement a persistent session history mechanism.
        *   **Storage**: When `tmuxai --agentic` is run, save the conversation history to a JSON file (e.g., `history.agentic.json`) inside a `.tmuxai_sessions` directory within the project's folder.
        *   **Session Management**: Introduce a new `/session` command to list and load previous conversations, restoring the full context.
        *   **AI-Generated Titles**: Use the AI to generate a concise, descriptive title for the session, to be displayed when listing sessions.
        *   **Automatic Save**: Save the session automatically on exit or if the pane dies to avoid context loss.

2.  **Deeper Aider Integration**:
    *   **Goal**: The primary long-term goal is to enhance `tmuxai` to act as an intelligent orchestrator for `aider`.
    *   **Workflow**: `tmuxai` would manage the high-level workflow (project comprehension, verification), while delegating file editing tasks to `aider`.
    *   **Implementation**: This will require teaching `tmuxai` to construct and execute precise, non-interactive `aider` commands (e.g., `aider --yes --message "..." file1 file2`) and then run verification steps like builds or tests.

