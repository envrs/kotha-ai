# Configuration Guide

Kotha reads configuration from a JSON file (`.kotha.json`) or a YAML file (`.kotha.yaml`) in the working directory, plus optional overrides from environment variables and a global config at `~/.config/kotha/config.json`.

## Configuration file

```json
{
  "provider": "openai",
  "model": "gpt-4o",
  "maxTokens": 4096,
  "temperature": 0.7,
  "workingDir": ".",
  "logLevel": "info",
  "theme": "catppuccin",
  "tui": {
    "theme": "catppuccin",
    "showTimestamps": true,
    "syntaxHighlight": true
  },
  "mcpServers": {
    "filesystem": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-filesystem", "."]
    }
  },
  "lsp": {
    "go": {
      "command": "gopls",
      "args": ["serve"]
    }
  }
}
```

## Environment variables

| Variable | Description |
|---|---|
| `KOTHA_PROVIDER` | Default LLM provider name |
| `KOTHA_MODEL` | Default model name |
| `KOTHA_API_KEY` | API key used by the CLI client |
| `KOTHA_LOG_LEVEL` | Log level: `debug`, `info`, `warn`, `error` |
| `KOTHA_WORKING_DIR` | Working directory override |

## LSP servers

Kotha spawns one LSP server per configured language. Each entry needs a `command` and optional `args`. Supported languages include Go (`gopls`), TypeScript/JavaScript (`typescript-language-server`), Python (`pylsp`), Rust (`rust-analyzer`), and more.

```json
{
  "lsp": {
    "go": { "command": "gopls", "args": ["serve"] },
    "typescript": { "command": "typescript-language-server", "args": ["--stdio"] }
  }
}
```

## MCP servers

MCP servers expose external tools to the agent. Configure them under `mcpServers` and access them from the TUI with `ctrl+m`.

## Themes

Built-in themes: `catppuccin`, `dracula`, `flexoki`, `gruvbox`. Custom themes can be loaded from `.kotha/themes/*.yaml`.