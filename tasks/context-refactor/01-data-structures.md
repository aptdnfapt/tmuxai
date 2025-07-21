# Plan 01: Foundational Data Structures

**Objective:** Define the core data structures required for the new context management system. These structures will be added to `internal/types.go`.

This is the first and most critical step, as it lays the foundation for all subsequent changes.

---

## 🔧 **Technical Implementation**

### **1. File to Modify:** `internal/types.go`

### **2. Add New `ItemStatus` Enum**

Create a new type and constants for tracking the status of a context item.

```go
// Add this block after the existing type declarations.

type ItemStatus string

const (
    StatusActive     ItemStatus = "ACTIVE"
    StatusNew        ItemStatus = "NEW"
    StatusUpdated    ItemStatus = "UPDATED"
    StatusUnchanged  ItemStatus = "UNCHANGED"
    StatusRemoved    ItemStatus = "REMOVED"
    StatusOldSession ItemStatus = "OLD_SESSION"
)
```

### **3. Define `SectionState` Struct**

This struct will track the state of an individual piece of context (like a file, a pane, or the repo map).

```go
// Add this struct definition.

type SectionState struct {
    PreviousContent string
    Content         string
    LastChanged     int        // which message number it was last changed
    Hash            string     // content hash for change detection
    Size            int        // content size in tokens/bytes
    Status          ItemStatus // ACTIVE, REMOVED, OLD_SESSION
    Timestamp       time.Time  // when last updated
    RemovedAt       time.Time  // when removed (if applicable)
}
```

### **4. Define `OldSessionData` Struct**

This struct will hold the data from a restored session.

```go
// Add this struct definition.

type OldSessionData struct {
    SessionName   string
    SavedAt       time.Time
    Files         map[string]SectionState // filepath -> state
    Panes         map[string]SectionState // paneID -> state
    Conversation  []ChatMessage
}
```

### **5. Define the Main `ContextState` Struct**

This will be the central struct holding the entire state of the context that has been sent to the AI.

```go
// Add this struct definition.

type ContextState struct {
    RepoMap        SectionState
    Files          map[string]SectionState  // filepath -> state
    Panes          map[string]SectionState  // paneID -> state
    Prompts        SectionState
    OldSession     *OldSessionData          // pointer to restored session data
    LastUpdate     int                      // message number
    CurrentTime    time.Time                // current timestamp
}
```

---

## ✅ **Validation**

After adding these structures, the project should still compile without errors. Run `go build .` to confirm. No functional changes are expected at this stage.

## ➡️ **Next Step**

Proceed to `02-context-state-tracker.md` to implement the logic that will manage and update these new data structures.
