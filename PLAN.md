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

## III. Development Roadmap

This section outlines the key features and changes required to evolve `tmuxai` into a powerful orchestrator for `aider`.
### 0 : THE GOAL OF THE PORJECT IS TO GIVE TMUXAI ENOUGH TOOLS SO THAT IT CAN SUPPORT MY WORKFLOW BUT NOT TO HARDCODE ANY AIDER OR AIDER RELATED STUFF INSIDE TMUXAI. TMXUAI IS A STAND ALONE PROJECT OF ITS OWN . WHICH GOING TO HOLD TOOLS POWERFUL ENOGH THAT USING THE AIDER-AGENTITC MD FILE ON THE AGETIC SYSTHEM PROMPT OF THE YAML WE SHOULD BE ABLE TO ACHIVE THE COMBINE POWER . TMUXAI AND AIDER WILL UNITE ON THE CONFIG YAML SYSTHEM PROPMPT AND MAKE SURE TO UNDERSTAND THEY CAN ALSO USE THIS TOOLS AS A STAND ALONE  PROGRAM OF ITS OWN.

### 1. Transient File Context ("Fresh Read" Strategy) ✅ **DONE**

- **Problem**: Reading files adds their full content to the permanent chat history, which is inefficient and leads to stale context.
- **Solution**: Adopt `aider`'s "fresh read" approach.
    - When `<ReadFile>` is used, its content will be injected into the AI context for the **current request only**.
    - File content will **not** be appended to the persistent chat history (`m.Messages`).
    - This ensures the AI always works with the latest version of a file from disk and keeps the long-term history lean and relevant.

### 2. Persistent, Project-Scoped Session Management ✅ **DONE** (FULLY IMPLEMENTED)

- **Problem**: Conversation history is lost on exit, preventing the continuation of complex tasks.
- **Solution**: Implement a robust session management system.
    - **Storage**: In a project's root, create a `.tmuxai/` directory to store session data. If the project is a Git repository, `tmuxai` must automatically add `.tmuxai/` to `.gitignore`.
    - it will also contain a part as "content of the last sessions exec panes " which will contain all the commads ran on ther other panes that was used on . this contains the aider --message and other commands .
    - also when sessions are being restore using --restore or /sessions and choose . it should also add those files on the chat those was added already on that last session . so we need to log which files was on the session too ? 
    - **Multiple Sessions**: Allow for multiple, named session histories within a project (e.g., `.tmuxai/feature_x.json`, `.tmuxai/bug_fix_y.json`).
    - **AI-Generated Titles**: Each session file will contain an AI-generated title for easy identification.
    - **`/session` Command and --resotre flag **: Introduce a `/session` command to list all available sessions (by title) and allow the user to switch between them.
    - **Automatic Restore**: When `tmuxai` is started in a directory, it shouldnt resotore anything auto matically unless tmuxai --restore was ran . only then it gong to restore the latest chat from the josns . normally running tmuxai or (--agentic) will result in a new session . and only can get the old session history back by now typing /sessions to choose session  . 

    #### we must add --restore and /session aka both of them  ✅ **DONE**

### 3. Reliable Command Execution Tracking ✅ **DONE**

- **Problem**: The previous method of waiting for commands relied on the AI model to manually append a static marker (`echo "TMUXAI:EXITCODE:$?"`) to the command string. This was fragile because:
    - The AI could forget to add the marker.
    - Shell-specific syntax (`$?` vs. `$status`) created complexity.
    - A static marker could conflict with previous command outputs if left in the pane history.
- **Solution**: Shift responsibility from the AI to the `tmuxai` application by implementing a structured, tool-based approach.
    - **Agentic Mode**: A `wait="true"` attribute was added to the `<ExecCommand>` tag. The AI now signals its intent to wait, and `tmuxai` handles the implementation.
    - **Normal Mode**: The `/prepare` command now flags a pane internally. Any command sent to that pane will automatically have the wait logic applied.
    - **Unique Marker**: The application now appends a unique, randomly generated marker to the command (`echo "tmuxai waiting for command id: <random_id> exitcode:..."`). This prevents conflicts with old output and ensures `tmuxai` waits for the correct command to finish.
    - This change makes the system more robust, simplifies the AI's task, and removes the need for the AI to know shell-specific syntax for exit codes.

### 4. Intelligent Project Comprehension (`RepoMap`) -- only for agentic ✅ **DONE**

- **Goal**: Automatically provide the AI with a high-level understanding of the codebase by creating a "repo map," similar to the one used by `aider`. This process should be entirely automated and transparent to the user.
- **Solution**: Implement a fully automatic, Git-aware `RepoMap` generation and caching system that runs in the background.

#### Implementation Steps:

1.  **Automatic Git Repository Detection**:
    -   On startup, `tmuxai` will check if the current working directory is within a Git repository.
    -   The `RepoMap` feature will only be activated if a Git repository is detected. This ensures it operates only in intended project environments.

2.  **Automated Background Processing**:
    -   The repo map generation is not a user command or an AI tool. It is an automatic background process.
    -   On the first interaction in a session, `tmuxai` will check for a cached repo map. If it's missing or stale, it will trigger a one-time generation process.
    -   The user may see a brief message like "Analyzing project structure..." while the initial map is created.

3.  **Map Generation Using `ctags` and `git`**:
    -   `tmuxai` will execute `git ls-files` to get a list of all files tracked by Git. This is more precise than scanning the entire directory and naturally respects `.gitignore`.
    -   This list of files will be passed to the `universal-ctags` command to generate a structured index of all code symbols (functions, classes, etc.).
    -   `tmuxai` will then parse the `ctags` output and format it into a token-efficient text map.

4.  **Automatic Context Injection**:
    -   Before sending any message to the AI, `tmuxai` will automatically load the repo map from its cache.
    -   This map will be prepended to the system prompt or injected into the turn's context, giving the AI a persistent, high-level overview of the entire codebase with every interaction.
    -   This is not transient context; it is a foundational part of the AI's knowledge for the session, refreshed as needed.

5.  **Performance via Caching**:
    -   The generated map and the modification times of all source files will be stored in the `.tmuxai/` directory.
    -   Before each new user prompt, `tmuxai` will quickly check if any files have changed since the last map was generated. If not, the cached map is used instantly. If files have changed, the map is regenerated in the background.

This approach aligns perfectly with the `aider` philosophy: it's a powerful, automated tool for the AI that "just works" in the background without any user or explicit AI intervention, providing crucial context for intelligent code assistance.

### 5. Enhanced Output Formatting
  - Context Usage Bar: A behavioral prompt to show token usage after each response. ✅ DONE

- **Problem**: Current AI responses in the chat pane can be large, unformatted blocks of text that are hard to read.
- **Solution**: Improve the presentation of AI output.
    - **Structure**: Format responses using clear structures like bullet points, headings, and lists.
    - **Clarity**: Focus on "less text, more bullet points." The output should clearly delineate what `tmuxai` understands, what it plans to do, and what it needs from the user.
    - **Goal**: Make the output more scannable, actionable, and less overwhelming for the user.

    and reformating the designs using bubbles (last goal avoid for now )
  

### 6. Refined Aider Integration with aider agentic md file . ✅ **DONE** 

- **Goal**: Solidify `tmuxai`'s role as the orchestrator and `aider` as the file editor.
- **Workflow**:
    1.  `tmuxai` uses its `RepoMap` and file-reading capabilities to understand the project and form a high-level plan.
    2.  It delegates specific file creation and modification tasks to `aider` by constructing precise, non-interactive `aider` commands.
    3.  After `aider` completes an edit, `tmuxai` takes over to run verification steps (builds, tests, linters).
    4.  This creates a clean separation of concerns: `tmuxai` for strategy, `aider` for execution.

### 7. Future Vision: Native Tool Calling (Long-Term--dont bother with it right now )

- **Problem**: The current tool-calling mechanism relies on parsing XML tags from the AI's text response. This is brittle and can lead to model "hallucinations" (e.g., inventing pane IDs) or formatting errors that break the parsing logic. It's also less efficient than using the native tool-calling features provided by modern AI APIs.
- **Goal**: Transition from the current text-parsing method to a robust, native tool-calling framework. This will improve reliability, reduce errors, and align `tmuxai` with modern AI development best practices. The key challenge is that different AI providers (OpenAI, Google, Anthropic) have different, incompatible native tool-calling APIs.
- **Solution**: Implement a provider-agnostic architecture using a strategy pattern. Instead of creating multiple binaries, we will have a single binary that can switch between different provider implementations at runtime based on the user's configuration.

#### Implementation Steps:

1.  **Define a Provider Interface**: Create a new `AIProvider` interface in Go. This interface will define a standard method for handling chat completions and tool calls, abstracting away the specifics of each provider's API.
    ```go
    // Example interface
    type ToolCall struct {
        Name      string
        Arguments map[string]interface{}
    }

    type AIProvider interface {
        GetToolResponse(ctx context.Context, messages []ChatMessage, tools []ToolDefinition) ([]ToolCall, string, error)
    }
    ```

2.  **Refactor `AiClient`**: Modify `internal/ai_client.go` to act as a factory or manager that holds the currently active `AIProvider` implementation based on the user's configuration.

