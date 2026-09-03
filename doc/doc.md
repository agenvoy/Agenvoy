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
go build -tags fts5 -ldflags "-X github.com/pardnchiu/agenvoy/internal/runtime.CurrentVersion=dev" -o agen ./cmd/app/
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

### Agent selection and confirmation routing

When a request matches a Skill, the dispatcher receives that Skill's description as a selection hint. Model selection therefore reflects the active task contract instead of relying on the user text alone.

Interactive requests also carry an origin prefix. CLI confirmations are consumed only by the TUI, web requests by the web confirmation stream, and Telegram or Discord requests by their matching channel listeners. Non-TUI confirmations expire after five minutes, preventing one channel from intercepting or indefinitely holding another channel's prompt.

### Restricted paths and commands

Paths outside `$HOME` and commands outside the allowlist are not refused outright: `boundary.Resolve` and `tools.RestrictedCommands` collect them and raise a confirmation that also demands the operating-system password (`sudo -v` inside the TUI). Approval is bound to that session and the specific path or binary, and the sudo timestamp is the only clock — there is no second TTL: while that ticket is still valid the prompt appears without a password field.

Two consequences worth knowing before automating anything:

- Normal tool confirmations return to the originating CLI, web, Telegram, or Discord channel. The TUI listens only for CLI-origin requests.
- Restricted operations that require operating-system verification still execute only when that verification is available. A channel may approve the prompt, but an unverified restricted call is skipped rather than elevated.
- Reads are not restricted by path. The sandbox constrains writes, not reads, so a command fails on a path only when the operating system itself refuses.

`$HOME` is always writable and needs no setup. When a command has to write outside it, the agent attaches `write_paths` to that one call; you get a prompt asking for your system password, and only after you approve are those paths bound in as well.

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

Agenvoy itself speaks MCP over stdio: run the `agen` binary with stdin piped (no TTY) and it starts the MCP server instead of the TUI. Register it in another agent by hand — Claude Code (`~/.claude.json`), OpenCode (`~/.config/opencode/opencode.jsonc`) and Codex (`~/.codex/config.toml`) each keep their own file:

```json
{
  "mcpServers": {
    "agenvoy": { "command": "agen" }
  }
}
```

```toml
[mcp_servers.agenvoy]
command = "agen"
```

### Session Classification and Monitoring

The TUI session selector groups sessions by ID prefix: `cli-` for local CLI, `tg-` for Telegram, `dc-` for Discord, `chat-` for Web/API, and `temp-` for short-lived work. When at least two groups are detected, the selector shows an `all` tab and one tab per prefix, with the current session listed first. The daemon watches newly created session directories with `fsnotify` and writes the session ID and configured name to the daemon log.

Session personas are stored in the history SQLite database. `self_id` is normalized to lowercase and accepts only up to 32 ASCII letters, digits, `_`, or `-`; non-empty values must be unique. At daemon startup, legacy per-session `bot.json`, bot markdown, `config.json`, and `status.json` files are migrated into SQLite/state tables.

The daemon also runs a background runtime monitor. Every 30 seconds it checks CPU usage, Go-process memory, and the TCP connection to `1.1.1.1:443`. High CPU, high memory, network interruption, and network recovery are written to the daemon log; on CPU anomalies it also attempts to list the top three processes.

## Usage

### Interactive TUI

```bash
agen
```

Type a message to run it in the current session. Everything else is a slash command; `/` alone opens a picker that also lists installed skills and scheduler entries.

| Command | Purpose |
|---|---|
| `/model` | Add or remove providers, pick the session / dispatcher / summary model, set the image generator |
| `/mcp` | List MCP servers; add, log in, reconnect, inspect tools, set per-tool permission, remove |
| `/switch` `/new` | Switch to another session, or create one (names are conflict-checked) |
| `/bot` | Rename the current session or edit its persona |
| `/memory` | `compact` / `reset` / `summary` for the current session |
| `/dangerous` | Remove a session, or edit the skill / command allowlists |
| `/discord` `/telegram` `/voice` | Enable or disable each channel; tokens are validated before they are stored |
| `/startup` | Enable or disable launching the daemon on login (launchd agent on macOS, systemd user unit on Linux) |
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

