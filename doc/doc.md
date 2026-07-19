# Agenvoy - Documentation

> Back to [README](../README.md)

## Prerequisites

- Go 1.25.1 or later
- macOS or another environment supporting Go, SQLite, and the `go-pkg` sandbox dependencies
- At least one configured model-provider credential; Telegram, Discord, voice, image generation, and KuraDB need their respective credentials

## Installation

### Official installer

```bash
curl -fsSL https://agenvoy.com/scripts/install.sh | bash
agen
```

### Build from source

```bash
git clone https://github.com/pardnchiu/agenvoy.git
cd agenvoy
go build -tags fts5 -ldflags "-X github.com/pardnchiu/agenvoy/internal/runtime/tui.projectVersion=dev" -o agen ./cmd/app/
./agen
```

### Run without installing

```bash
go run ./cmd/app/
```

## Configuration

Agenvoy stores runtime data in `~/.config/agenvoy/` and keeps credentials in the operating-system keychain.

### Runtime configuration

`~/.config/agenvoy/config.json` contains user settings and runtime limits. Missing limit fields are populated with defaults.

| Setting | Default | Description |
|---|---:|---|
| `limits.port` | `17989` | Local HTTP daemon port |
| `limits.max_tool_iterations` | `128` | Maximum tool iterations per run |
| `limits.agent_send_timeout_seconds` | `600` | Model-request timeout |
| `limits.max_history_messages` | `8` | Recent history messages retained |
| `limits.max_history_bytes` | `5242880` | History-size ceiling |
| `limits.max_session_tasks` | `3` | Concurrent tasks per session |

### MCP client configuration

Configure stdio or streamable HTTP MCP servers in `~/.config/agenvoy/mcp.json`:

```json
{
  "servers": {
    "local-tools": {
      "command": "node",
      "args": ["/absolute/path/server.js"]
    },
    "remote-tools": {
      "url": "http://127.0.0.1:8000/mcp",
      "headers": { "Authorization": "Bearer ${MCP_TOKEN}" }
    }
  }
}
```

## Usage

### Interactive TUI

```bash
agen
```

Use the TUI to manage sessions, models, skills, tools, MCP servers, and optional Telegram, Discord, voice, or KuraDB integrations.

### Agent runs

```bash
# Keep per-tool confirmation
agen cli 'Summarize the main modules in this Go project'

# Allow tools automatically for this run
agen run 'Inspect the latest Git changes and produce a summary'
```

`run` bypasses per-call confirmation only; sandbox, denied-path, exclusions, and runtime limits still apply.

### Local HTTP API

The daemon listens on `127.0.0.1:17989` by default.

```bash
curl --fail-with-body -sS \
  -H 'Content-Type: application/json' \
  -d '{"content":"List the available tools","persist":false,"allow_all":false}' \
  http://127.0.0.1:17989/v1/send
```

`/v1/chat/completions` is OpenAI-compatible and stateless: include prior messages in every request when continuity is needed.

### MCP server mode

When stdin is not a terminal, `agen` serves newline-delimited JSON-RPC over stdin/stdout:

```bash
printf '%s\n' \
  '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}' \
  '{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}' \
  | agen
```

## CLI Reference

| Command | Syntax | Description |
|---|---|---|
| TUI | `agen` | Open the interactive TUI and attach to the local daemon. |
| Interactive run | `agen cli <input...>` | Run an agent with tool confirmation. |
| Automatic run | `agen run <input...>` | Run an agent with tools allowed for that request. |
| Stop | `agen stop` | Stop the daemon. |
| Update | `agen update` | Execute the official updater. |
| Daemon | `agen --daemon` | Start the daemon directly. |
| MCP server | `agen` with non-TTY stdin | Serve MCP JSON-RPC over standard I/O. |

### HTTP endpoints

| Method | Path | Description |
|---|---|---|
| `POST` | `/v1/send` | Run an agent request. |
| `POST` | `/v1/chat/completions` | Stateless OpenAI-compatible chat completion. |
| `GET` | `/v1/tools` | List current tools. |
| `POST` | `/v1/tool/:tool_name` | Run a named tool. |
| `GET` | `/v1/sessions` | List sessions. |
| `GET` | `/v1/models` | List models. |
| `GET` | `/v1/session/:session_id/status` | Get session status and usage. |
| `GET` | `/v1/session/:session_id/pending` | List pending tasks. |

## Architecture

See [Architecture](./architecture.md) for module relationships, data flows, and security boundaries. Traditional Chinese: [架構](./architecture.zh.md).

## License

This project is licensed under the [Apache License 2.0](../LICENSE).

***

©️ 2026 [邱敬幃 Pardn Chiu](https://www.linkedin.com/in/pardnchiu)
