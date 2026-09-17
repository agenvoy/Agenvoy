# Agenvoy - Architecture

> Back to [README](../README.md)

## Overview

Agenvoy is a local Go agent runtime. One execution engine powers the interactive TUI, browser dashboard, Telegram and Discord, and the stdin MCP server. It routes each request to a configured model, runs Skills and sandboxed tools, persists session history, notes, schedules, and usage locally, and can create tools when a capability is missing.

```mermaid
graph TB
    User[User] --> TUI[CLI / TUI]
    User --> Dashboard[Local Web Dashboard]
    User --> Channels[Telegram / Discord]
    Client[Claude Code / Codex / MCP Client] --> MCPServer[stdin MCP Server]
    TUI --> Exec[Agent Execution]
    Dashboard --> Daemon[Local Daemon]
    Channels --> Daemon
    Daemon --> Exec
    MCPServer --> Tools[Shared Tool Registry]
    Exec --> Router[Model Router]
    Exec --> Tools
    Exec --> Sessions[Sessions & Memory]
    Tools --> Guard[Permissions & Sandbox]
    Tools --> External[MCP / Web / Local Services]
```

## Module: Entry Points

The `agen` binary opens the TUI by default. The local daemon serves the browser dashboard at `http://127.0.0.1:17989`; Telegram and Discord connect outward from that daemon, so no inbound port or public host is required. When stdin is not a terminal, `agen` serves local tools through newline-delimited JSON-RPC MCP instead of opening the TUI.

Every session-backed surface — TUI, dashboard `/send`, pending resume, Telegram, and Discord — enters execution through the same two steps: `exec.Prepare` rescans Skills, excludes TUI-only tools and Skills outside the TUI, and resolves a leading `/<skill_name>` or a named Skill; `exec.Start` then records the input, resolves the model, builds the session, and runs the agent. Entry points only handle their own transport, authorization, and rendering. Telegram and Discord share one reply pipeline for status updates, chunking, footers, error notices, and attachments. The TUI and the daemon both watch `config.json` and reload the model registry when it changes (the daemon also reconnects the chat bots), and the TUI subscribes to the daemon log so chat verification codes and host reloads appear in the terminal.

```mermaid
graph LR
    CLI[agen] --> Mode{Invocation}
    Mode -->|terminal| TUI[Interactive TUI]
    Mode -->|--daemon| Daemon[Local daemon]
    Mode -->|non-TTY stdin| MCP[MCP server]
    Mode -->|stop / update| Maintenance[Lifecycle command]
    Dashboard[Browser] --> Daemon
    Telegram[Telegram] --> Daemon
    Discord[Discord] --> Daemon
```

## Module: Agent Execution and Model Routing

The runtime matches a request to a Skill when applicable, then selects the configured primary model and fallbacks. A leading `/<skill_name>` is the only inline syntax; delegating to a named session goes through the `subagents` tool with a `self_id`. A model named explicitly by the caller (for example the `model` field of `/send`) is used as-is and fails when it is not registered; otherwise the session-bound model or the dispatcher decides. Completion events carry the remaining provider quota — a percentage for `codex`, `grok-oauth`, `copilot`, and `ollama-cloud`, a balance for `openrouter` and `deepseek` — fetched once before the event is delivered, so the TUI footer, dashboard badge, and chat footers all show the same value. Dispatcher, summary, image generation, speech-to-text (STT), and text-to-speech (TTS) are separate optional roles. Registered models keep a user-defined priority order that decides fallback, with `pass`-tier models always last. Each model can carry a tier (`S` `A` `B` `C` `pass`) in `model_tag`; the dispatcher orders tiers by the kind of work, defaulting to A, and when one model is registered under several providers it prefers `codex` / `grok-oauth`, then `copilot`, then the direct API, then `openrouter`. Subagent legs are sized by the same tiers according to their job. Local OpenAI-compatible endpoints are registered as `<name>@<model>`; custom endpoint URLs are kept under `compats` in `config.json`, while the Ollama and llama.cpp defaults that `/model add` detects on their usual ports are built in. History is compacted once it reaches 80% of the model's input window. Window sizes come from `llm-io.agenvoy.com`, refreshed at most hourly before a run and keyed by vendor and model (`nvidia` and `openrouter` models resolve through the vendor inside the model name); when a model is not listed, `copilot@` models assume 256K and everything else 128K. During prompt assembly it injects the common official operating guide plus any guide matching the selected model. The NVIDIA NIM `nvidia/nemotron-3.5-lightning-30b-a3b` model is documented as a free, non-large model for trying Agenvoy, not as a required dispatcher or primary model.

