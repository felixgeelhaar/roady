# Installation

## Quick Install

### Homebrew (macOS/Linux)

```bash
brew install felixgeelhaar/tap/roady
```

### Go Install

```bash
go install github.com/felixgeelhaar/roady@latest
```

### Download Binary

Download the latest release from [GitHub Releases](https://github.com/felixgeelhaar/roady/releases/latest):

```bash
# macOS (Apple Silicon)
curl -L https://github.com/felixgeelhaar/roady/releases/latest/download/roady-darwin-arm64 -o /usr/local/bin/roady

# macOS (Intel)
curl -L https://github.com/felixgeelhaar/roady/releases/latest/download/roady-darwin-amd64 -o /usr/local/bin/roady

# Linux
curl -L https://github.com/felixgeelhaar/roady/releases/latest/download/roady-linux-amd64 -o /usr/local/bin/roady

chmod +x /usr/local/bin/roady
```

## AI Coding Tool Setup

### One-Command Setup

Roady works with Claude Code, OpenCode, Claude Desktop, OpenAI Codex, and Gemini.

```bash
# Claude Code CLI
roady setup claude-code

# OpenCode
roady setup opencode

# Claude Desktop
roady setup claude-desktop

# OpenAI Codex
roady setup openai

# Google Gemini
roady setup gemini

# All platforms (commands only)
roady setup global
```

### OpenCode Setup

Add to `~/.opencode/config.json`:

```json
{
  "mcpServers": {
    "roady": {
      "command": "roady",
      "args": ["mcp"]
    }
  }
}
```

### OpenAI Codex Setup

```python
from agents import Agent
import subprocess

# Start Roady MCP server
roady_process = subprocess.Popen(
    ["roady", "mcp", "--transport", "stdio"],
    stdout=subprocess.PIPE,
    stdin=subprocess.PIPE,
)

# Use with Codex agent
agent = Agent(
    name="Developer",
    mcp_servers=[roady_process],
)
```

### Google Gemini Setup

Via Google AI Studio or Vertex AI Agent Builder, add Roady as an MCP server:

```json
{
  "mcpServers": {
    "roady": {
      "command": "roady",
      "args": ["mcp"]
    }
  }
}
```

## Manual MCP Configuration

## Verify Installation

```bash
# Check version
roady --version

# Run setup wizard
roady config-wizard

# Check health
roady doctor
```

## AI Provider Setup

### Ollama (Local)

```bash
export ROADY_AI_PROVIDER=ollama
export ROADY_AI_MODEL=llama3
```

### OpenAI

```bash
export ROADY_AI_PROVIDER=openai
export OPENAI_API_KEY=sk-...
```

### Anthropic

```bash
export ROADY_AI_PROVIDER=anthropic
export ANTHROPIC_API_KEY=sk-ant-...
```

## Shell Completion

```bash
# Bash
roady completion bash > /etc/bash_completion.d/roady

# Zsh
roady completion zsh > "${fpath[1]}/_roady"

# Fish
roady completion fish > ~/.config/fish/completions/roady.fish
```

## Next Steps

1. Initialize a project: `roady init my-project`
2. Add a feature: `roady spec add "Feature Name" "Description"`
3. Generate plan: `roady plan generate --ai`
4. Approve and execute: `roady plan approve`