### Web and file-response rendering

The backend preserves `[SEND_FILE:...]` markers while events move through result, SSE, pending, and multilog handlers so delivery metadata remains available to channel consumers. The web dashboard removes those markers only when rendering message text; users see the response body without the internal transport marker.

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
| `GET` | `/v1/info/version` | Build version stamped at compile time (`{version, dev}`); `dev` is true for an untagged build. |
| `GET` | `/v1/log` | SSE stream. With no query it carries daemon `slog` records only (`EventDaemonLog` frames with `source` as the level) — the same feed the TUI header shows, and it includes new-chat verification codes, so daemon frames are attached only for loopback callers. `?sessions=a,b` adds those sessions' events on the same connection; `replay=0` skips the backlog, `daemon=0` drops the daemon frames. A remote caller must pass `sessions`. |
| `GET` | `/v1/mcp/tools` | List the tools registered from connected MCP servers (`mcp__*`). |

**Models**

| Method | Path | Description |
|---|---|---|
| `GET` | `/v1/models` | List registered models (OpenAI `{data:[...]}` shape, `auto` included). |
| `GET` | `/v1/models/*id` | Read one registered model. |
| `POST` `DELETE` | `/v1/models` `/v1/models/*name` | **local** — add / remove a model. |
| `GET` `POST` | `/v1/model` | **local** — model routing: `dispatcher`, `summary`, `image`, plus `image_options` on read. The three fields hold different kinds of value: `dispatcher` and `summary` name a registered model (`prefix@model`), while `image` names a provider endpoint (`openai`, `codex`, `grok`, `grok-oauth`, `gemini`) because each provider's image model is fixed inside `go-llm-router` — `image_options` lists only the providers that currently hold credentials. `POST` is a partial update: a field left out (or `null`) is untouched, `""` clears it, and `off` is accepted for `image` as an alias of `""`. An unregistered model, an unknown provider, or a provider with no credentials is rejected and nothing is written. Both verbs return the same object. |

**Sessions**

| Method | Path | Description |
|---|---|---|
| `GET` | `/v1/sessions` | List sessions and status. |
| `GET` | `/v1/usage` | **local** — 24h/7d/28d total token usage across sessions. |
| `POST` | `/v1/session` | **local** — create a session; `{prefix}` defaults to `cli-`. |
| `GET` `POST` `DELETE` | `/v1/session/:id` | **local** — one session's full state: `id`, `self_id`, `name`, `rule`, `state`, `model`, `reasoning`, `levels`, `count`. `POST` is a partial update — `self_id` / `name` / `rule` / `model` / `reasoning` are all optional and a field left out (or `null`) is untouched; `model: ""` resets to `auto`, `reasoning` must be one of `levels`. `GET` and `POST` return the same object. A duplicate `self_id` returns 409. `DELETE` removes the session directory, history, state and vectors. `GET` also takes `?chat=1` to append the raw action log under `chat`, and `?usage=1` to append 24h/7d/28d per-model token usage under `usage` (same aggregation as the TUI `/usage` screen); both are off by default because the log can be large. |
| `POST` | `/v1/session/:id/event` | **local** — publish an event into a session's stream. |
| `GET` | `/v1/session/:id/task` | List resumable pending (`ask_user`/confirm) tasks. Tasks whose run is still live are excluded — a run refreshes `action:<session_id>:<task_hash>` in ToriiDB every 3s with a 5s TTL, so a task left behind by a closed window or a killed process reappears here within 5 seconds. |
| `GET` | `/v1/session/:id/task/:task_hash/questions` | Get a pending task's questions. |
| `POST` | `/v1/session/:id/task/:task_hash/resume` | Answer a pending task and resume. |
| `DELETE` | `/v1/session/:id/task/:task_hash` | Discard a pending task without answering it. |
| `POST` | `/v1/session/:id/cancel/:once_id` | Cancel one running task; 404 when that id is not running in this process. |
| `POST` | `/v1/session/:id/confirm/:once_id` | Resolve an outstanding tool confirmation: `{approve, remember?, allow_turn?, abort?, reason?}`. Restricted paths and non-allowlisted commands cannot be approved here — they need the password check only the TUI can collect, so an approval without it comes back as skipped. |
| `POST` | `/v1/session/:id/memory` | **local** — one memory operation on the session, picked by `action`: `summary` rebuilds the rolling summary and returns `count`; `compact` drops older messages and returns `removed`; `reset` clears the conversation and returns `removed`, and requires `mode` — `summary` keeps the rolling summary, `all` wipes it too. |
| `GET` | `/v1/session/:id/task/history` | **local** — completed tasks of this session, newest first: `{task_hash, end_at, objective, model, reasoning}` per row. `?keyword=` filters on the objective and the recorded action text. |
| `GET` | `/v1/session/:id/task/:task_hash/history` | **local** — the full action record of one completed task, returned as a JSON string under `content`. 404 when that hash has no record. |