```mermaid
graph TB
    Input[User request] --> Prepare[Prepare: rescan Skills, exclude TUI-only]
    Prepare --> Skill{/skill_name or named Skill?}
    Skill --> Select[Resolve explicit model, session model, or dispatcher]
    Select --> Session[Build session context]
    Session --> Prompt[Compose system prompt + official model guide + tools]
    Prompt --> Model[Selected model]
    Model --> Result{Response}
    Result -->|tool call| ToolExec[Tool executor]
    ToolExec --> Model
    Result -->|context limit| Compact[Compact history]
    Compact --> Model
    Result -->|send failure| Fallback[Fallback model]
    Fallback --> Model
    Result -->|final answer| Quota[Attach provider quota to completion]
    Quota --> Output[Channel / TUI / dashboard reply]
```

## Module: Tools, Skills, and Sandbox

Built-in tools, generated API/script tools, installed extensions, and MCP tools share one registry. Tools load their full schema only when needed to keep routine requests lightweight. Before execution, the executor checks denied and sensitive paths, command policy, confirmation requirements, argument validation, and the OS sandbox. A normal tool confirmation asks whether to allow that specific tool call. Restricted paths and package-management operations additionally require system verification where the channel supports it. Denied paths and configured denied commands are hard rejected; commands outside the denylist are not an allowlist failure, though they can still enter the normal confirmation flow. Long-form deliverables go through `write_report`, which writes to the configured output directory (`output_dir`; `~/Downloads` by default, or `~/.config/agenvoy/download` when that folder does not exist) rather than the work directory. Other files made for the user land there too unless the request names a location. `run_command` waits for the process to exit, so commands that start a file watcher (`--watch`, `chokidar`, or a package script that runs one, followed through `sh -c` and `package.json`) are refused before they run. File search is paged, and `read_files` returns 2048 lines by default with a notice naming the next offset. If live data needs a tool that does not exist, the agent can build, test, and retain a new tool.

```mermaid
graph TB
    Builtin[Built-in tools] --> Registry[Tool registry]
    Generated[Generated API / Script tools] --> Registry
    Extension[Extensions] --> Registry
    Remote[External MCP tools] --> Registry
    Skill[Skills] --> Agent[Agent execution]
    Registry --> Agent
    Agent --> Check[Permission / confirmation / validation]
    Check --> Sandbox[OS sandbox]
    Sandbox --> Result[Tool result]
```

## Module: Sessions, Memory, and Task Lifecycle

Every request belongs to a session. Session configuration is stored in SQLite; sessions retain messages, summaries, logs, usage, and pending questions. Active tasks publish short-lived `action:<session>:<task>` markers in ToriiDB and refresh them while running, so pending lists expose only tasks that can actually be resumed. Pending requests have two routing keys: `Origin` selects the listener (CLI/TUI, web, Telegram, or Discord), while `DeliverTo` selects the session window that receives the prompt and result. A subagent runs in its own session but inherits the parent origin and delivers confirmations and `ask_user` prompts back to the parent session. Tasks are registered before they compete for a per-session concurrency slot, so queued work remains visible and cancellable. A user cancel (TUI cancel or `ctrl+c`, **Abort task**, the cancel API) discards the task's pending entry; a pause or any other interruption stops the run but keeps the entry resumable.

```mermaid
graph TB
    Request[Request] --> Session[Session]
    Session --> History[History + summary]
    Session --> Logs[Action + usage logs]
    Session --> Pending[Pending question / confirmation]
    Pending --> Routing{Origin + DeliverTo}
    Routing --> CLI[CLI / TUI]
    Routing --> Web[Dashboard]
    Routing --> TG[Telegram]
    Routing --> DC[Discord]
    Subagent[Subagent session] -->|deliver to parent| Pending
    Request --> Register[Register task]
    Register --> Gate{Session slot free?}
    Gate -->|yes| Execute[Run agent]
    Gate -->|no| Queue[Queued & cancellable]
    Queue --> Execute
    Execute --> Finish[Completed / failed / canceled]
    Execute -->|pause / interruption| Pending
```

## Module: Daemon, Dashboard, and Chat Channels

The daemon initializes ToriiDB before SQLite, clears stale in-flight markers, then opens the history database and migrates notes before starting the local HTTP server. It registers the web confirmation listener before serving the loopback API. Its dashboard is embedded in the binary and served by the same localhost-only daemon. The dashboard's System tab compares the running version with the latest GitHub release and can open a terminal running `agen update` (Terminal.app on macOS, `cmd.exe` into the WSL distro, or a Linux terminal emulator). Telegram and Discord require only their bot tokens because the daemon initiates the connection. Since **v0.34.4**, the default voice-input-to-voice-output loop is paused for those channels; STT/TTS tools can still generate audio files and send them through either channel.

