# TmuxAI Context Refactor - Master Plan

## 🎯 **Project Goal**
Implement a structured message format to eliminate 80-95% of context duplication in TmuxAI, reducing token waste and improving AI effectiveness.

## 📋 **Current Problem**
TmuxAI sends the same context (repo map + pane content + files + environment) with every message, causing:
- 60-80% token waste through duplication
- Premature context squashing (losing conversation history)
- Slower responses due to large context processing
- Higher API costs

**Example Current Waste:**
```
Message 1: [1000 tokens context + 20 tokens user] = 1020 tokens
Message 2: [1000 tokens SAME context + 15 tokens user] = 1015 tokens (90% duplicate!)
Message 3: [1000 tokens SAME context + 18 tokens user] = 1018 tokens (90% duplicate!)
Total: 3053 tokens (2000+ tokens wasted)
```

## 🚀 **Proposed Solution: Structured Message Format**

### **New Message Structure:**
The new format is a single, flat structure of top-level sections. The full context, including all file contents, is sent with every API call.

```
----PROMPTS----
[System prompts and instructions]
----END-OF-PROMPTS----

----REPO-MAP----
[Project structure and file tree]
[NEW], [UPDATED], or [UNCHANGED since message X]
----END-OF-REPO-MAP----

----FILES----
--- file: /path/to/main.go [UPDATED] (last modified: 14:30:15) ---
[Full, updated content of main.go]
--- end of file: /path/to/main.go ---

--- file: /path/to/config.yaml [UNCHANGED since message 3] (last modified: 12:45:30) ---
[Full, unchanged content of config.yaml]
--- end of file: /path/to/config.yaml ---

--- file: /path/to/old_feature.go [REMOVED] (removed at: 14:32:10) ---
----END-OF-FILES----

----OLD-SESSION-DATA----
[Restored from: "feature-branch-work" - saved: Sun, 20 Jul 2025 14:30:00 UTC]
### day time g f
====pane: %1====
$ ls -l
...output...
====end of pane %1====

[CHAT]
User: "List the files"
AI: "Okay, running ls -l"
### end of day time g f
----END-OF-OLD-SESSION-DATA----

----CURRENT-SESSION-DATA----
[CURRENT TIME: Mon, 21 Jul 2025 19:00:04 UTC]

====pane: %1 [UPDATED]====
$ ls -l
...output...
$ cat main.go
...main.go content...
___NEW-CONTENT___
$ git status
...git status output...
____END-OF-NEW-CONTENT____
====end of pane %1====

[CURRENT CHAT HISTORY]
User: "What's in this project?"
AI: "I can see this is a Go project..."
User: "Run the tests"
----END-OF-CURRENT-SESSION-DATA----
```

### **Smart Change Tracking:**
- **Stateless Principle**: The AI is stateless. The entire context, including full file contents, must be sent with every API call.
- **`[UPDATED]`**, **`[UNCHANGED since message X]`**, **`[REMOVED]`**, **`[NEW]`**: These are metadata markers for the AI. They explain the timeline of changes but DO NOT replace the content. The full content for `[UPDATED]` and `[UNCHANGED]` files is always included.
- **`----OLD-SESSION-DATA----`**: A clean, read-only block containing the final pane content and user-facing conversation from a restored session. It MUST NOT contain nested context blocks like `----REPO-MAP----`.
- **`----CURRENT-SESSION-DATA----`**: Contains the live state of the current session, including the "rolling" `___NEW-CONTENT___` blocks for panes.

### **Timestamp System:**
- **Current time** shown at top of every message
- **Individual timestamps** for each pane/file showing last update
- **Session timestamps** showing when old sessions were saved
- **Removal timestamps** showing when items were deleted/ended

### **Expected Results:**
```
Message 1: [Full context] = 1000 tokens
Message 2: [Only changed sections] = 100 tokens (90% reduction)
Message 3: [Only changed sections] = 50 tokens (95% reduction)
Total: 1150 tokens vs 3000+ tokens (62% overall reduction)
```

## 🏗️ **Implementation Architecture**

### **Core Components to Build:**

1. **Context State Tracker**
   - Track what content AI has seen
   - Detect changes in each section
   - Maintain change history with timestamps
   - Handle removed items with status markers

2. **Structured Message Builder**
   - Build messages in new format with timestamps
   - Add appropriate change tags and time markers
   - Handle section updates and old session data
   - Include current time at message start

3. **Section Managers**
   - Repo Map Manager (project structure changes)
   - Files Manager (file content changes, deletion detection)
   - Panes Manager (terminal content changes, session monitoring)
   - Prompts Manager (system prompt changes)
   - Old Session Manager (restored session handling)

