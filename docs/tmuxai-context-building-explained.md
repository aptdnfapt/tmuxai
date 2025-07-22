# TmuxAI Context Building: The Structured Context System

This document explains how TmuxAI builds and sends context to the AI model. The system is designed to be highly efficient, minimizing token usage by tracking changes and sending only what's new or updated.

## Overview: Structured, State-Aware Context

Instead of sending the full content of all panes and files with every message, TmuxAI uses a **structured context format**. It maintains a state of what the AI has already seen and only transmits the differences.

This approach has several key benefits:
- **Reduced Token Usage**: Dramatically lowers API costs and speeds up response times.
- **Longer Conversations**: Allows for much longer interactions before hitting the model's context limit.
- **Better Contextual Awareness**: By using status markers and timestamps, the AI can build a more accurate timeline of events in the user's terminal.

## Core Components

The new context system is powered by two main components:

1.  **`ContextStateTracker` (`internal/context_tracker.go`)**: This is the "brain" of the system. It keeps track of the content and status of every piece of context (panes, files, repo map). Before each message, it compares the current state of the terminal and files against its stored state to detect what has changed.

2.  **`StructuredMessageBuilder` (`internal/message_builder.go`)**: This component takes the information from the `ContextStateTracker` and builds the final message payload. It formats the context into sections with clear headers and status markers.

## The Structured Message Format

The message is broken into clear sections. It separates static context (like files), historical context from previous sessions, and the live data from the current session.

```
----REPO-MAP----
[UNCHANGED since message 1]
----END-OF-REPO-MAP----

----FILES----
file: /path/to/main.go [UPDATED] (last modified: Mon, 21 Jul 2025 19:00:04 UTC)
[... new file content ...]

file: /path/to/config.yaml [UNCHANGED since message 3]
----END-OF-FILES----

----OLD-SESSION-DATA----
[Restored from: "feature-branch" - saved: Sun, 20 Jul 2025 14:30:00 UTC]
### [PREVIOUS PANES]
====pane: %1====
$ ls -l
...previous output...
====end of pane %1====

### [PREVIOUS CHAT HISTORY]
User: "What was in the directory?"
AI: "It contained these files..."
----END-OF-OLD-SESSION-DATA----

----CURRENT-SESSION-DATA----
[CURRENT TIME: Mon, 21 Jul 2025 19:00:04 UTC]

====pane: %1 [UPDATED]====
$ ls -l
...previous output...
___NEW-CONTENT___
$ git status
...git status output...
____END-OF-NEW-CONTENT____
====end of pane %1====

====pane: %11 [UNCHANGED since message 2]====

[CURRENT CHAT HISTORY]
... (the user's actual message) ...
----END-OF-CURRENT-SESSION-DATA----
```

### How to Interpret the Format

The system prompt explains this format to the AI:

-   **`----SECTION----`**: Delimits different types of context.
-   **`[NEW]`**, **`[UPDATED]`**, **`[UNCHANGED since message X]`**, **`[REMOVED]`**: These status markers apply to static context like `REPO-MAP` and `FILES`. `[UNCHANGED]` is the primary mechanism for saving tokens.
-   **`----OLD-SESSION-DATA----`**: When a session is restored, the full pane content and conversation from the previous session are loaded here as a read-only historical reference.
-   **`----CURRENT-SESSION-DATA----`**: This block contains the live, dynamic state of the current terminal session.
-   **`___NEW-CONTENT___`**: Inside a pane in the `CURRENT-SESSION-DATA` block, this special marker contains **only** the new lines that have appeared since the last message. In the next turn, this "new" content will become part of the pane's base content, and a new `___NEW-CONTENT___` block will show the next update. This creates a "rolling" view of the pane's history.
-   **Timestamps**: Each item includes a timestamp of its last modification, helping the AI understand the sequence of events.

## Step-by-Step Context Building Process

This process occurs in `internal/process_message.go:ProcessUserMessage()` for every user input.

1.  **Increment Message Counter**: The `ContextTracker` increments its internal message count. This is used for the `[UNCHANGED since message X]` marker.

2.  **Update Context State**: The tracker updates its knowledge of the environment:
    *   **Repo Map**: It gets the current map of the repository.
    *   **Panes**: It captures the current content of every pane.
    *   **Files**: It re-reads any files that have been added to the context via the `<ReadFile>` tool.

3.  **Detect Changes**: For each item, the `ContextTracker` computes a hash of its new content and compares it to the previous hash.
    *   If the hashes are the same, the item is marked as `UNCHANGED`.
    *   If the hashes differ, the item is marked as `UPDATED`. The old content is saved to allow for diffing.
    *   If the item is new, it's marked as `NEW`.

4.  **Build the Structured Message**: The `StructuredMessageBuilder` takes the state from the tracker and assembles the final message string, applying the correct headers, status markers, and timestamps.

5.  **Assemble Final Payload**: The new structured message is combined with the appropriate system prompt (based on the mode) and the recent conversation history.

6.  **Send to AI**: The final payload is sent to the AI model.

## Key Files Involved

-   **`internal/process_message.go`**: Orchestrates the context building and message sending loop.
-   **`internal/context_tracker.go`**: Manages the state and detects changes in context items.
-   **`internal/message_builder.go`**: Constructs the final structured message string.
-   **`internal/types.go`**: Defines the core data structures (`ContextState`, `SectionState`, etc.).
-   **`internal/prompts.go`**: Contains the `baseSystemPrompt` which explains the structured format to the AI.