3.  **Create Provider Implementations**:
    *   **`OpenAICompatibleTextProvider`**: This will be the default implementation. It will encapsulate the *current* XML-in-prompt logic. This ensures full backward compatibility and continued support for any OpenAI-compatible endpoint that doesn't support a specific native tool-calling format.
    *   **`GeminiNativeProvider`**: A new implementation that specifically targets the Google Gemini API. It will format requests and parse responses according to Gemini's native tool-calling JSON structure.
    *   **Future Providers**: This architecture makes it easy to add more native providers in the future (e.g., `AnthropicNativeProvider`) without disrupting existing ones.

4.  **Update Configuration**: Enhance `config.yaml` to allow users to select their desired provider strategy.
    ```yaml
    # Example config.yaml addition
    openrouter:
      api_key: "..."
      model: "..."
      base_url: "..."
      # New setting to choose the strategy
      # options: "text_xml" (default), "gemini_native", "anthropic_native"
      provider_strategy: "text_xml"
    ```

5.  **Adapt Response Processing**: Update `internal/process_message.go` to handle the structured `ToolCall` objects returned by the `AIProvider` interface, instead of relying on `parseAIResponse` to extract tools from a string. The text-parsing logic will only be used when the `OpenAICompatibleTextProvider` is active.

This approach provides a clear path to adopting more reliable native tool-calling features while maintaining the flexibility and broad provider support that `tmuxai` currently offers.
















### bubble docs here 
TITLE: Customize Default `ItemDelegate` Styles (Go)
DESCRIPTION: This Go code demonstrates how to customize the default `ItemDelegate` styles for a `list` bubble. It shows how to create a new default delegate, modify its `SelectedTitle` and `SelectedDesc` styles using `lipgloss.Color`, and then initialize or update the list model with the customized delegate to apply the new visual settings.
SOURCE: https://github.com/charmbracelet/bubbles/blob/master/list/README.md#_snippet_2

LANGUAGE: go
CODE:
```
import "github.com/charmbracelet/bubbles/list"

// Create a new default delegate
d := list.NewDefaultDelegate()

// Change colors
c := lipgloss.Color("#6f03fc")
d.Styles.SelectedTitle = d.Styles.SelectedTitle.Foreground(c).BorderLeftForeground(c)
d.Styles.SelectedDesc = d.Styles.SelectedTitle.Copy() // reuse the title style here

// Initailize the list model with our delegate
width, height := 80, 40
l := list.New(listItems, d, width, height)

// You can also change the delegate on the fly
l.SetDelegate(d)
```

----------------------------------------

TITLE: Define `list.Item` Interface for Custom List Items (Go APIDOC)
DESCRIPTION: To create custom items for the `list` bubble, they must implement the `list.Item` interface. This interface requires a `FilterValue()` method, which provides the string used for filtering items within the list.
SOURCE: https://github.com/charmbracelet/bubbles/blob/master/list/README.md#_snippet_0

LANGUAGE: go
CODE:
```
// Item is an item that appears in the list.
type Item interface {
	// FilterValue is the value we use when filtering against this item when
	// we're filtering the list.
	FilterValue() string
}
```

----------------------------------------

TITLE: Define `list.DefaultItem` Interface for Default List Items (Go APIDOC)
DESCRIPTION: The `list.DefaultItem` interface extends `list.Item` and is specifically designed to work with `DefaultDelegate`. In addition to `FilterValue()`, it requires `Title()` and `Description()` methods to provide display text for the item.
SOURCE: https://github.com/charmbracelet/bubbles/blob/master/list/README.md#_snippet_1

LANGUAGE: go
CODE:
```
// DefaultItem describes an item designed to work with DefaultDelegate.
type DefaultItem interface {
	Item
	Title() string
	Description() string
}
```

----------------------------------------

TITLE: Define and Use Keybindings with Key Component in Go
DESCRIPTION: This snippet illustrates how to define custom keybindings using the `key.Binding` type and a `KeyMap` struct. It shows how to associate actual key combinations with help text and how to match incoming `tea.KeyMsg` events against these defined keybindings within a `tea.Model`'s `Update` method.
SOURCE: https://github.com/charmbracelet/bubbles/blob/master/README.md#_snippet_0

LANGUAGE: go
CODE:
```
type KeyMap struct {
    Up key.Binding
    Down key.Binding
}

var DefaultKeyMap = KeyMap{
    Up: key.NewBinding(
        key.WithKeys("k", "up"),        // actual keybindings
        key.WithHelp("↑/k", "move up"), // corresponding help text
    ),
    Down: key.NewBinding(
        key.WithKeys("j", "down"),
        key.WithHelp("↓/j", "move down"),
    ),
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch {
        case key.Matches(msg, DefaultKeyMap.Up):
            // The user pressed up
        case key.Matches(msg, DefaultKeyMap.Down):
            // The user pressed down
        }
    }
    return m, nil
}
```