4. **Change Detection System**
   - Content hashing for change detection
   - Timestamp tracking for all items
   - File existence validation
   - Tmux session monitoring
   - Smart diff algorithms

5. **Session Restoration Handler**
   - Load old session data as reference
   - Keep current session as primary
   - Handle pane/file ID conflicts
   - Preserve old conversation history

6. **State Validation System**
   - Check file existence before each message
   - Detect tmux session changes/termination
   - Mark removed items instead of deleting
   - Validate context integrity

7. **Message Parser Updates**
   - Update AI response parsing for new format
   - Handle structured context with timestamps
   - Process old session references

## 📊 **Success Metrics**

### **Token Efficiency:**
- **Target**: 80-95% reduction in context duplication
- **Measure**: Average tokens per message in 10+ message conversations
- **Baseline**: Current ~1000 tokens context per message
- **Goal**: ~100-200 tokens context per message (after first message)

### **Performance:**
- **Faster AI responses** (less context to process)
- **Longer conversations** before context squashing
- **Lower API costs**
- **Better conversation quality** (retained history)

### **Reliability:**
- **No lost context** - AI maintains full understanding
- **Accurate change detection** - No missed updates
- **Backward compatibility** - Existing features work

## 🎯 **Implementation Phases**

### **Phase 1: Foundation (Week 1-2)**
- Context state tracking system
- Basic change detection
- Structured message format

### **Phase 2: Section Managers (Week 2-3)**
- Repo map change detection
- File content tracking
- Pane content monitoring

### **Phase 3: Integration (Week 3-4)**
- Update message building pipeline
- Integrate with existing AI response handling
- Add change tags and references

### **Phase 4: Optimization (Week 4-5)**
- Performance tuning
- Advanced change detection
- Smart caching strategies

### **Phase 5: Testing & Refinement (Week 5-6)**
- Comprehensive testing
- User feedback integration
- Bug fixes and optimizations

## 🔧 **Technical Requirements**

### **New Data Structures:**
```go
type ContextState struct {
    RepoMap        SectionState
    Files          map[string]SectionState  // filepath -> state
    Panes          map[string]SectionState  // paneID -> state
    Prompts        SectionState
    OldSession     *OldSessionData          // restored session data
    LastUpdate     int                      // message number
    CurrentTime    time.Time                // current timestamp
}

type SectionState struct {
    Content       string
    LastChanged   int        // which message number
    Hash          string     // content hash for change detection
    Size          int        // content size in tokens
    Status        ItemStatus // ACTIVE, REMOVED, OLD_SESSION
    Timestamp     time.Time  // when last updated
    RemovedAt     time.Time  // when removed (if removed)
}

type OldSessionData struct {
    SessionName   string
    SavedAt       time.Time
    Files         map[string]SectionState
    Panes         map[string]SectionState
    Conversation  []ChatMessage
}

type ItemStatus string
const (
    StatusActive     ItemStatus = "ACTIVE"
    StatusRemoved    ItemStatus = "REMOVED"
    StatusOldSession ItemStatus = "OLD_SESSION"
)
```

## 🚨 **Risks & Mitigation**

### **Risks:**
1. **AI confusion** with new format and timestamps
2. **Implementation complexity** with session handling
3. **State validation overhead**
4. **Change detection bugs**
5. **Old session conflicts** with current context

### **Mitigation:**
1. **Clear system prompts** explaining format and timestamp meaning
2. **Incremental implementation** starting with basic structure
3. **Robust state validation** before each message
4. **Comprehensive testing** of change detection and session restoration
5. **Clear separation** between current and old session data

## 📋 **Deliverables**

1. **Context State Tracking System** with timestamps
2. **Structured Message Builder** with time-aware sections
3. **Section-Specific Managers** (Files, Panes, Repo, Old Sessions)
4. **Change Detection Algorithms** with state validation
5. **Session Restoration Handler** for old session data
6. **State Validation System** for file/session monitoring
7. **Updated System Prompts** explaining new format and timestamps
8. **Performance Monitoring** and token usage tracking
9. **Documentation & Tests** for all new components

## 🎯 **Next Steps**

1. **Break down into smaller tasks** (separate files in this folder)
2. **Implement foundation components** first
3. **Test with simple scenarios** before full integration
4. **Measure token reduction** at each step
5. **Iterate based on results**

---

**This refactor will transform TmuxAI from a token-wasteful system to a highly efficient, context-aware AI assistant that maintains full understanding while dramatically reducing API costs and improving performance.**
