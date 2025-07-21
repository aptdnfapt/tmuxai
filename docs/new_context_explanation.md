
# TmuxAI Context Building: A High-Level Explanation

This document explains how TmuxAI efficiently builds the context it sends to the AI model with each request.

## The Problem: Wasted Tokens

Traditional AI assistants often re-send the entire terminal state (all visible content, files, etc.) with every single message. This is highly inefficient and leads to:
-   **High API Costs**: Sending the same unchanged information repeatedly wastes tokens.
-   **Slow Responses**: Large context payloads take longer for the AI to process.
-   **Limited Conversation Length**: The context window fills up quickly with redundant data, forcing the conversation history to be cut short.

## The Solution: Structured and State-Aware Context

TmuxAI solves this problem by being "state-aware." It remembers what context the AI has already seen and only sends updates. This is achieved through a **structured context format** that uses sections and status markers.

### Key Concepts:

1.  **Context State Tracker**: TmuxAI maintains an internal "memory" of the state of your panes, files, and repository map.

2.  **Change Detection**: Before sending a message, TmuxAI compares the current state of your terminal to its memory. It uses hashing to efficiently detect any changes.

3.  **Sending Only What's New**: The message sent to the AI is built intelligently:
    *   **Unchanged Content**: If a pane or file hasn't changed, TmuxAI simply sends a marker like `[UNCHANGED since message 3]`. The AI is instructed to refer to its memory of that item from the previous turn. This is the biggest source of token savings.
    *   **Updated Content**: If something has changed, only the new version is sent, marked as `[UPDATED]`.
    *   **Pane Diffing**: For terminal panes, TmuxAI goes a step further and highlights only the new lines that have appeared, making it easy for the AI to see the result of the last command.

4.  **Session Restoration**: When you restore a previous session, its context (panes, files, conversation) is loaded into a special `----OLD-SESSION-DATA----` block. This gives the AI historical context without mixing it up with the current, active state of your terminal.

By combining these techniques, TmuxAI provides a rich, detailed, and up-to-date context to the AI while using a fraction of the tokens of a traditional approach.
