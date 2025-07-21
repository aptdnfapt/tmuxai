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
```
----CURRENT-TIME----
Date: July 21, 2024 - Time: 14:35:22
----END-OF-CURRENT-TIME----

----PROMPTS----
[System prompts and instructions]
----END-OF-PROMPTS----

----REPO-MAP----
[Project structure and file tree]
[UPDATED] or [UNCHANGED since message X]
----END-OF-REPO-MAP----

----FILES----
file: /path/to/main.go [UPDATED] (last modified: 14:30:15)
[file content]

file: /path/to/config.yaml [UNCHANGED since message 3] (last modified: 12:45:30)

file: /path/to/deleted.go [REMOVED] (removed at: 14:32:10)
----END-OF-FILES----

----PANES----
pane: %1 (tmuxai_exec_pane) [UPDATED] (last updated: 14:35:20)
[pane content]

pane: %2 (agentic_exec_pane) [UNCHANGED since message 5] (last updated: 14:20:45)

pane: %3 (old_session) [REMOVED] (session ended: 14:25:00)
----END-OF-PANES----

----OLD-SESSION-DATA----
[Restored from: "feature-branch-work" - saved: July 19, 2024 12:45 PM]

pane: %1 [OLD SESSION] (from: July 19, 12:45 PM)
[old pane content from restored session]

pane: %2 [OLD SESSION] (from: July 19, 12:45 PM)
[more old content]

[OLD CONVERSATION HISTORY]
User: "Run the tests"
AI: "I'll run the test suite..."
User: "Check results"
AI: "Tests passed with 85% coverage..."
----END-OF-OLD-SESSION-DATA----

----CONVERSATION----
[Only the current conversation history]
User: "What's in this project?"
AI: "I can see this is a Go project..."
User: "Run the tests"
----END-OF-CONVERSATION----
```

### **Smart Change Tracking:**
- **`[UPDATED]`** = Section has new content since last message
- **`[UNCHANGED since message X]`** = Refer to content from message X
- **`[REMOVED]`** = Item no longer exists (files deleted, sessions ended)
- **`[NEW]`** = First time this item appears
- **`[OLD SESSION]`** = Content from restored session (reference only)

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