**Channels**

| Method | Path | Description |
|---|---|---|
| `GET` | `/v1/channel` | **local** — every channel read in one object: `telegram` and `discord` each carry `{enabled, username, has_token}`, and `admin` carries `{channel, authorized, chats:[{value,type,id,name}]}`. `chats` comes from the `.telegram` / `.discord` auth files (tg first, then dc) and each `value` can be posted back as-is; `authorized` says whether the current relay target is still on that list (a hand-typed ID reads `false`). |
| `POST` | `/v1/channel/telegram` `/v1/channel/discord` | **local** — `{action:"enable"\|"disable", token?}`. Enable stores the token and flips the config flag only; the `GetMe` verification the TUI does is intentionally skipped, since the daemon's existing config-file watcher already reconnects the bot and fills in its username. |
| `GET` | `/v1/channel/:channel/chats` | **local** — chats that finished verification for `telegram` / `discord`, read from the `.telegram` / `.discord` auth files. Only meaningful while the bot runs, so ask for it after `status` reports `enabled`. |
| `DELETE` | `/v1/channel/:channel/chat` | **local** — `{id}`. Drops one chat from that auth file; the chat has to verify again before the bot answers it. 404 when the id is not on the list. |
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
| `GET` | `/v1/providers/usage` | **local** — remaining quota for `codex`, `grok-oauth`, `copilot` (`kind:"percent"`) and remaining credit for `openrouter`, `deepseek` (`kind:"balance"`), fetched in parallel with a 15s ceiling. Successful reads are cached in ToriiDB for 3 minutes and come back flagged `cached:true`; `?refresh=1` drops the cache and re-reads, and saving a key or finishing an OAuth login drops that provider's entry on its own. Providers without a credential come back with `error` instead of `value` and are never cached. |
| `POST` | `/v1/provider/:provider/key` | **local** — set an API key. |
| `GET` | `/v1/provider/:provider/oauth` | **local** — SSE device-code OAuth flow. |
| `DELETE` | `/v1/provider/:provider/oauth` | **local** — clear a stored provider login (`codex`, `copilot`, `grok-oauth`). The token keys belong to the OAuth libraries (`CODEX_OAUTH_TOKEN` and a legacy name each), so this goes through their own `ClearToken` rather than `DELETE /v1/key`. |
| `GET` | `/v1/provider/:provider/models` | **local** — list models available to this provider. |

**MCP**

| Method | Path | Description |
|---|---|---|
| `GET` `POST` | `/v1/mcp` | **local** — list/add MCP servers. The `GET` also returns `oauth: {name: bool}` for the HTTP servers, saying which ones already hold a token. |
| `POST` | `/v1/mcp/remove` | **local** — remove an MCP server. |
| `GET` | `/v1/mcp/status` | **local** — connection status per server. |
| `POST` | `/v1/mcp/reconnect` | **local** — reconnect all MCP clients and re-register tools. |
| `GET` | `/v1/mcp/oauth?name=X` | **local** — SSE OAuth login for one HTTP MCP server, mirroring `/v1/provider/:provider/oauth`: emits `{"url":...}` for the browser, then `{"done":true,"ok":...}` (plus `reconnect_error` when the post-login reconnect fails). Times out after 10 minutes, or when the client disconnects. |
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
| `GET` | `/v1/skill/*name` | **local** — read one installed skill. |
| `DELETE` | `/v1/skill` | **local** — remove one installed skill. |

