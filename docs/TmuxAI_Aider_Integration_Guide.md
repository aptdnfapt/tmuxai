# TmuxAI-Aider Integration Guide

## Introduction

This guide explains how TmuxAI can integrate with Aider to provide a powerful AI-driven development environment. It's important to understand that **TmuxAI and Aider are completely separate, standalone projects** developed independently. The integration is achieved solely through TmuxAI's configuration system, which allows TmuxAI to execute Aider commands.

In this integration, TmuxAI acts as the "maestro" orchestrating the overall workflow, while Aider serves as the specialized "file surgeon" handling precise code modifications that TmuxAI itself cannot perform directly.

## Table of Contents

1. [Overview](#overview)
2. [Setup and Configuration](#setup-and-configuration)
3. [Workflow](#workflow)
4. [Command Patterns](#command-patterns)
5. [Best Practices](#best-practices)
6. [Troubleshooting](#troubleshooting)
7. [Advanced Usage](#advanced-usage)

## Overview

### What is Aider?

[Aider](https://github.com/paul-gauthier/aider) is a command-line AI pair programming tool that allows you to chat with GPT about your code. It can:
- Edit files in your local git repository
- Make multiple changes across multiple files
- Explain its changes and reasoning
- Commit changes to git when you're satisfied

### The TmuxAI-Aider Integration

The integration between TmuxAI and Aider is **not a built-in feature** but rather a configuration-based approach that connects these two separate tools. This connection is established through TmuxAI's agentic prompt configuration, which instructs TmuxAI on how to execute Aider commands.

In this workflow:

- **TmuxAI** (one standalone project) handles:
  - Project exploration and understanding
  - Managing multiple tmux panes
  - Running verification commands
  - Orchestrating the overall workflow
  - Executing Aider as an external tool

- **Aider** (a completely separate standalone project) handles:
  - Precise file editing (which TmuxAI cannot do directly)
  - Code modifications
  - Git integration

This configuration-based integration preserves TmuxAI's non-intrusive philosophy while enabling a powerful, agentic coding workflow by leveraging Aider's specialized capabilities.

## Setup and Configuration

Since TmuxAI and Aider are separate standalone projects, you need to install and configure both independently, then set up TmuxAI to work with Aider through its configuration system.

### Prerequisites

1. Install TmuxAI (see the [TmuxAI User Guide](TmuxAI_User_Guide.md))
2. Separately install Aider (a completely different project):
   ```bash
   pip install aider-chat
   ```
3. Configure Aider with your API key (Aider uses its own configuration):
   ```bash
   export OPENAI_API_KEY="your-api-key-here"
   ```

### Connecting the Two Projects via Configuration

The connection between these separate tools is established solely through TmuxAI's configuration:

1. Create or edit your TmuxAI configuration file at `~/.config/tmuxai/config.yaml`
2. Configure the agentic prompt file that teaches TmuxAI how to use the external Aider tool:

   ```yaml
   prompts:
     agentic_prompt_file: "/path/to/aider_agentic_prompt.md"
   ```

   Alternatively, you can copy the content of the `aider_agentic_prompt.md` file directly into your config:

   ```yaml
   prompts:
     agentic: |
       ==== TMUXAI AIDER ORCHESTRATION MODE ====

       You are TmuxAI operating in Aider Orchestration Mode. Your role is to act as the "maestro" while Aider serves as the specialized "file surgeon."

       CORE PHILOSOPHY:
       - TmuxAI handles project understanding, exploration, and workflow orchestration
       - Aider (a separate tool) handles precise file editing and code modifications that TmuxAI cannot do directly
       - Use NON-INTERACTIVE aider commands for seamless automation
       - Leverage TmuxAI's native multi-file reading, repo mapping, and shell execution capabilities
       
       # Additional prompt content...
   ```

This configuration does not modify either project - it simply instructs TmuxAI's AI model on how to execute commands to the external Aider tool.

## Workflow

The TmuxAI-Aider integration follows a structured workflow:

### 1. Initialization

Start TmuxAI in agentic mode in your project's root directory:

```bash
tmuxai --agentic
```

### 2. Project Comprehension

Ask TmuxAI to understand your project:

```
TmuxAI » Check out this project and let me know when you're ready
```

TmuxAI will:
- Explore the project structure using commands like `ls -la` and `tree`
- Read key files like README.md, go.mod, package.json, etc.
- Build a mental model of the project

### 3. Task Definition

Describe the task you want to accomplish:

```
TmuxAI » Add a user authentication feature to this project
```

### 4. Autonomous Exploration

TmuxAI will:
- Analyze the requirements
- Identify relevant files
- Read and understand the code structure
- Plan the necessary changes

### 5. Execution with Aider

TmuxAI will execute Aider commands to make the necessary changes:

```
TmuxAI » <ExecCommand>aider --yes --message "Add a user authentication feature with login, registration, and password reset functionality"</ExecCommand>
```

The `--yes` flag makes Aider non-interactive, allowing TmuxAI to orchestrate the workflow seamlessly.

### 6. Verification

After Aider makes the changes, TmuxAI will:
- Run tests to verify the changes
- Build the project if necessary
- Check for errors or issues
- Make additional changes if needed

### 7. Iteration

If issues are found, TmuxAI will:
- Analyze the errors
- Plan additional changes
- Execute more Aider commands
- Verify again until the task is complete

## Command Patterns

### TmuxAI Commands for Project Exploration

```
# Read key files
<ReadFile>README.md</ReadFile>
<ReadFile>go.mod</ReadFile>
<ReadFile>main.go</ReadFile>

# Explore project structure
<ExecCommand>ls -la</ExecCommand>
<ExecCommand>find . -type f -name "*.go" | sort</ExecCommand>
<ExecCommand>git log --oneline -n 10</ExecCommand>
```

### Aider Commands for Code Modification

```
# Basic Aider command
<ExecCommand>aider --yes --message "Add a user authentication feature"</ExecCommand>

# Aider with specific files
<ExecCommand>aider --yes --message "Update the login function to include rate limiting" auth.go handlers.go</ExecCommand>

# Aider with git integration
<ExecCommand>aider --yes --message "Fix the bugs in the authentication middleware" --commit</ExecCommand>
```

### Verification Commands

```
# Run tests
<ExecCommand>go test ./...</ExecCommand>

# Build the project
<ExecCommand>go build</ExecCommand>

# Check for errors
<ExecCommand>go vet ./...</ExecCommand>
```

## Best Practices

### 1. Start with Project Understanding

Always begin by asking TmuxAI to understand the project structure before making changes:

```
TmuxAI » Please explore this project and understand its structure before we make any changes
```

### 2. Be Specific with Requirements

Provide clear, specific requirements for the changes you want to make:

```
TmuxAI » Add a user authentication system with the following features:
1. User registration with email verification
2. Login with JWT token generation
3. Password reset functionality
4. Role-based access control
```

### 3. Use Incremental Changes

For complex tasks, break them down into smaller, incremental changes:

```
TmuxAI » Let's implement the authentication system in steps:
1. First, create the user model and database schema
2. Then, implement the registration endpoint
3. Next, add the login functionality
4. Finally, implement password reset
```

### 4. Always Verify Changes

Ask TmuxAI to verify changes after they're made:

```
TmuxAI » After implementing the authentication system, please run the tests and make sure everything works
```

### 5. Review Git Diffs

Ask TmuxAI to show you the changes that were made:

```
TmuxAI » Show me a summary of the changes that were made to implement authentication
```

### 6. Use Multiple Panes Effectively

Take advantage of TmuxAI's agentic mode to use multiple panes:

```
TmuxAI » Create a new pane for running tests, another for viewing logs, and keep the main pane for code editing
```

## Troubleshooting

### Common Issues and Solutions

1. **Aider not found**
   - Ensure Aider is installed and in your PATH
   - Try using the full path to the Aider executable

2. **Aider API key issues**
   - Verify that your API key is set correctly
   - Try exporting the API key in the same session

3. **Git repository issues**
   - Ensure you're in a git repository
   - Check that git is initialized and configured

4. **TmuxAI not using Aider correctly**
   - Verify that the agentic prompt file is configured correctly
   - Check that TmuxAI is running in agentic mode

5. **Aider making incorrect changes**
   - Be more specific with your requirements
   - Break down complex tasks into smaller steps
   - Review changes before committing

### Debugging

If you encounter issues with the TmuxAI-Aider integration:

1. Enable debug mode in TmuxAI:
   ```yaml
   debug: true
   ```

2. Run Aider manually to verify it works:
   ```bash
   aider --help
   ```

3. Check the TmuxAI debug logs:
   ```bash
   cat ~/.config/tmuxai/debug/debug-*.txt
   ```

## Advanced Usage

### Custom Aider Configuration

You can customize Aider's behavior by creating an `.aider.conf.yml` file in your project root:

```yaml
# .aider.conf.yml
model: gpt-4-turbo
edit_format: diff
auto_commits: false
```

### Combining with Git Hooks

Set up git hooks to run tests automatically before committing:

```bash
# .git/hooks/pre-commit
#!/bin/sh
go test ./...
```

Make the hook executable:
```bash
chmod +x .git/hooks/pre-commit
```

### Using Aider with Specific Files

For large projects, you can direct Aider to focus on specific files:

```
TmuxAI » <ExecCommand>aider --yes --message "Update the authentication middleware" auth/middleware.go auth/handlers.go</ExecCommand>
```

### Creating Complex Workflows

You can create complex workflows by combining TmuxAI's pane management with Aider's code editing:

```
TmuxAI » Create a new pane for running the application, another for viewing logs, and use Aider in the main pane to implement the authentication system. After each change, restart the application and test the new functionality.
```

## Conclusion

The TmuxAI-Aider integration provides a powerful environment for AI-assisted development. By combining TmuxAI's terminal orchestration capabilities with Aider's code editing prowess, you can create a seamless workflow that handles everything from project exploration to code modification and verification.

Remember that the key to effective use of this integration is to:
1. Start with thorough project understanding
2. Be specific with your requirements
3. Break complex tasks into smaller steps
4. Always verify changes
5. Review and understand the modifications

With practice, the TmuxAI-Aider integration can significantly enhance your development workflow and productivity.