```mermaid
graph TB
    Daemon[Local daemon] --> API[HTTP API on 127.0.0.1:17989]
    API --> Dashboard[Embedded dashboard]
    Daemon --> Scheduler[Schedules]
    Daemon --> Telegram[Telegram bot]
    Daemon --> Discord[Discord bot]
    Telegram --> Attachment[Attachments + optional STT]
    Discord --> Attachment
    Attachment --> ChannelRun[Agent execution]
    ChannelRun --> Delivery[Text / file delivery]
    Scheduler --> ScheduledRun[Scheduled agent run]
```

## Module: MCP Client and Server

Agenvoy connects to external MCP servers over stdio or streamable HTTP, refreshes their tools when their catalog changes, and can complete OAuth sign-in while storing credentials in the operating-system keychain. In the other direction, running `agen` with non-terminal stdin exposes the same sandboxed local tools to Claude Code, Codex, OpenCode, and other MCP-compatible clients.

```mermaid
graph LR
    Config[mcp.json] --> MCPClient[MCP client]
    MCPClient --> Stdio[stdio MCP server]
    MCPClient --> HTTP[HTTP MCP server]
    Stdio --> Registry[Tool registry]
    HTTP --> Registry
    External[Claude Code / Codex / OpenCode] --> MCPServer[agen stdin MCP server]
    MCPServer --> Registry
    OAuth[OAuth callback] --> Keychain[OS keychain]
    Keychain --> MCPClient
```

## Data Flow

```mermaid
sequenceDiagram
    participant User
    participant Entry as TUI / Web / Channel / MCP
    participant Exec as Agent executor
    participant Router as Model router
    participant Tools as Tool executor
    participant Store as Session store

    User->>Entry: Submit request
    Entry->>Exec: Prepare (Skill match) and start with origin and session
    Exec->>Store: Load history and configuration
    Exec->>Router: Send prompt and available tools
    Router-->>Exec: Model response
    alt Tool call
        Exec->>Tools: Validate and execute
        Tools-->>Exec: Result
        Exec->>Router: Continue
    else Final response
        Exec->>Store: Append history, logs, and usage
        Exec-->>Entry: Publish result
        Entry-->>User: Render text or deliver file
    end
```

## Security Boundaries

- The dashboard and management API bind to `127.0.0.1`; the host is not exposed for normal browser or chatbot use.
- Telegram and Discord use outbound connections from the local daemon and need only a bot token.
- Denied paths and denied commands are hard rejected. Sensitive paths, writes outside `$HOME`, and other restricted actions require explicit confirmation and, where supported, system verification; approval is scoped to the session and requested path or binary.
- Command execution is validated and sandboxed (`sandbox-exec` on macOS and `bwrap` on Linux). `sudo` is refused inside `run_command`; a command that writes outside `$HOME` declares those paths through `write_paths` on that call and is approved with the system password.
- Credentials, including provider and MCP OAuth tokens, are stored in the operating-system keychain rather than the repository.

## Persistence Layout

```mermaid
flowchart LR
    Config[~/.config/agenvoy/config.json] --> Runtime[Runtime settings]
    Config --> Sessions[Session directories]
    Sessions --> History[history.json]
    Sessions --> Summary[summary.json]
    SQLite[~/.config/agenvoy/.store/history.db] --> Search[History search]
    SQLite --> Note[Notes]
    Torii0[~/.config/agenvoy/.store/db_0] --> ToolCache[Tool cache]
    Torii1[~/.config/agenvoy/.store/db_1] --> SessionMemory[Session conversation vectors]
    Torii2[~/.config/agenvoy/.store/db_2] --> ErrorMemory[Error memory]
    Torii3[~/.config/agenvoy/.store/db_3] --> Online[In-flight task markers]
    Tools[~/.config/agenvoy/tools] --> Registry[Tool registry]
    Skills[~/.config/agenvoy/skills] --> Scanner[Skill scanner]
    MCP[~/.config/agenvoy/mcp.json] --> MCPClient[MCP clients]
    Schedules[crons.json / tasks.json] --> Scheduler[Scheduler]
```

---

©️ 2026 [邱敬幃 Pardn Chiu](https://www.linkedin.com/in/pardnchiu)
