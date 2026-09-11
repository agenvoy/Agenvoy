# Agenvoy - Documentation

> Back to [README](../README.md)

## Prerequisites

- Go 1.25.1 or later
- macOS or Linux; on Windows, install a WSL Linux distribution first and run Agenvoy inside its WSL terminal
- At least one configured model-provider credential; Telegram and Discord require their respective bot tokens. Speech-to-text and text-to-speech require a selected audio model and its provider credential. Image generation requires a configured image-capable provider.

## Installation

### Official installer

```bash
curl -fsSL https://agenvoy.com/scripts/install.sh | bash
agen
```

### Windows (via WSL)

Open PowerShell as an administrator, then list and install a Linux distribution:

```powershell
wsl --online --list
wsl --install <distribution-name>
```

Restart, open the installed WSL distribution, then run the [official installer](#official-installer) in its terminal.

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
| `OPENAI_API_KEY` | OpenAI and OpenAI audio models |
| `CLAUDE_API_KEY`, `GROK_API_KEY`, `DEEPSEEK_API_KEY` | The matching model providers |
| `TELEGRAM_TOKEN`, `DISCORD_TOKEN` | Chat-bot integrations |
| `GEMINI_API_KEY` | Gemini audio models and voice features |
| `OLLAMA-CLOUD_API_KEY` | Ollama Cloud (the hyphen is part of the name) |
| `COMPAT_<NAME>_API_KEY` | Optional key for a local / custom OpenAI-compatible endpoint named `<NAME>` |

### Audio model routing

Speech-to-text and text-to-speech models are configured separately from the session, dispatcher, summary, and image settings. In the TUI, use `/model stt` or `/model tts`; choose `disable` (listed last) to turn the corresponding capability off. The TUI prints `loading models...` while the available models are fetched in parallel from configured OpenAI and Gemini providers. Since **v0.34.4**, Telegram and Discord pause only the default flow that automatically returns voice output after voice input. The local `generate_audio` tool and audio model settings remain available: you can generate audio and send the resulting file to either channel.

### Chatbot integrations

Agenvoy currently supports Telegram and Discord. Both integrations use outbound connections from the local daemon, so the host does not need to expose an inbound port or public endpoint. Configuration requires only the corresponding bot token. Since **v0.34.4**, their automatic voice-input-to-voice-output default is paused; STT/TTS tools can still create audio files for delivery through either channel. Other chatbot platforms are out of scope unless they provide a meaningful security improvement.

### Runtime configuration

`~/.config/agenvoy/config.json` contains user settings and runtime limits. Missing limit fields are populated with defaults.

| Setting | Default | Description |
|---|---:|---|
| `limits.max_tool_iterations` | `128` | Maximum tool iterations per run |
| `limits.agent_send_timeout_seconds` | `600` | Model-request timeout |
| `limits.max_history_messages` | `24` | Recent history messages retained |
| `limits.max_history_bytes` | `5242880` | History-size ceiling |
| `reply_lang` | `"auto"` | Reply language. `auto` keeps the default behaviour of matching the user's message; any other value forces every reply into that language regardless of what the user writes |
| `output_dir` | `""` | Where files made for the user land when the request names no location: `write_report` output, plus the documents, exports and images the agent is told to place there by default. Empty means `~/Downloads`, or `~/.config/agenvoy/download` when that folder does not exist |
| `model_tag` | `{}` | Per-model tier, `{"<model>": "S"\|"A"\|"B"\|"C"\|"pass"}`; see [Model tiers](#model-tiers) |

`reply_lang` accepts `auto`, a code listed in `configs/jsons/reply_lang.json` (`en`, `zh-TW`, `zh-HK`, `zh-CN`, `ja`, `ko`, `es`, `fr`, `de`, `pt`, `it`, `ru`, `vi`, `th`, `id`, `ar`) or any other language name, which is passed through to the model as written. `zh-TW` and `zh-HK` are separate: Taiwan and Hong Kong Traditional Chinese differ in vocabulary and phrasing. It applies to the agent system prompt, the `/v1/chat/completions` system prompt and the generated follow-up suggestions. Set it from **Config › System**, which applies it to the running daemon at once; editing `config.json` by hand takes effect at the next daemon start.

`output_dir` expands `~` and creates the directory when it is saved; a path that cannot be created is rejected, and one that later becomes unusable falls back to the default. **Config › System** applies it to the running daemon at once. The TUI's `/reply-language` and `/output-dir` write `config.json` the same way a hand edit does.

Package defaults (not currently read from `config.json`):

| Constant | Default | Description |
|---|---:|---|
| `MaxSessionTasks` | `NumCPU × 4` | Concurrent tasks per session; further tasks queue rather than fail |
| `MaxSubagentTimeoutMin` | `30` | Subagent timeout in minutes |
| `MaxResumeWaitMin` | `60` | How long a pending resume waits for answers |

### TUI execution modes

The runtime ships 13 model providers, Ollama Cloud included, plus the `compat` entry for local or custom OpenAI-compatible endpoints (Ollama, LM Studio, self-hosted gateways).

`/model add` probes a local Ollama (`http://localhost:11434/v1`) and llama.cpp (`http://localhost:8080/v1`) with `GET /models`; each one that answers is listed as **Ollama Local** or **Llama.cpp Local** at the top of the provider list and goes straight to model selection without asking for a URL or key. Those two endpoints are built in (`configs/jsons/local_compat.json`), so nothing is written for them. Other ports or hosts are added through **Local/Custom**, which records the URL under `compats` in `config.json`. Either way the model list comes from the endpoint's own `GET /models`. Endpoint models are registered as `<name>@<model>` with the endpoint name lowercased (`ollama@gemma3:4b`); the older `compat[NAME]@<model>` form is still accepted and rewritten to the new one whenever `config.json` is loaded or saved.

When the input area is empty, press `Shift+F` to toggle fast mode. The header displays `[fast]` while it is enabled. Fast mode is process-local and is not persisted in `config.json`; it passes `provider.ModeFast` through `go-llm-router` v0.6.0 so supported provider backends can request a faster service tier. The default mode remains available when fast mode is disabled.

### Agent selection and confirmation routing

When a request matches a Skill, the dispatcher receives that Skill's description as a selection hint. Model selection therefore reflects the active task contract instead of relying on the user text alone. While assembling the prompt, Agenvoy adds its common official operating guide and, when configured, the guide that matches the selected model.

### Model tiers

Each registered model can carry a tier in `model_tag`:

| Tier | Meaning |
|---|---|
| `S` | Strongest; code and work that asks for depth or precision |
| `A` | Default for most work, one step below the flagship |
| `B` | Mainstream mid tier |
| `C` | Fast and cheap; calls tools reliably as instructed |
| `pass` | Never picked by auto routing or subagents; last in fallback, or used when a session is set to it |

Set a tier with `t` on a model row in the TUI `/model`, or with the tier button on each card under **Config › Model › Fallback Priority**. Tiers are read per request, so a change applies without a restart.

A model pinned to the session skips routing. Otherwise the dispatcher receives the tiers along with the model list; an untiered model is placed by its name (`claude-opus` S, `claude-sonnet` A, `claude-haiku` B, `*-mini` C, and so on). The kind of work decides the order: code, or a request that asks outright for depth or precision, tries S first; greetings, short answers, chat and translation try B; fetching data with tools and returning it tries C; everything else, reports and analysis included, tries A. When the same model is registered under several providers, the order within it is `codex` / `grok-oauth`, then `copilot`, then the direct API, then `openrouter`. `pass` is enforced in the prompt rather than by removal: the dispatcher is told not to return a `pass` model unless the request names it, and the fallback list appends `pass` models after every other model.

Subagent legs follow the same tiers. The planner gives each leg one job — collect, review, transform or reason — and picks its model by that job: collect C>B>A>S, transform B>C>A>S, review and reason A>S>B>C, code or high-precision work S>A>B>C.

Interactive requests also carry an origin prefix. CLI confirmations are consumed only by the TUI, web requests by the web confirmation stream, and Telegram or Discord requests by their matching channel listeners. Non-TUI confirmations expire after five minutes, preventing one channel from intercepting or indefinitely holding another channel's prompt.

### Restricted paths and commands

Paths outside `$HOME` and paths on the sensitive list (`configs/jsons/sensitive_path.json`, extendable through `sensitive_path` in `config.json`) are not refused outright: `boundary.Resolve` collects them and raises a confirmation that also demands the operating-system password (`sudo -v` inside the TUI, a password field in the web confirmation). `pkg_manage` calls are gated the same way. Approval is bound to that session and the specific path, and the sudo timestamp is the only clock — there is no second TTL: while that ticket is still valid the prompt appears without a password field. `denied_path` and `denied_command` in `config.json` are hard rejects that no approval can lift; any command not on `denied_command` runs, subject to the normal `run_command` confirmation.

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

The TUI `/sessions` selector groups sessions by ID prefix: `cli-` for local CLI, `tg-` for Telegram, `dc-` for Discord, `chat-` for Web/API, and `temp-` for short-lived work. When at least two groups are detected, the selector shows an `all` tab and one tab per prefix, with the current session listed first. The daemon watches newly created session directories with `fsnotify` and writes the session ID and configured name to the daemon log.

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
| `/model` | Pick the session model (`auto` or a registered model; `d` removes the highlighted model, `t` sets its tier); `add` a provider; set the dispatch, summary, image, STT and TTS models |
| `/mcp` | List MCP servers (`d` removes one) and `add` new ones; per server: log in, set the OAuth client, pick always-allowed `tools` (first row `all`), reconnect |
| `/sessions` `/new` | Switch to another session (`d` deletes it), or create one (names are conflict-checked) |
| `/bot` | Rename the current session or edit its persona |
| `/compact` `/reset` | Drop redundant exchanges from the current session, or reset it (double-confirmed) |
| `/allow-skill` | Mark skills as always allowed, globally or for this project |
| `/rule` `/note` | List, add or edit rules and notes |
| `/channel` | Enable or disable Telegram / Discord (tokens are validated before they are stored; `d` revokes an authorized chat), or pick the `admin` chat that receives new-chat verification codes (shown only while a channel is enabled) |
| `/startup` | Enable or disable launching the daemon on login (launchd agent on macOS, systemd user unit on Linux) |
| `/schedule` | Recurring (cron) and one-shot (task) entries in one list; `enter` fires one now, `d` deletes it; add or edit by asking the agent |
| `/pending` | List and resume interrupted tasks (`ask_user`, error recovery) |
| `/resume` `/log` `/usage` | Reload the visible transcript, follow `daemon.log` in `$PAGER`, show per-model token usage for this session above and all sessions below (24h / 7d / 28d) |
| `/key` | Edit a stored credential (`d` deletes it) |
| `/reply-language` | Pick the language every reply is written in; `auto` follows each message |
| `/output-dir` | Set where generated files land; blank uses `~/Downloads` |
| `/update` | Fetch the latest release, rebuild, quit |
| `/clear` `/exit` | Clear the visible transcript, or leave the TUI (the daemon keeps running) |
| `/<skill>` `/sched-<name>` | Run an installed skill or a scheduler entry directly |

Shortcuts work while the input area is empty:

| Key | Action |
|---|---|
| `Shift+W` / `Shift+S` | Cycle the session model backwards / forwards |
| `Shift+A` / `Shift+D` | Cycle the reasoning level |
| `Shift+F` | Toggle fast mode |
| `Shift+U` | Provider quota and balance |

Inside a popup, `esc` returns to the page that opened it and closes only the first page; list-only popups such as `/usage` scroll instead of selecting.

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
| `POST` `DELETE` | `/v1/models` `/v1/models/*name` | **local** — add / remove a model. `POST` takes `{prefix, models}`; `prefix` is a provider id from `GET /v1/providers` or the name of a local / custom endpoint (`ollama`, `llama.cpp`, or one recorded through `POST /v1/provider/compat/key`). |
| `GET` `POST` | `/v1/model` | **local** — model routing: `dispatcher`, `summary`, `image`, `stt`, and `tts`; on read it also returns `image_options`, `image_providers`, and `audio_providers`. `dispatcher` and `summary` name registered models (`prefix@model`); `image` names a provider endpoint (`openai`, `codex`, `grok`, `grok-oauth`, `gemini`) because each provider's image model is fixed inside `go-llm-router`. `stt` and `tts` name a model from the respective options exposed by `GET /v1/model/audio`. `POST` is a partial update: an omitted (or `null`) field is unchanged, `""` clears it, and `off` clears only `image`. Unknown models, providers, or unavailable audio models are rejected and nothing is written. Both verbs return the same object. |
| `GET` | `/v1/model/audio` | **local** — list the available `stt_options` and `tts_options`, derived from configured OpenAI and Gemini providers. |
| `GET` `POST` | `/v1/model/priority` | **local** — read / reorder the registered models. The order decides fallback priority; the last entry is the final line of defense, and `pass` models run after all others whatever their place here. `GET` also returns `tiers` (`{model: tier}`) and `tier_options` (`[{tier, detail}]`, the empty tier last). `POST` `{models}` moves the listed names to the front in that order and keeps the rest after them; an unknown name returns 400. |
| `POST` | `/v1/model/tier` | **local** — `{model, tier}` sets one model's tier (`S` `A` `B` `C` `pass`); `""` clears it. An unregistered model or unknown tier returns 400. |

**Sessions**

| Method | Path | Description |
|---|---|---|
| `GET` | `/v1/sessions` | List sessions and status. |
| `GET` | `/v1/usage` | **local** — 24h/7d/28d total token usage across sessions. |
| `POST` | `/v1/session` | **local** — create a session; `{prefix}` defaults to `cli-`. |
| `GET` `POST` `DELETE` | `/v1/session/:id` | **local** — one session's full state: `id`, `self_id`, `name`, `rule`, `state`, `model`, `reasoning`, `levels`, `count`. `POST` is a partial update — `self_id` / `name` / `rule` / `model` / `reasoning` are all optional and a field left out (or `null`) is untouched; `model: ""` resets to `auto`, `reasoning` must be one of `levels`. `GET` and `POST` return the same object. A duplicate `self_id` returns 409. `DELETE` removes the session directory, history, state and vectors. `GET` also takes `?chat=1` to append the raw action log under `chat`, and `?usage=1` to append 24h/7d/28d per-model token usage under `usage` (same aggregation as the TUI `/usage` screen); both are off by default because the log can be large. |
| `POST` | `/v1/session/:id/event` | **local** — publish an event into a session's stream. |
| `GET` | `/v1/session/:id/task` | List resumable pending (`ask_user`/confirm) tasks. Tasks whose run is still live are excluded — a run refreshes `action:<session_id>:<task_hash>` in ToriiDB every 55s with a 60s TTL, so a task left behind by a closed window or a killed process reappears here within a minute. |
| `GET` | `/v1/session/:id/task/:task_hash/questions` | Get a pending task's questions. |
| `POST` | `/v1/session/:id/task/:task_hash/resume` | Answer a pending task and resume. |
| `DELETE` | `/v1/session/:id/task/:task_hash` | Discard a pending task without answering it. |
| `POST` | `/v1/session/:id/cancel/:once_id` | Cancel one running task; 404 when that id is not running in this process. |
| `POST` | `/v1/session/:id/confirm/:once_id` | Resolve an outstanding tool confirmation: `{approve, remember?, allow_turn?, abort?, reason?, password?}`. Approving a restricted path or `pkg_manage` call requires `password`, is accepted only from this machine (403 otherwise) and returns 401 when the system password is wrong. 410 when the confirmation is already resolved or expired. |
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
| `GET` | `/v1/providers/quota` | **local** — remaining quota for `codex`, `grok-oauth`, `copilot`, `ollama-cloud` (`kind:"percent"`) and remaining credit for `openrouter`, `deepseek` (`kind:"balance"`), fetched in parallel with a 15s ceiling. Successful reads are cached in ToriiDB for 3 minutes and come back flagged `cached:true`; `?refresh=1` drops the cache and re-reads, and saving a key or finishing an OAuth login drops that provider's entry on its own. Providers without a credential come back with `error` instead of `value` and are never cached. |
| `POST` | `/v1/provider/:provider/key` | **local** — set an API key. For `compat` the body is `{name, url, api_key?}`: the URL is recorded under `compats` and the key, when given, is stored as `COMPAT_<NAME>_API_KEY`. |
| `GET` | `/v1/provider/:provider/oauth` | **local** — SSE device-code OAuth flow. |
| `DELETE` | `/v1/provider/:provider/oauth` | **local** — clear a stored provider login (`codex`, `copilot`, `grok-oauth`). The token keys belong to the OAuth libraries (`CODEX_OAUTH_TOKEN` and a legacy name each), so this goes through their own `ClearToken` rather than `DELETE /v1/key`. |
| `GET` | `/v1/provider/:provider/models` | **local** — list models available to this provider. For a local / custom endpoint name (`ollama`, `llama.cpp`, or a recorded one) it asks the endpoint's `GET /models`, returning 502 when the endpoint does not answer. |

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

**Rules, notes & skills**

| Method | Path | Description |
|---|---|---|
| `GET` | `/v1/rules` | **local** — list session-prompt rules stored as `.md` files under `prompts/`. |
| `GET` | `/v1/rule/*name` | **local** — read one rule. |
| `POST` `PATCH` `DELETE` | `/v1/rule` | **local** — create / update (with optional `rename`) / delete a rule. |
| `GET` | `/v1/notes` | **local** — list operator notes (name, size, `updated_at`); records live in the `note` table of `history.db`, not on disk. |
| `GET` | `/v1/note/*name` | **local** — read one note. |
| `POST` `PATCH` `DELETE` | `/v1/note` | **local** — create / update / delete a note. The name defaults to the first line when omitted. |
| `GET` | `/v1/skills` | **local** — list installed skills. |
| `GET` | `/v1/skill/*name` | **local** — read one installed skill. |
| `DELETE` | `/v1/skill` | **local** — remove one installed skill. |

**Automation**

| Method | Path | Description |
|---|---|---|
| `GET` | `/v1/schedule` | **local** — list cron entries and one-off tasks as one `schedules` array, each tagged `type=cron\|task`; `?type=` narrows to one. |
| `GET` | `/v1/schedule/*skill` | **local** — read a scheduler skill as `name` / `body` (raw SKILL.md, frontmatter included) / `files` (the other files in its folder). |
| `POST` `PATCH` | `/v1/schedule` | **local** — create/update a scheduler skill from `name` / `content` (`content` carrying its own frontmatter is written as-is, otherwise the frontmatter is composed server-side) and rebind its whole entry set to `type=cron\|task`; switching type drops the entries the skill held under the other one. |
| `DELETE` | `/v1/schedule` | **local** — delete a skill's entries (`type` narrows to one, omitted removes both); trashes the skill when nothing else binds it. |
| `POST` | `/v1/schedule/run` | **local** — fire a schedule now (`202 Accepted`). |

**Allowlists**

| Method | Path | Description |
|---|---|---|
| `GET` `POST` | `/v1/allowlist` | **local** — both allowlists in one object, `skill` and `tool`. `GET` reads `?scope=global\|project` (with `?work_dir=` required for `project`) for the skill block and `?prefix=` to narrow the tool block. `POST` takes `{skill: {name, scope?, work_dir?}}` to toggle one skill and/or `{tool: {prefix, entries}}` to replace just that prefix's auto-approve entries (same call the TUI's `/mcp` → tools makes), so unrelated rules survive; every entry must start with `prefix`, and `prefix*` collapses the rest. A block left out is untouched. |

**Configuration**

| Method | Path | Description |
|---|---|---|
| `GET` `POST` | `/v1/config/startup` | **local** — read/set launch-on-login. `POST` `{enable}` writes or removes the launchd agent (macOS) or systemd user unit (Linux); it never starts or stops the running daemon, and takes effect at the next login. Both verbs answer with `enabled` (the setting, recorded under `startup` in `config.json` whenever it is changed) and `installed` (whether the unit file is actually on disk right now) — they disagree when the unit was removed outside Agenvoy. |
| `GET` `POST` | `/v1/config/system` | **local** — read/set the System tab. `GET` returns `{reply_lang, languages:[{code,label}]}`, `languages` being the select options with `auto` first. `POST` `{reply_lang}` canonicalises a known code, writes it to `config.json` and applies it to the running daemon immediately; an empty string means `auto`, an unknown value is kept as written and passed to the model as a language name. |
| `GET` `POST` | `/v1/config/output_dir` | **local** — read/set `output_dir`. `GET` returns `{output_dir, resolved}`, `resolved` being the directory actually in use. `POST` `{output_dir}` expands `~`, creates the directory, writes `config.json` and applies it to the running daemon at once, answering `{ok, output_dir, resolved}`; an empty string restores the default, and a path that cannot be created returns 400. |

**Inspection**

| Method | Path | Description |
|---|---|---|
| `GET` | `/v1/torii/error` | **local** — read the tool-error memory (the web **Lessons** tab). Without `keyword` it lists records, optionally narrowed by `tool` (`limit` defaults to 50); with `keyword` it runs the same search the agent uses (`limit` defaults to 16). |
| `PATCH` | `/v1/torii/error` | **local** — `{id, action}`. Rewrites the action of one lesson; 404 when the id no longer exists. |
| `GET` | `/v1/daemon` | **local** — the last 28 days of `daemon.log` as `content`, narrowed by `from` / `to` (`yyyy-MM-dd-HH-mm`) and `keyword`. |

## Tool Reference

27 tools are always available; four more are excluded from a run until their prerequisite exists. Tools that cover several related actions take a `mode` argument rather than splitting into separate names.

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
| | `write_report` | Save a long-form report as `report-<timestamp>.md` in the output directory (`output_dir`; `~/Downloads` by default, or `~/.config/agenvoy/download` when `~/Downloads` does not exist); the model supplies only the content |
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
| | `find_note` | The operator's own notes, stored in SQLite (`mode=search\|list\|read`); search and list return names only |
| | `reasoning_guide` | Full reasoning rules by `topic` |
| Support | `calculate` | Arithmetic, unit and currency conversion |
| | `store_secret` | Masked prompt, stored in the keychain |
| Conditional | `generate_image` | Text to image, saved to disk — excluded while the image generator is off |
| | `generate_audio` | Text to speech, saved to disk — excluded while no TTS model is selected |
| | `list_chatbot`, `send_to_chatbot` | Cross-channel push — needs Telegram or Discord enabled |

Fifteen tools ship with full schemas — `ask_user`, `calculate`, `chat_history`, `edit_file`, `fetch_page`, `find_files`, `find_note`, `find_tools`, `read_files`, `reasoning_guide`, `run_command`, `run_skill`, `search_web`, `write_report`, `write_todo`. Everything else arrives as a name and a description; its parameters load on first use through `find_tools(mode=search)`, keeping the initial tool payload well under the full registry. The `edit_file` patch mode accepts only `{old_string, new_string}` targets (plus optional `replace_all`); `new_string` replaces `old_string`, and insertion is expressed by repeating `old_string` at the start of `new_string`. Every target is matched against the file as it is on disk before any of them apply, so their order does not matter; an `old_string` that matches more than once without `replace_all`, or two targets covering the same lines, rejects the whole batch without writing.

## Architecture

See [Architecture](./architecture.md) for module relationships, data flows, and security boundaries. Traditional Chinese: [架構](./architecture.zh.md).

## License

This project is dual-licensed: [AGPL-3.0](../LICENSE) for open source use, or a commercial license for use that cannot meet its source-disclosure requirement — see [COMMERCIAL.md](../COMMERCIAL.md).

***

©️ 2026 [邱敬幃 Pardn Chiu](https://www.linkedin.com/in/pardnchiu)
