
# TmuxAI Context Building Explained

This document explains how TmuxAI builds the context it sends to the AI model with each request. Understanding this process is crucial for customizing prompts and interpreting the AI's behavior.

The context is constructed in a specific order, with each new piece of information appended to the previous one. This creates a comprehensive snapshot of the user's environment and the ongoing conversation.

## 1. The Base System Prompt

The context begins with the **base system prompt**. This prompt provides the AI with its core instructions, defining its role, capabilities, and constraints. It sets the foundation for all interactions.

**Key elements of the base system prompt:**

*   **Identity:** "You are TmuxAI assistant."
*   **Core Function:** Explains that the AI lives in the user's tmux window, can see all panes, and can execute commands.
*   **Rules:** Sets high-priority rules, such as using common sense, preferring shell commands, and using the `<ReadFile>` tag instead of `cat`.
*   **Conciseness:** Emphasizes the importance of being concise and avoiding verbosity.

## 2. Agentic or Chat Assistant Prompt

Next, a more specific prompt is added based on the current mode:

*   **Agentic Mode:** This prompt provides instructions for the AI to act as an autonomous agent. It details the pane targeting system, the use of XML tags for actions (`<ExecCommand>`, `<TmuxSendKeys>`, etc.), and the rules for combining them.
*   **Chat Assistant Mode:** This prompt is used for more direct, conversational interactions. It outlines the available tools and provides examples of how to use them.

## 3. The Repository Map (`repomap`)

If the `repomap` feature is enabled, TmuxAI generates a map of the current Git repository. This map is created using `ctags` and provides a high-level overview of the codebase, including file names and the symbols (functions, classes, etc.) within them.

The `repomap` is injected into the context to give the AI a better understanding of the project structure.

## 4. Current Tmux Window State

TmuxAI captures the current state of the tmux window and formats it as an XML-like structure. This includes:

*   **Pane Details:** The ID, title, and content of each pane in the window.
*   **Pane Types:** Panes are categorized as `tmuxai_exec_pane`, `agentic_exec_pane`, or `read_only_pane`.

This information allows the AI to "see" what the user sees and to target specific panes for actions.

## 5. Recently Read Files

When the user asks the AI to read a file using the `<ReadFile>` tag, the content of that file is added to the context for the next turn. This allows the AI to analyze file contents without cluttering the terminal with `cat` output.

## 6. Conversation History

Finally, the history of the current conversation is appended to the context. This includes all previous user messages and AI responses. The conversation history provides the AI with the necessary context to understand the user's intent and to carry on a coherent conversation.

By combining these elements, TmuxAI creates a rich and detailed context that enables the AI to provide accurate, relevant, and helpful assistance.
