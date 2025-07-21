# TmuxAI User Guide

## Introduction

TmuxAI is an intelligent terminal assistant that integrates with tmux sessions to provide AI-powered assistance directly within your terminal environment. This guide will help you install, configure, and use TmuxAI effectively.

## Table of Contents

1. [Installation](#installation)
2. [Post-Installation Setup](#post-installation-setup)
3. [Basic Usage](#basic-usage)
4. [Operation Modes](#operation-modes)
5. [Commands](#commands)
6. [Configuration](#configuration)
7. [Advanced Features](#advanced-features)
8. [Troubleshooting](#troubleshooting)
9. [Tips and Best Practices](#tips-and-best-practices)

## Installation

TmuxAI requires only tmux to be installed on your system. It's designed to work on Unix-based operating systems including Linux and macOS.

### Quick Install

The fastest way to install TmuxAI is using the installation script:

```bash
# install tmux if not already installed
curl -fsSL https://get.tmuxai.dev | bash
```

This installs TmuxAI to `/usr/local/bin/tmuxai` by default.

### Homebrew

If you use Homebrew, you can install TmuxAI with:

```bash
brew install tmuxai
```

### Manual Download

You can also download pre-built binaries from the [GitHub releases page](https://github.com/alvinunreal/tmuxai/releases).

After downloading, make the binary executable and move it to a directory in your PATH:

```bash
chmod +x ./tmuxai
sudo mv ./tmuxai /usr/local/bin/
```

## Post-Installation Setup

After installing TmuxAI, you need to configure your API key to start using it:

1. **Set the API Key**  
   TmuxAI uses the OpenRouter endpoint by default. Set your API key by adding the following to your shell configuration (e.g., `~/.bashrc`, `~/.zshrc`):

   ```bash
   export TMUXAI_OPENROUTER_API_KEY="your-api-key-here"
   ```

2. **Start TmuxAI**

   ```bash
   tmuxai
   ```

## Basic Usage

### Starting TmuxAI

You can start TmuxAI with an initial message, task file, or flags from the command line:

- **Basic Start:**
  ```sh
  tmuxai
  ```

- **With Initial Message:**
  ```sh
  tmuxai "Help me understand this Python script"
  ```

- **With Task File:**
  ```sh
  tmuxai -f path/to/your_task.txt
  ```

- **With Agentic Mode:**
  ```sh
  tmuxai --agentic
  ```

### TmuxAI Layout

TmuxAI operates within a single tmux window with the following pane structure:

1. **Chat Pane**: Where you interact with the AI
2. **Exec Pane**: Where commands can be executed
3. **Context Panes**: All other panes in the current window that provide context

### Basic Interaction

1. Type your question or request in the Chat Pane
2. TmuxAI will respond with information and may suggest commands
3. If a command is suggested, TmuxAI will ask for your confirmation
4. After executing a command, TmuxAI will wait for the output and continue helping you

Example interactions:

```
TmuxAI » How do I check disk usage on Linux?

TmuxAI : You can check disk usage on Linux using the `df` command. Here's how:

df -h

Execute this command?
[y/n/e(dit)]: y

Executing in pane %1: df -h
```

## Operation Modes

TmuxAI has four main operation modes, each with specific behaviors:

### 1. Observe Mode (Default)

In this mode, TmuxAI:
- Observes all visible panes
- Suggests commands that you can approve
- Uses a countdown timer before checking for new output

To use Observe Mode effectively:
- Keep relevant information visible in your panes
- Use clear, specific requests
- Review suggested commands carefully before approving

### 2. Prepare Mode

Prepare Mode provides a more robust, synchronous execution flow:
- Commands are executed with a special marker
- TmuxAI waits for the command to finish by watching for this marker
- The command's exit code is captured for better context

To enable Prepare Mode:
```
TmuxAI » /prepare
```

To disable it:
```
TmuxAI » /unprepare
```

### 3. Agentic Mode

Agentic Mode allows TmuxAI to interact with your entire tmux window:
- Create new panes
- Execute commands in any pane
- Send keystrokes or paste content into any pane

To enable Agentic Mode:
```sh
tmuxai --agentic
```

Or in your config file:
```yaml
agentic_mode: true
```

### 4. Watch Mode

Watch Mode transforms TmuxAI into a proactive assistant:
- Continuously monitors your terminal activity
- Provides suggestions based on what you're doing

To enable Watch Mode:
```
TmuxAI » /watch spot and suggest more efficient alternatives to my shell commands
```

## Commands

TmuxAI provides several commands for user interaction:

| Command                     | Description                                                                                             |
| --------------------------- | ------------------------------------------------------------------------------------------------------- |
| `/info`                     | Display system information, pane details, and context statistics                                        |
| `/clear`                    | Clear chat history                                                                                      |
| `/reset`                    | Clear chat history and reset all panes                                                                  |
| `/config`                   | View current configuration settings                                                                     |
| `/config set <key> <value>` | Override configuration for current session                                                              |
| `/squash`                   | Manually trigger context summarization                                                                  |
| `/prepare [pane_id]`        | Toggles a pane into 'prepared' mode for synchronous execution                                           |
| `/unprepare [pane_id]`      | Toggles 'prepared' mode off for a pane                                                                  |
| `/watch <description>`      | Enable Watch Mode with specified goal                                                                   |
| `/exit`                     | Exit TmuxAI                                                                                             |

## Configuration

TmuxAI can be configured through a YAML file, environment variables, or runtime commands.

### Configuration File

TmuxAI looks for its configuration file at `~/.config/tmuxai/config.yaml`. Here's a sample configuration:

```yaml
# Basic settings
debug: false
agentic_mode: false
editor: nano
max_capture_lines: 200
max_context_size: 20000
wait_interval: 5

# Confirmation settings
send_keys_confirm: true
paste_multiline_confirm: true
exec_confirm: true
read_file_confirm: true
multi_file_read: false
max_read_file_size: 250000

# Command patterns
whitelist_patterns: []
blacklist_patterns: []

# AI service configuration
openrouter:
  api_key: "your-api-key-here"
  model: "google/gemini-2.5-flash-preview"
  base_url: "https://openrouter.ai/api/v1"

# Custom prompts
prompts:
  base_system: ""
  agentic: ""
  agentic_prompt_file: ""
  chat_assistant: ""
  chat_assistant_prepared: ""
  watch: ""
```

### Environment Variables

All configuration options can also be set via environment variables using the prefix `TMUXAI_`:

```bash
export TMUXAI_DEBUG=true
export TMUXAI_MAX_CAPTURE_LINES=300
export TMUXAI_OPENROUTER_API_KEY="your-api-key-here"
export TMUXAI_OPENROUTER_MODEL="google/gemini-2.5-flash-preview"
```

### Session-Specific Configuration

You can override configuration values for your current TmuxAI session using the `/config` command:

```bash
# View current configuration
TmuxAI » /config

# Override a configuration value for this session
TmuxAI » /config set max_capture_lines 300
TmuxAI » /config set openrouter.model gpt-4o-mini
```

## Advanced Features

### Using Other AI Providers

TmuxAI can be configured to use different AI providers:

#### OpenAI

```yaml
openrouter:
  api_key: sk-proj-XXX
  model: o4-mini-2025-04-16
  base_url: https://api.openai.com/v1
```

#### Anthropic's Claude

```yaml
openrouter:
  api_key: sk-proj-XXX
  model: claude-3-7-sonnet-20250219
  base_url: https://api.anthropic.com/v1
```

#### Gemini

```yaml
openrouter:
  model: gemini-2.5-pro-preview-06-05
  api_key: XXXX
  base_url: https://generativelanguage.googleapis.com/v1beta/openai/
```

#### Local Ollama

```yaml
openrouter:
  api_key: api-key
  model: gemma3:1b
  base_url: http://localhost:11434/v1
```

### Context Management (Squashing)

As you work with TmuxAI, your conversation history grows. To manage token usage and prevent context limits from being reached, TmuxAI uses two primary techniques:

#### Efficient Context Tracking
TmuxAI employs a sophisticated context tracking system. After the first message, it only sends content from panes or files that have been updated. This dramatically reduces the number of tokens sent with each message, saving costs and allowing for longer, more coherent conversations without losing important details from your terminal.

#### Squashing
When the conversation history still grows too large despite the efficient context tracking, TmuxAI uses "squashing" as a secondary mechanism. This process summarizes older parts of the conversation to free up space for new messages, ensuring the interaction can continue.

- Check your current context utilization with `/info`
- Manually trigger squashing with `/squash`
- Automatic squashing occurs when context reaches 80% of maximum size

### External Tool Integration (Aider)

TmuxAI can work with Aider (a completely separate, standalone command-line AI coding assistant) through configuration:

**Important: TmuxAI and Aider are entirely separate projects developed independently.**

The connection between these tools is established solely through TmuxAI's configuration system, which allows TmuxAI to:
- Handle project understanding and exploration
- Execute Aider as an external program for precise file editing (which TmuxAI cannot do directly)
- Use non-interactive Aider commands for seamless automation
- Verify changes through tests and builds

To set up this configuration-based connection:
1. Install Aider separately (it's a different project): `pip install aider-chat`
2. Set up the agentic prompt file in your TmuxAI config:
   ```yaml
   prompts:
     agentic_prompt_file: "/path/to/aider_agentic_prompt.md"
   ```
3. Start TmuxAI in agentic mode:
   ```sh
   tmuxai --agentic
   ```

This configuration does not modify either project - it simply instructs TmuxAI's AI model on how to execute commands to the external Aider tool.

## Troubleshooting

### Common Issues and Solutions

1. **TmuxAI doesn't start**
   - Ensure tmux is installed and running
   - Check that your API key is set correctly
   - Verify that you have the correct permissions for the TmuxAI binary

2. **AI responses are slow or timing out**
   - Check your internet connection
   - Try a different AI model
   - Reduce the context size with `/squash`

3. **Commands aren't executing correctly**
   - Check that you have the necessary permissions
   - Try using Prepare Mode for more reliable execution
   - Verify that the command is compatible with your shell

4. **Context is getting too large**
   - Use `/squash` to manually reduce context size
   - Configure a smaller `max_context_size` in your config
   - Clear history with `/clear` when starting a new task

### Debugging

If you encounter issues, you can enable debug mode:

```yaml
debug: true
```

This will create debug logs in `~/.config/tmuxai/debug/` that can help diagnose problems.

## Tips and Best Practices

1. **Use Prepare Mode for complex commands**
   - Prepare Mode ensures commands complete before TmuxAI continues

2. **Leverage Agentic Mode for multi-pane workflows**
   - Create dedicated panes for different tasks
   - Let TmuxAI manage the workflow across panes

3. **Use Watch Mode for learning**
   - Enable Watch Mode to get suggestions for more efficient commands
   - Great for improving your command-line skills

4. **Manage context effectively**
   - Use `/squash` before starting new, unrelated tasks
   - Keep an eye on context size with `/info`

5. **Customize your configuration**
   - Adjust `max_capture_lines` based on your needs
   - Set up whitelist patterns for commonly used commands

6. **Use TmuxAI with external tools like Aider for coding tasks**
   - Let TmuxAI handle project exploration and workflow orchestration
   - Configure TmuxAI to execute Aider (a separate tool) for precise code editing
   - Remember that TmuxAI and Aider are completely separate projects connected only through configuration

7. **Use clear, specific requests**
   - Provide context in your requests
   - Break complex tasks into smaller steps

8. **Leverage file reading for project understanding**
   - Ask TmuxAI to read key files like README.md first
   - Build context incrementally with related files
