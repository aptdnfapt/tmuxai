# How AI Context Building Works in TmuxAI

## The Story of Sending a Message to AI

Imagine you're having a conversation with an AI assistant. Here's exactly what happens when you send a message in TmuxAI:

### Step 1: You Type a Message
You type something like "Show me the files in this directory" and press enter.

### Step 2: Building Your Message Structure
The system takes your message and builds a rich context around it. Think of it like preparing a detailed report:

- **Repository Information**: What files and folders exist in your project
- **File Contents**: Contents of any files you've asked the AI to read
- **Terminal State**: What's currently showing in your terminal panes
- **Your New Message**: Your latest request
- **Special Reminder**: A note reminding the AI to use XML tags

All of this gets packaged into one big structured message.

### Step 3: Creating the Message History
The system also maintains a separate history of your conversation, like index cards:

- **System Instructions**: Rules for how the AI should behave (first card - not shown in debug)
- **Past Messages**: Each "You said..." and "AI replied..." as separate cards
- **Your New Structured Message**: The detailed report from step 2 (final card)

### Step 4: Sending to AI
The AI receives all these "cards" in order:
1. First card: System rules and instructions (not shown in debug)
2. Middle cards: Your conversation history  
3. Last card: The detailed structured report with repository info, file contents, terminal state, and the special reminder

### Step 5: What Debug Shows vs Reality
The debug file shows you the conversation history and current structured message (2nd and 3rd blocks) to keep things simple. But the AI actually receives the full deck of cards including the system prompt.

## Detailed Examples

### Example 1: What the AI Receives (Full Context)

The AI receives multiple message objects:

**Message 1 (System Prompt - NOT shown in debug):**
```
Role: system
Content: You are TmuxAI, an AI programming assistant that helps users with terminal tasks...
```

**Message 2 (Previous User Message):**
```
Role: user
Content: Can you help me understand this project?
```

**Message 3 (Previous AI Response):**
```
Role: assistant
Content: I'd be happy to help! I can see this is a Go project with several files...
<RequestAccomplished>1</RequestAccomplished>
```

**Message 4 (Current Structured Message):**
```
Role: user
Content: [Repo Map]
main.go
internal/
  manager.go
  session.go
config/
  config.go

====

[Files]

====

[Pane Content]
Pane %12 (Command: bash) [PID: 12345]:
~/project $ ls -la
total 24
drwxr-xr-x 4 user user 4096 Jan 1 12:00 .
drwxr-xr-x 8 user user 4096 Jan 1 12:00 ..
-rw-r--r-- 1 user user  156 Jan 1 12:00 main.go

====

[Previous Session Summary]

====

User: Show me the files in this directory

[Reminder: Please reply with at least one XML tag (e.g., <RequestAccomplished>, <ExecCommand>, etc.) or your message will be considered invalid. This reminder is for the AI only and should not be included in your response to the user.]
```

### Example 2: What Debug Shows (2nd and 3rd Blocks Only)

The debug file shows only the conversation history and current structured message:

```
==================    SENT REQUEST ==================

-------------------- SECOND BLOCK: CONVERSATION HISTORY --------------------

Message 1 (user) at 2025-01-01T12:00:01Z:
Can you help me understand this project?

Message 2 (assistant) at 2025-01-01T12:00:02Z:
I'd be happy to help! I can see this is a Go project with several files...
<RequestAccomplished>1</RequestAccomplished>

-------------------- THIRD BLOCK: CURRENT STRUCTURED CONTEXT --------------------

Role: user at 2025-01-01T12:00:03Z
Content:
[Repo Map]
main.go
internal/
  manager.go
  session.go
config/
  config.go

====

[Files]

====

[Pane Content]
Pane %12 (Command: bash) [PID: 12345]:
~/project $ ls -la
total 24
drwxr-xr-x 4 user user 4096 Jan 1 12:00 .
drwxr-xr-x 8 user user 4096 Jan 1 12:00 ..
-rw-r--r-- 1 user user  156 Jan 1 12:00 main.go

====

[Previous Session Summary]

====

User: Show me the files in this directory

[Reminder: Please reply with at least one XML tag (e.g., <RequestAccomplished>, <ExecCommand>, etc.) or your message will be considered invalid. This reminder is for the AI only and should not be included in your response to the user.]

==================    RECEIVED RESPONSE ==================

I can see the files in your current directory. Here's what I found:
<ExecCommand>ls -la</ExecCommand>
```

## Key Points About Context Structure

### 1. No Duplication (Fixed)
- **Before**: Conversation history appeared in both separate messages AND structured context
- **After**: Conversation history only appears as separate message objects
- **Savings**: ~200-300 tokens per request

### 2. Conversation History vs Structured Context
**Conversation History** (in message objects):
```
User: Can you help me understand this project?
AI: I'd be happy to help! I can see this is a Go project...
<RequestAccomplished>1</RequestAccomplished>
```

**Structured Context** (in final message content - NO duplication):
```
[Repo Map]
main.go
internal/
  manager.go
  session.go
config/
  config.go

====

[Files]

====

[Pane Content]
Pane %12 (Command: bash) [PID: 12345]:
~/project $ ls -la
total 24
drwxr-xr-x 4 user user 4096 Jan 1 12:00 .
drwxr-xr-x 8 user user 4096 Jan 1 12:00 ..
-rw-r--r-- 1 user user  156 Jan 1 12:00 main.go

====

[Previous Session Summary]

====

User: Show me the files in this directory

[Reminder: Please use XML tags...]
```

### 3. Token Usage Considerations
- System prompt: Sent with every request (~500-1000 tokens) - NOT shown in debug
- Conversation history: Sent with every request (variable tokens) - shown in debug
- Structured context: Sent with every request (~1000+ tokens) - shown in debug
- Reminder text: Minimal impact (~50 tokens)

### 4. What Users See vs What AI Receives
**User Sees**:
```
TmuxAI » Show me the files in this directory
TmuxAI : I can see the files in your current directory...
```

**AI Receives**: Full context as shown in Example 1 above

## Common Misconceptions

### Misconception 1: "Debug shows everything AI receives"
**Reality**: Debug shows only conversation history and current structured message. AI also receives the system prompt.

### Misconception 2: "Reminder is added to conversation history"
**Reality**: Reminder is only in the current structured message, not in `m.Messages`.

### Misconception 3: "Context is duplicated"
**Reality**: Fixed! Conversation history now only sent once as separate message objects.

This system ensures the AI has complete context while maintaining clean conversation history and minimal token waste.