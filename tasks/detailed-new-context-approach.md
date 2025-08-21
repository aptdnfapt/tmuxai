# TmuxAI New Context Approach: Simplified State Management

## Overview

This document provides a detailed specification for implementing a simpler, more efficient approach to TmuxAI's context management. This approach eliminates complex change tracking while reducing context bloat and improving reliability.

## Current Problem Analysis

The current implementation in `tasks/context-refactor/` uses complex change tracking with status markers:

### Issues with Current Approach:
1. **Complexity**: Requires maintaining hash-based change detection for repo maps, files, and panes
2. **Edge Cases**: Difficult to handle when content is partially changed or when panes are updated incrementally
3. **Maintenance Overhead**: Complex codebase with multiple trackers and state managers
4. **Potential Bugs**: Change detection algorithms can fail in edge cases
5. **Token Inefficiency**: Still sends status markers and references that consume tokens

### Current Implementation Details:
- Uses `ContextStateTracker` to track changes with SHA256 hashing
- Maintains `SectionState` with `PreviousContent`, `Content`, `Hash`, `Status`, etc.
- Sends markers like `[UNCHANGED since message X]`, `[UPDATED]`, `[NEW]`
- Implements complex diff algorithms for pane content

## New Approach: Fresh State Each Call

Instead of tracking changes, we send the complete current state with each API call:

### Key Principles

1. **Each API call is self-contained** - contains all necessary information
2. **Send fresh complete state** - no historical diffs or change markers
3. **System prompt guides interpretation** - AI understands the workflow
4. **Conversation history for reference** - maintains context of previous interactions

### What Gets Sent in Each API Call

**Each API Call Contains:**
- **System Prompt** (instructions on how TmuxAI works)
- **Repo Map** (complete current project structure)
- **Files** (complete current content of relevant files)
- **Pane Content** (complete current output from all panes)
- **Conversation History** (previous user messages and AI responses)
- **New User Message**

### Detailed Implementation

#### 1. Message Structure

The new message structure will be simplified:

```
[System Prompt Instructions]
====
[Repo Map]
Project structure:
.
├── src/
│   ├── main.go
│   └── utils.go
├── config/
│   └── config.yaml
└── README.md
====
[Files]
File: src/main.go
```go
package main

import "fmt"

func main() {
    fmt.Println("Hello World")
}
```

File: config/config.yaml
```yaml
debug: true
max_lines: 100
```
====
[Pane Content]
Pane %1:
$ go run main.go
Hello World

Pane %2:
$ ls -la
total 24
drwxr-xr-x  4 user  staff  128 Jan  1 12:00 .
drwxr-xr-x  3 user  staff   96 Jan  1 12:00 ..
-rw-r--r--  1 user  staff  156 Jan  1 12:00 main.go
====

[Previous Session Summary]
Session 'feature-login' (closed 2 hours ago):
- Implemented user authentication system
- Added login/logout endpoints
- Created user database schema
- Fixed 3 authentication bugs
- Ran successful integration tests
====

[Conversation History]
User: Can you help me run this Go program?
AI: I can see your Go program. Let me run it for you.
<ExecCommand>
<command>go run main.go</command>
<pane_id>%1</pane_id>
</ExecCommand>
====
[New User Message]
User: Great! Now can you show me the file structure?
```

#### 2. System Prompt Updates

The system prompt will be updated to include clear instructions:

```text
You are TmuxAI, an intelligent terminal assistant that lives inside tmux sessions.

WORKFLOW INSTRUCTIONS:
1. Each message you receive contains the COMPLETE CURRENT STATE of the user's environment
2. The "Pane Content" section shows the current output from all panes
3. The "Files" section shows the current content of relevant files
4. The "Repo Map" section shows the current project structure
5. The "Conversation History" section shows previous interactions
6. Check the current state to see if your previous suggestions were implemented
7. Based on the current state and conversation history, provide your next response

CONTEXT MANAGEMENT:
- You will receive fresh complete state with each request
- Do not rely on memory of previous states - always check the current attachments
- The conversation history is for reference only - the current state is what matters
- If you suggested a command in the past, check the current pane content to see if it was executed
```

#### 3. Code Changes Required

##### A. Remove Obsolete Components

**Files to Remove/Modify:**
1. `internal/context_tracker.go` - Remove entirely
2. `internal/message_builder.go` - Rewrite with simplified logic
3. Parts of `internal/types.go` - Remove `ItemStatus`, `SectionState`, `ContextState`, `OldSessionData`
4. `internal/process_message.go` - Simplify message construction

