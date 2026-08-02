# Agenvoy - Documentation

> Back to [README](../README.md)

## Prerequisites

- Go 1.25.1 or later
- macOS or another environment supporting Go, SQLite, and the `go-pkg` sandbox dependencies
- At least one configured model-provider credential; Telegram, Discord, voice, and KuraDB need their respective credentials. Image generation is temporarily unavailable.

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

### Using Makefile

```bash
make build
agen
```

`make build` installs the binary to `/usr/local/bin/agen` and therefore requires `sudo`.

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
| `limits.max_history_messages` | `24` | Recent history messages retained |
| `limits.max_history_bytes` | `5242880` | History-size ceiling |

Package defaults (not currently read from `config.json`):

| Constant | Default | Description |
|---|---:|---|
| `MaxSessionTasks` | `NumCPU × 4` | Concurrent tasks per session; further tasks queue rather than fail |
| `MaxSubagentTimeoutMin` | `30` | Subagent timeout in minutes |
| `MaxResumeWaitMin` | `60` | How long a pending resume waits for answers |

### TUI execution modes

When the input area is empty, press `Shift+F` to toggle fast mode. The header displays `[fast]` while it is enabled. Fast mode is process-local and is not persisted in `config.json`; it passes `provider.ModeFast` through `go-llm-router` v0.4.0 so supported provider backends can request a faster service tier. The default mode remains available when fast mode is disabled.

### Image generation

Image generation support is temporarily removed while the router integration is being redesigned. The `image2` command, `enable_image2` configuration flag, `generate_image` tool, and related registration path are no longer available.

- Remove `enable_image2` from persisted configuration.
- Stop invoking `/image2` and `generate_image`.
- Use an external image-generation integration if image output is still required.


### MCP client configuration