**Automation**

| Method | Path | Description |
|---|---|---|
| `GET` | `/v1/schedule` | **local** — list cron entries and one-off tasks as one `schedules` array, each tagged `type=cron\|task`; `?type=` narrows to one. |
| `GET` | `/v1/schedule/*skill` | **local** — read a scheduler skill split into `name` / `description` / `body` (frontmatter parsed off). |
| `POST` `PATCH` | `/v1/schedule` | **local** — create/update a scheduler skill from `name` / `description` / `content` (the frontmatter is composed server-side) and rebind its whole entry set to `type=cron\|task`; switching type drops the entries the skill held under the other one. |
| `DELETE` | `/v1/schedule` | **local** — delete a skill's entries (`type` narrows to one, omitted removes both); trashes the skill when nothing else binds it. |
| `POST` | `/v1/schedule/run` | **local** — fire a schedule now (`202 Accepted`). |

**Allowlists**

| Method | Path | Description |
|---|---|---|
| `GET` `POST` | `/v1/allowlist` | **local** — both allowlists in one object, `skill` and `tool`. `GET` reads `?scope=global\|project` (with `?work_dir=` required for `project`) for the skill block and `?prefix=` to narrow the tool block. `POST` takes `{skill: {name, scope?, work_dir?}}` to toggle one skill and/or `{tool: {prefix, entries}}` to replace just that prefix's auto-approve entries (same call the TUI's `/mcp` → permission makes), so unrelated rules survive; every entry must start with `prefix`, and `prefix*` collapses the rest. A block left out is untouched. |

**Configuration**

| Method | Path | Description |
|---|---|---|
| `GET` `POST` | `/v1/config/startup` | **local** — read/set launch-on-login. `POST` `{enable}` writes or removes the launchd agent (macOS) or systemd user unit (Linux); it never starts or stops the running daemon, and takes effect at the next login. |

**Inspection**

| Method | Path | Description |
|---|---|---|
| `GET` `POST` `DELETE` | `/v1/toriidb/:db/*key`, `/v1/toriidb` | **local** — the single ToriiDB gate. `GET` on a plain key returns one entry, on a pattern (`*`) returns every matching entry — narrowed server-side by `?contains=<substr>` (case-insensitive, matched against the value) and `?after=<unix seconds>` (entry mtime); `?keys=1` returns names only and `?search=<text>&limit=` runs vector search. `POST` `{db,key,value,expire_at,vector}` writes with an absolute expiry, or `{db,keys,ttl}` re-expires existing keys by relative seconds; `DELETE` `{db,keys}` removes. Every process other than the daemon reaches ToriiDB only through here, so tool cache, chat vectors, knowledge, error memory and pending liveness are shared across the TUI and the daemon. |
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
| | `pkg_manage` | Drive the Linux package manager (install / remove / update / upgrade / search / info); Linux only, every channel |
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
| Conditional | `generate_image` | Text to image, saved to disk — excluded while the image generator is off |
| | `list_chatbot`, `send_to_chatbot` | Cross-channel push — needs Telegram or Discord enabled |

Thirteen tools ship with full schemas — `ask_user`, `calculate`, `edit_file`, `fetch_page`, `find_files`, `find_knowledge`, `find_tools`, `read_files`, `reasoning_guide`, `run_command`, `run_skill`, `search_web`, `write_todo`. Everything else arrives as a name and a description; its parameters load on first use through `find_tools(mode=search)`, keeping the initial tool payload well under the full registry.

## Architecture

See [Architecture](./architecture.md) for module relationships, data flows, and security boundaries. Traditional Chinese: [架構](./architecture.zh.md).

## License

This project is licensed under the [Apache License 2.0](../LICENSE).

***

©️ 2026 [邱敬幃 Pardn Chiu](https://www.linkedin.com/in/pardnchiu)
