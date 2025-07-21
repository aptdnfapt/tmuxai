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

Here is an example of what the AI receives. Notice how it's broken into clear sections with status markers.

```
----CURRENT-TIME----
Date: Mon, 21 Jul 2025 19:00:04 UTC
----END-OF-CURRENT-TIME----

----REPO-MAP----
[UNCHANGED since message 1]
----END-OF-REPO-MAP----

----FILES----
file: /path/to/main.go [UPDATED] (last modified: 7:00PM)
[... new file content ...]

file: /path/to/config.yaml [UNCHANGED since message 3]
----END-OF-FILES----

----PANES----
pane: %8 [UPDATED] (last updated: 7:00PM)
[... old pane content ...]
----NEW-CONTENT----
[... new lines that appeared in the pane ...]
----END-OF-NEW-CONTENT----

pane: %11 [UNCHANGED since message 2]
----END-OF-PANES----

----OLD-SESSION-DATA----
[Restored from: "feature-branch" - saved: Sun, 20 Jul 2025 14:30:00 UTC]
... (content from a restored session) ...
----END-OF-OLD-SESSION-DATA----

----CONVERSATION----
... (the user's actual message) ...
----END-OF-CONVERSATION----
```

### How to Interpret the Format

The system prompt explains this format to the AI:

-   **`----SECTION----`**: Delimits different types of context (e.g., `PANES`, `FILES`).
-   **`[NEW]`**: The first time an item (like a file or pane) is seen.
-   **`[UPDATED]`**: The item has changed since the last message. The new content is provided.
-   **`[UNCHANGED since message X]`**: The item has not changed. The AI is instructed to use its memory of the content from message number `X`. **This is the primary mechanism for saving tokens.**
-   **`[REMOVED]`**: An item has been closed or deleted.
-   **`----NEW-CONTENT----`**: For updated panes, this special block highlights only the new lines that have appeared, making it easy for the AI to see what just happened.
-   **`----OLD-SESSION-DATA----`**: When a session is restored, the context from the previous session is loaded into this block as a read-only reference.
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