MCP client and server live in `internal/runtime/mcp` and use the official [`modelcontextprotocol/go-sdk`](https://github.com/modelcontextprotocol/go-sdk). Configure stdio or streamable HTTP MCP servers in `~/.config/agenvoy/mcp.json`. Clients subscribe to tool-list change notifications and re-register tools when a remote server updates its catalog; server instructions are surfaced into the agent system prompt.

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

The daemon binds to `127.0.0.1` only. Endpoints marked **local** additionally require the request to originate from `127.0.0.1`/`::1` (`localhostOnly()` guard) — they manage credentials, config files, or process lifecycle and are meant for a same-machine dashboard, not remote clients.

**Agent execution**

| Method | Path | Description |
|---|---|---|
| `POST` | `/v1/send` | Run an agent request. |
| `POST` | `/v1/chat/completions` | Stateless OpenAI-compatible chat completion. |
| `GET` | `/v1/log` | SSE stream of events across all sessions. |
| `GET` | `/v1/tools` | List current tools. |
| `POST` | `/v1/tool/:tool_name` | Run a named tool directly. |

**Models**

| Method | Path | Description |
|---|---|---|
| `GET` | `/v1/models` | List registered models. |
| `POST` `DELETE` | `/v1/models` `/v1/models/*name` | **local** — add / remove a model. |
| `GET` `POST` | `/v1/model/dispatcher` | **local** — get/set the dispatcher model. |
| `GET` `POST` | `/v1/model/summary` | **local** — get/set the summary model. |

**Sessions**

| Method | Path | Description |
|---|---|---|
| `GET` | `/v1/sessions` | List sessions and status. |
| `POST` `PUT` `DELETE` | `/v1/session` | **local** — create / rename / delete a session. |
| `POST` | `/v1/session/:id/model` | Set the model for a session. |
| `GET` | `/v1/session/:id/status` | Get session status and usage. |
| `GET` | `/v1/session/:id/log` | SSE stream of events for one session. |
| `POST` | `/v1/session/:id/event` | **local** — publish an event into a session's stream. |
| `GET` | `/v1/session/:id/pending` | List pending (`ask_user`/confirm) tasks. |
| `GET` | `/v1/session/:id/pending/:task_hash/questions` | Get a pending task's questions. |
| `POST` | `/v1/session/:id/pending/:task_hash/resume` | Answer a pending task and resume. |
| `DELETE` | `/v1/session/:id/pending/:task_hash` | Discard a pending task without answering it. |
| `POST` | `/v1/session/:id/cancel/:task_id` | Cancel one running task. `task_id` comes from `/status`; cancelling is per-task by design, there is no cancel-everything variant. |
| `GET` `POST` | `/v1/session/:id/persona` | **local** — get/set a session's persona. |
| `POST` | `/v1/session/:id/compact` | **local** — compact history in the background (fire-and-forget, `202 Accepted`). |
| `GET` | `/v1/session/:id/daemon` | **local** — `daemon.log` lines mentioning this session ID (best-effort grep, not a true per-session log). |
| `GET` | `/v1/session/:id/action` | **local** — that session's `action.log` content. |
| `GET` | `/v1/session/:id/usage` | **local** — 24h/7d/28d per-model token usage (same aggregation as the TUI `/usage` screen). |
| `GET` | `/v1/session/:id/history` | **local** — list archived completed pending-task files. |
| `GET` | `/v1/session/:id/history/*file` | **local** — read one archived pending-task file. |

**Channels**

| Method | Path | Description |
|---|---|---|
| `GET` | `/v1/channel/status` | **local** — Telegram/Discord enabled state, bot username, whether a token is stored. |
| `POST` | `/v1/channel/telegram` `/v1/channel/discord` | **local** — `{action:"enable"\|"disable", token?}`. Enable stores the token and flips the config flag only; the `GetMe` verification the TUI does is intentionally skipped, since the daemon's existing config-file watcher already reconnects the bot and fills in its username. |

**Files & credentials**

| Method | Path | Description |
|---|---|---|
| `GET` `PUT` | `/v1/file` | **local** — read/write a file. |
| `GET` | `/v1/file/open` | **local** — open a file/URL with the OS default handler. |
| `GET` `DELETE` | `/v1/key` | **local** — check/delete a single credential in the keychain. |
| `GET` `POST` | `/v1/keys` | **local** — list/set credentials. |

**Providers**

| Method | Path | Description |
|---|---|---|
| `GET` | `/v1/providers` | **local** — list providers and their available operations. |
| `GET` | `/v1/provider/:provider/check` | **local** — whether a credential exists for this provider. |
| `POST` | `/v1/provider/:provider/key` | **local** — set an API key. |
| `GET` | `/v1/provider/:provider/oauth` | **local** — SSE device-code OAuth flow. |
| `GET` | `/v1/provider/:provider/models` | **local** — list models available to this provider. |

**MCP**

| Method | Path | Description |
|---|---|---|
| `GET` `POST` | `/v1/mcp` | **local** — list/add MCP servers. |
| `POST` | `/v1/mcp/remove` | **local** — remove an MCP server. |
| `GET` | `/v1/mcp/status` | **local** — connection status per server. |
| `GET` | `/v1/mcp/health` | **local** — health probe per server. |
| `POST` | `/v1/mcp/reconnect` | **local** — reconnect all MCP clients and re-register tools. |

**Automation**

| Method | Path | Description |
|---|---|---|
| `GET` | `/v1/schedule/*skill` | **local** — read a scheduler skill's contents. |
| `GET` `DELETE` | `/v1/cron` | **local** — list/delete cron entries. |
| `POST` | `/v1/cron/run` | **local** — fire a cron entry now (`202 Accepted`). |
| `GET` `DELETE` | `/v1/task` | **local** — list/delete one-off tasks. |
| `POST` | `/v1/task/run` | **local** — fire a task now (`202 Accepted`). |

**KuraDB & allowlists**

| Method | Path | Description |
|---|---|---|
| `GET` `POST` | `/v1/kuradb` | **local** — status/enable/disable/start/stop/restart KuraDB. Install/uninstall stay TUI-only (need a real terminal for `sudo`/install-script prompts); this API only flips the `enabled` flag and controls an already-installed `kura` process. |
| `GET` `POST` | `/v1/allowlist/cmd` | **local** — list/append the command allowlist (append-only, restart required to take effect). |
| `GET` `POST` | `/v1/allowlist/skill` | **local** — list/toggle the skill allowlist (`scope=global\|project`). |

**Inspection**

| Method | Path | Description |
|---|---|---|
| `GET` | `/v1/torii/error` | **local** — read the tool-error memory store; unfiltered when `tool`/`keyword` are both omitted. |

## Tool Reference

| Tool | Purpose |
|---|---|
| `read_files`, `list_files`, `glob_files`, `search_files` | Batch-read, list, and search files |
| `write_file`, `patch_file` | Create, overwrite, or precisely edit files |
| `run_command` | Run commands under shell validation and sandbox constraints |
| `ask_user`, `write_todo` | Interactive input and multi-step progress tracking |
| `search_tools` | Search registered tools |
| `reasoning_guide` | Fetch full reasoning rules by `topic` (`tool_generate`, `tool_error`, `subagent_dispatch`, `html_render`, …) |
| `invoke_subagent` | Delegate a single subtask |

## Architecture

See [Architecture](./architecture.md) for module relationships, data flows, and security boundaries. Traditional Chinese: [架構](./architecture.zh.md).

## License

This project is licensed under the [Apache License 2.0](../LICENSE).

***

©️ 2026 [邱敬幃 Pardn Chiu](https://www.linkedin.com/in/pardnchiu)
