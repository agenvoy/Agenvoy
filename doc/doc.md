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

`make build` installs the binary to `/usr/local/bin/agen` (hence the `sudo` prompt) and refreshes the bundled extensions, replacing `~/.config/agenvoy/skills/.system` and `~/.config/agenvoy/tools/.system` with the contents of `extensions/`.

| Target | What it does |
|---|---|
| `make build` | Build with the current git tag as version, install to `/usr/local/bin/agen`, refresh bundled skills/tools |
| `make app` | `stop` → `build` → launch the TUI |
| `make cli <input>` / `make run <input>` | Run one request without installing |
| `make stop` | Stop the daemon |
| `make update` | Run the official updater |
| `make test` | `go test -v -count=1 ./...` |
| `make setup` | Build the macOS installer app bundles into `dist/` |

### Run without installing

```bash
go run ./cmd/app/
```

## Configuration

Agenvoy stores runtime data in `~/.config/agenvoy/` and keeps credentials in the operating-system keychain.

### Common credentials

| Keychain entry | Used by |
|---|---|
| `OPENAI_API_KEY` | OpenAI and KuraDB |
| `CLAUDE_API_KEY`, `GROK_API_KEY`, `DEEPSEEK_API_KEY` | The matching model providers |
| `TELEGRAM_TOKEN`, `DISCORD_TOKEN` | Chat-bot integrations |
| `GEMINI_API_KEY` | Voice replies and `transcribe_media`; without it the tool is not registered |

### Runtime configuration

`~/.config/agenvoy/config.json` contains user settings and runtime limits. Missing limit fields are populated with defaults.

| Setting | Default | Description |
|---|---:|---|
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

The runtime ships 12 model providers, plus the `compat` entry for local or custom OpenAI-compatible endpoints (Ollama, LM Studio, self-hosted gateways).

When the input area is empty, press `Shift+F` to toggle fast mode. The header displays `[fast]` while it is enabled. Fast mode is process-local and is not persisted in `config.json`; it passes `provider.ModeFast` through `go-llm-router` v0.5.1 so supported provider backends can request a faster service tier. The default mode remains available when fast mode is disabled.

### Restricted paths and commands

Paths under `denied_map` (credential stores, key material) and commands outside the allowlist are not refused outright: `boundary.Resolve` and `tools.RestrictedCommands` collect them and raise a confirmation that also demands the operating-system password (`sudo -v` inside the TUI). Approval is bound to that session and the specific path or binary, and the sudo timestamp is the only clock — there is no second TTL.

Two consequences worth knowing before automating anything:

- Only the TUI can approve them. Any channel that cannot collect a password (HTTP API, Telegram, Discord, subagents) gets the call back as skipped, with a message saying so.
- `run_command` never negotiates `denied_map`; the sandbox profile denies those paths unconditionally, so the error points at `find_files` / `read_files`, which can ask.

Commands that need to write outside `$HOME` declare `write_paths` instead of escalating; only those paths are bound into the sandbox.

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

Type a message to run it in the current session. Everything else is a slash command; `/` alone opens a picker that also lists installed skills and scheduler entries.

| Command | Purpose |
|---|---|
| `/model` | Add or remove providers, pick the session / dispatcher / summary model |
| `/mcp` | List MCP servers; add, install into other agents, log in, reconnect, inspect tools, set per-tool permission, remove |
| `/switch` `/new` | Switch to another session, or create one (names are conflict-checked) |
| `/bot` | Rename the current session or edit its persona |
| `/memory` | `compact` / `reset` / `summary` for the current session |
| `/dangerous` | Remove a session, or edit the skill / command allowlists |
| `/discord` `/telegram` `/voice` | Enable or disable each channel; tokens are validated before they are stored |
| `/admin-channel` | Pick which authorized chat receives new-chat verification codes |
| `/kuradb` | Install, update or reconnect KuraDB as an MCP server |
| `/cron` `/task` | Add, edit or remove recurring and one-shot scheduled work |
| `/pending` | List and resume interrupted tasks (`ask_user`, error recovery) |
| `/resume` `/log` `/usage` | Reload the visible transcript, open `action.log` in `$PAGER`, show per-model token usage |
| `/key` | Rotate a stored credential |
| `/update` | Fetch the latest release, rebuild, quit |
| `/clear` `/exit` | Clear the visible transcript, or leave the TUI (the daemon keeps running) |
| `/<skill>` `/sched-<name>` | Run an installed skill or a scheduler entry directly |

Shortcuts work while the input area is empty:

| Key | Action |
|---|---|
| `Shift+W` / `Shift+S` | Cycle the session model backwards / forwards |
| `Shift+A` / `Shift+D` | Cycle the reasoning level |
| `Shift+F` | Toggle fast mode |
| `Shift+T` | Toggle command mode — input runs as a shell command in the current directory, outside the sandbox |
| `Shift+U` | Provider quota and balance |
| `Shift+M` | Registered model list |

### Agent runs

```bash
# Keep per-tool confirmation
agen cli 'Summarize the main modules in this Go project'

# Allow tools automatically for this run
agen run 'Inspect the latest Git changes and produce a summary'
```