##### B. Simplified Message Builder

**New `internal/message_builder.go`:**

```go
package internal

import (
    "bytes"
    "fmt"
    "strings"
    "time"
)

// SimplifiedMessageBuilder creates messages with fresh state
type SimplifiedMessageBuilder struct {
    Manager *Manager
}

// NewSimplifiedMessageBuilder initializes a new message builder
func NewSimplifiedMessageBuilder(manager *Manager) *SimplifiedMessageBuilder {
    return &SimplifiedMessageBuilder{Manager: manager}
}

// BuildMessage constructs the simplified message
func (b *SimplifiedMessageBuilder) BuildMessage(userInput string) string {
    var buf bytes.Buffer
    
    // System Prompt
    buf.WriteString(b.buildSystemPrompt())
    buf.WriteString("====\n")
    
    // Repo Map
    buf.WriteString(b.buildRepoMap())
    buf.WriteString("====\n")
    
    // Files
    buf.WriteString(b.buildFiles())
    buf.WriteString("====\n")
    
    // Pane Content
    buf.WriteString(b.buildPaneContent())
    buf.WriteString("====\n")
    
    // Old Session Summary (AI-generated summary of previous sessions)
    buf.WriteString(b.buildOldSessionSummary())
    buf.WriteString("====\n")
    
    // Conversation History
    buf.WriteString(b.buildConversationHistory())
    buf.WriteString("====\n")
    
    // New User Message
    buf.WriteString(fmt.Sprintf("User: %s\n", userInput))
    
    return buf.String()
}

func (b *SimplifiedMessageBuilder) buildSystemPrompt() string {
    // Return updated system prompt with workflow instructions
    return b.getSystemPrompt()
}

func (b *SimplifiedMessageBuilder) buildRepoMap() string {
    repoMap, _ := b.Manager.RepoMap.GetMap()
    return fmt.Sprintf("[Repo Map]\n%s", repoMap)
}

func (b *SimplifiedMessageBuilder) buildFiles() string {
    var buf bytes.Buffer
    buf.WriteString("[Files]\n")
    
    for _, filePath := range b.Manager.ReadFiles {
        content, err := readFileContent(filePath)
        if err == nil {
            buf.WriteString(fmt.Sprintf("File: %s\n%s\n\n", filePath, content))
        }
    }
    
    return buf.String()
}

func (b *SimplifiedMessageBuilder) buildPaneContent() string {
    var buf bytes.Buffer
    buf.WriteString("[Pane Content]\n")
    
    panes, _ := b.Manager.GetTmuxPanes()
    for _, pane := range panes {
        if pane.IsTmuxAiPane {
            continue
        }
        paneContent, _ := system.TmuxCapturePane(pane.Id, b.Manager.GetMaxCaptureLines())
        buf.WriteString(fmt.Sprintf("Pane %s:\n%s\n\n", pane.Id, paneContent))
    }
    
    return buf.String()
}

func (b *SimplifiedMessageBuilder) buildOldSessionSummary() string {
    var buf bytes.Buffer
    buf.WriteString("[Previous Session Summary]\n")
    
    // When a session is opened for the second time, generate an AI summary
    // of what was accomplished in the previous session instead of sending raw data
    if b.Manager.HasPreviousSession() {
        summary := b.Manager.GetPreviousSessionSummary()
        buf.WriteString(summary)
    }
    
    return buf.String()
}

func (b *SimplifiedMessageBuilder) buildConversationHistory() string {
    var buf bytes.Buffer
    buf.WriteString("[Conversation History]\n")
    
    for _, msg := range b.Manager.Messages {
        role := "AI"
        if msg.FromUser {
            role = "User"
        }
        buf.WriteString(fmt.Sprintf("%s: %s\n", role, msg.Content))
    }
    
    return buf.String()
}
```

##### C. Simplified Process Message

**Updated `internal/process_message.go`:**