`run` bypasses per-call confirmation only; sandbox, denied-path, exclusions, and runtime limits still apply.

### Local HTTP API

The daemon listens on `127.0.0.1:17989`. The port is fixed and not configurable; Open WebUI, when deployed, is likewise fixed on `17990` and proxied at `/webui`.

```bash
curl --fail-with-body -sS \
  -H 'Content-Type: application/json' \
  -d '{"content":"List the available tools","persist":false,"allow_all":false}' \
  http://127.0.0.1:17989/v1/send
```

`/v1/chat/completions` is OpenAI-compatible and stateless: include prior messages in every request when continuity is needed. `reasoning_effort` accepts `none` `low` `medium` `high` `xhigh` `max` (plus the aliases `minimal` `extra` `ultra`); omitted or unrecognized values fall back to the session's reasoning setting.

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

## HTTP API Reference

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
| `GET` | `/v1/models` | List registered models (OpenAI `{data:[…]}` shape, `auto` included). |
| `GET` | `/v1/models/*id` | Read one registered model. |
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
| `POST` | `/v1/session/:id/confirm/:request_id` | Resolve an outstanding tool confirmation: `{approve, remember?, allow_turn?, abort?, reason?}`. Restricted paths and non-allowlisted commands cannot be approved here — they need the password check only the TUI can collect, so an approval without it comes back as skipped. |
| `GET` `POST` | `/v1/session/:id/persona` | **local** — get/set a session's persona. |
| `GET` `POST` | `/v1/session/:id/reasoning` | **local** — get/set the session's reasoning level. |
| `POST` | `/v1/session/:id/reset` | **local** — clear the session's history and summary. |
| `POST` | `/v1/session/:id/summary` | **local** — rebuild the session summary in the background. |
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
| `GET` | `/v1/channel/admin` | **local** — current relay target plus the pickable ones: `{admin_channel, authorized, chats:[{value,type,id,name}]}`. `chats` comes from the `.telegram` / `.discord` auth files (tg first, then dc) and each `value` can be posted back as-is; `authorized` says whether the current value is still on that list (a hand-typed ID reads `false`). |
| `POST` | `/v1/channel/admin` | **local** — `{value:"tg@<chatID>"\|"dc@<channelID>"\|""}`. Sets where new-chat verification codes are relayed; an empty string clears it. `value` is required (omitting it returns 400 so an empty body cannot silently clear the setting). Only the format is validated — an ID that is not in the authorized list makes `NotifyAdminCode` log a warning and keep the code log-only. |

**Files & credentials**

| Method | Path | Description |
|---|---|---|
| `GET` `PUT` | `/v1/file` | **local** — read/write a file. |
| `GET` | `/v1/file/open` | **local** — open a file/URL with the OS default handler. |
| `GET` | `/v1/file/locate` | **local** — find candidate paths for a bare file name (`name`, plus optional `dir=1`, `child`, `size`, `mtime` filters). |
| `GET` | `/v1/workdir` | **local** — resolve and validate a work directory (`?path=`), returning the absolute path. |
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
| `GET` `POST` | `/v1/mcp` | **local** — list/add MCP servers. The `GET` also returns `oauth: {name: bool}` for the HTTP servers, saying which ones already hold a token. |
| `POST` | `/v1/mcp/remove` | **local** — remove an MCP server. |
| `GET` | `/v1/mcp/status` | **local** — connection status per server. |
| `GET` | `/v1/mcp/health` | **local** — health probe per server. |
| `POST` | `/v1/mcp/reconnect` | **local** — reconnect all MCP clients and re-register tools. |
| `GET` | `/v1/mcp/oauth?name=X` | **local** — SSE OAuth login for one HTTP MCP server, mirroring `/v1/provider/:provider/oauth`: emits `{"url":…}` for the browser, then `{"done":true,"ok":…}` (plus `reconnect_error` when the post-login reconnect fails). Times out after 10 minutes, or when the client disconnects. |
| `POST` | `/v1/mcp/oauth/callback` | **local** — `{name, url}`. Hands the redirect URL back when the browser cannot reach the daemon's loopback listener on `localhost:17988`; the code is parsed out of the URL's query. 400 if no login is waiting for that server. |
| `POST` | `/v1/mcp/oauth/client` | **local** — `{name, client_id, client_secret?, redirect_uri?}`. Stores a pre-registered OAuth client for servers that reject dynamic registration; `redirect_uri` defaults to `http://localhost:17988/callback` and must match the provider console exactly. Clears any existing token first. |
| `DELETE` | `/v1/mcp/oauth` | **local** — `{name}`. Clears both the stored token and the client registration for that server. |

**Rules, knowledge & skills**

| Method | Path | Description |
|---|---|---|
| `GET` | `/v1/rules` | **local** — list session-prompt rules stored as `.md` files under `prompts/`. |
| `GET` | `/v1/rule/*name` | **local** — read one rule. |
| `POST` `PATCH` `DELETE` | `/v1/rule` | **local** — create / update (with optional `rename`) / delete a rule. |
| `GET` | `/v1/knowledges` | **local** — list operator notes (name, size, `updated_at`); records live in ToriiDB, not on disk. |
| `GET` | `/v1/knowledge/*name` | **local** — read one note. |
| `POST` `PATCH` `DELETE` | `/v1/knowledge` | **local** — create / update / delete a note. The name defaults to the first line when omitted. |
| `GET` | `/v1/skills` | **local** — list installed skills. |

**Automation**

| Method | Path | Description |
|---|---|---|
| `GET` | `/v1/schedule/*skill` | **local** — read a scheduler skill's contents. |
| `GET` `DELETE` | `/v1/cron` | **local** — list/delete cron entries. |
| `POST` | `/v1/cron/run` | **local** — fire a cron entry now (`202 Accepted`). |
| `GET` `DELETE` | `/v1/task` | **local** — list/delete one-off tasks. |
| `POST` | `/v1/task/run` | **local** — fire a task now (`202 Accepted`). |

**Allowlists**

| Method | Path | Description |
|---|---|---|
| `GET` `POST` | `/v1/allowlist/cmd` | **local** — list/append the command allowlist (append-only, restart required to take effect). |
| `GET` `POST` | `/v1/allowlist/skill` | **local** — list/toggle the skill allowlist (`scope=global\|project`). |
| `GET` `POST` | `/v1/allowlist/tool` | **local** — the global tool auto-approve list. `GET` returns every entry, or only those under `?prefix=`. `POST` `{prefix, entries}` replaces just that prefix's entries (same call the TUI's `/mcp` → permission makes), so unrelated rules survive; every entry must start with `prefix`, and `prefix*` collapses the rest. |

**Inspection**

| Method | Path | Description |
|---|---|---|
| `GET` | `/v1/torii/error` | **local** — read the tool-error memory store; unfiltered when `tool`/`keyword` are both omitted. |

## Tool Reference

26 tools ship in the registry; three more register only when their prerequisite exists. Tools that cover several related actions take a `mode` argument rather than splitting into separate names.

| Group | Tool | Purpose |
|---|---|---|
| Tools | `find_tools` | Discover what exists and pull a tool's schema in (`mode=search\|list`) |
| | `edit_tool` | Create, fix or trash a tool definition (`mode=write\|patch\|remove`) |
| | `test_tool` | Run a script tool in the sandbox before it goes live |
| Skills | `run_skill` | Load a named skill's reference material into the turn |
| | `edit_skill` | Author the files under the skills directory (`mode=write\|patch\|remove`) |
| Scheduling | `schedules` | Inspect, reschedule or cancel timed and recurring runs (`mode=list\|patch\|remove\|write`) |
| Files | `find_files` | Locate by directory, name pattern or content (`mode=list\|glob\|search`) |
| | `read_files` | Batch-read text, PDF, DOCX, PPTX, CSV or images |
| | `edit_file` | Create, edit, move aside or restore a file (`mode=write\|patch\|remove\|restore`) |
| | `file_history` | Recorded versions of every file the tools changed (`mode=list\|read`) |
| Execution | `run_command` | Run a binary in the work directory under sandbox constraints |
| | `open_file` | Hand a file to the OS default application |
| | `download_file` | Fetch a binary asset to disk |
| | `install_dependence` | Install a missing system binary |
| Coordination | `subagents` | Delegate a subtask to its own session (`mode=invoke\|list`) |
| | `write_todo` | Live checklist the user watches |
| | `ask_user` | Pause and ask; the turn resumes on the answer |
| Network | `search_web` | DuckDuckGo results and Google News headlines together |
| | `fetch_page` | Full page content as markdown, html or json |
| | `http_request` | Raw HTTP call, multipart upload included |
| State | `chat_history` | This session's action log and messages (`mode=list\|read\|search`) |
| | `error_history` | Tool failures kept across sessions (`mode=search\|read\|write`) |
| | `find_knowledge` | The operator's own notes, stored in ToriiDB (`mode=search\|list\|read`); search and list return names only |
| | `reasoning_guide` | Full reasoning rules by `topic` |
| Support | `calculate` | Arithmetic, unit and currency conversion |
| | `store_secret` | Masked prompt, stored in the keychain |
| Conditional | `transcribe_media` | Audio and video to text — needs `GEMINI_API_KEY` |
| | `list_chatbot`, `send_to_chatbot` | Cross-channel push — needs Telegram or Discord enabled |

Thirteen tools ship with full schemas — `ask_user`, `calculate`, `edit_file`, `fetch_page`, `find_files`, `find_knowledge`, `find_tools`, `read_files`, `reasoning_guide`, `run_command`, `run_skill`, `search_web`, `write_todo`. Everything else arrives as a name and a description; its parameters load on first use through `find_tools(mode=search)`, keeping the initial tool payload well under the full registry.

## Architecture

See [Architecture](./architecture.md) for module relationships, data flows, and security boundaries. Traditional Chinese: [架構](./architecture.zh.md).

## License

This project is licensed under the [Apache License 2.0](../LICENSE).

***

©️ 2026 [邱敬幃 Pardn Chiu](https://www.linkedin.com/in/pardnchiu)