```go
func (m *Manager) ProcessUserMessage(ctx context.Context, message string) bool {
    // Check if context management is needed before sending
    if m.needSquash() {
        m.Println("Exceeded context size, squashing history...")
        m.squashHistory()
    }

    s := spinner.New(spinner.CharSets[26], 100*time.Millisecond)
    s.Start()

    // Create simplified message builder
    messageBuilder := NewSimplifiedMessageBuilder(m)
    
    // Build the simplified message
    simplifiedMessage := messageBuilder.BuildMessage(message)
    
    // Create the ChatMessage with the simplified content
    currentMessage := ChatMessage{
        Content:   simplifiedMessage,
        FromUser:  true,
        Timestamp: time.Now(),
    }

    // build current chat history
    var history []ChatMessage
    switch {
    case m.WatchMode:
        history = []ChatMessage{m.watchPrompt()}
    case m.GetAgenticMode():
        history = []ChatMessage{m.agenticPrompt()}
    default:
        history = []ChatMessage{m.chatAssistantPrompt(m.ExecPane.IsPrepared)}
    }

    history = append(history, m.Messages...)
    sending := append(history, currentMessage)

    response, err := m.AiClient.GetResponseFromChatMessages(ctx, sending, m.GetOpenRouterModel())
    // ... rest of the function remains the same
}
```

#### 4. Data Structure Changes

**Updated `internal/types.go`:**

Remove the following types:
- `ItemStatus` enum
- `SectionState` struct
- `ContextState` struct
- `OldSessionData` struct

Keep only the essential types needed for the simplified approach.

#### 5. Old Session Data Enhancement

To reduce context bloat from previous sessions, instead of sending raw session data:

1. **AI-Generated Summaries**: When a session is opened for the second time, generate a concise AI summary of what was accomplished in the previous session
2. **Summary Placement**: Include this summary in a dedicated `[Previous Session Summary]` block in the message structure
3. **Token Efficiency**: Summaries are typically 50-100 tokens vs. potentially thousands of tokens for raw session data
4. **Contextual Value**: Summaries provide high-level context about previous work without overwhelming detail

Example summary format:
```
[Previous Session Summary]
Session 'feature-login' (closed 2 hours ago):
- Implemented user authentication system
- Added login/logout endpoints
- Created user database schema
- Fixed 3 authentication bugs
- Ran successful integration tests
```

This approach reduces context size by 80-95% for sessions with significant history while still providing valuable context to the AI.

#### 6. System Prompt Updates

Update the system prompts in `internal/prompts.go` to include the new workflow instructions as shown in the example above.

### Benefits of This Approach

1. **Simplified Implementation** - No complex change tracking algorithms
2. **Eliminated Context Bloat** - No repeated historical diffs or status markers
3. **More Reliable** - AI always has complete current information
4. **Easier to Debug** - Clear what state is being sent each time
5. **Aligned with AI Behavior** - Matches how web-based AI assistants work
6. **Reduced Code Complexity** - Fewer components to maintain
7. **Better Performance** - No hashing or diff calculations
8. **Eliminates Edge Cases** - No complex state management logic

### Implementation Steps

#### Phase 1: Foundation
1. Create new simplified message builder
2. Update system prompts with workflow instructions
3. Modify process message to use new builder

#### Phase 2: Component Removal
1. Remove context tracker code
2. Remove obsolete data structures
3. Clean up dependencies

#### Phase 3: Testing
1. Test with simple scenarios
2. Verify token usage reduction
3. Ensure AI behavior is consistent

#### Phase 4: Optimization
1. Performance tuning
2. Edge case handling
3. Documentation updates

### Expected Results

- **Token Reduction**: 80-95% less context duplication
- **Performance**: Faster message processing (no hashing/diff calculations)
- **Reliability**: More consistent AI responses
- **Maintainability**: Simpler codebase with fewer components
- **Developer Experience**: Easier to understand and modify

### Migration Plan

1. **Backup Current Implementation**: Keep the current change tracking code in a branch
2. **Implement New Approach**: Build the simplified system alongside current code
3. **Testing**: Thoroughly test the new approach with various scenarios
4. **Gradual Rollout**: Start with non-critical features
5. **Full Deployment**: Replace old system once new approach is validated
6. **Cleanup**: Remove obsolete code and documentation

### Risk Mitigation

1. **Fallback Plan**: Keep current implementation as backup
2. **Incremental Implementation**: Build new system alongside old one
3. **Comprehensive Testing**: Test all modes (observe, agentic, watch)
4. **Monitoring**: Track token usage and AI response quality
5. **Rollback Strategy**: Easy switch back to old system if needed

This approach treats each API call as an independent session while maintaining conversation context through chat history, resulting in a cleaner, more robust implementation that's easier to maintain and debug.