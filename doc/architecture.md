# Agenvoy - Architecture

> Back to [README](../README.md)

## Overview

Agenvoy is a local Go agent runtime. One execution engine powers the interactive TUI, browser dashboard, Telegram and Discord, and the stdin MCP server. It routes each request to a configured model, runs Skills and sandboxed tools, persists session history, and can create tools when a capability is missing.

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

The runtime matches a request to a Skill when applicable, then uses the Skill description and task text to select the primary model and fallbacks. It separately configures the dispatcher, summary, image generation, speech-to-text (STT), and text-to-speech (TTS) roles. During prompt assembly it injects the common official operating guide plus any guide matching the selected model. This enables task-aware model routing instead of one model handling every operation. For multi-provider setups, `gpt-oss-20b` through NVIDIA NIM can optionally act as a fast dispatcher.

```mermaid
graph TB
    Input[User request] --> Skill{Match Skill?}
    Skill --> Select[Resolve primary model & fallbacks]
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
    Result -->|final answer| Output[Channel / TUI / dashboard reply]
```

## Module: Tools, Skills, and Sandbox

Built-in tools, generated API/script tools, installed extensions, and MCP tools share one registry. Tools load their full schema only when needed to keep routine requests lightweight. Before execution, filesystem and command actions pass permission checks, confirmation gates, shell validation, and OS-level sandbox rules. If live data needs a tool that does not exist, the agent can build, test, and retain a new tool.

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

Every request belongs to a session. Sessions retain configuration, model choices, messages, summaries, logs, usage, and pending questions. Origin prefixes keep interactive work with the correct listener: local CLI/TUI, web, Telegram, and Discord each resume only their own pending request. Tasks are registered before they compete for a per-session concurrency slot, so queued work remains visible and cancellable.

```mermaid
graph TB
    Request[Request] --> Session[Session]
    Session --> History[History + summary]
    Session --> Logs[Action + usage logs]
    Session --> Pending[Pending question / confirmation]
    Pending --> Origin{Origin}
    Origin --> CLI[CLI / TUI]
    Origin --> Web[Dashboard]
    Origin --> TG[Telegram]
    Origin --> DC[Discord]
    Request --> Register[Register task]
    Register --> Gate{Session slot free?}
    Gate -->|yes| Execute[Run agent]
    Gate -->|no| Queue[Queued & cancellable]
    Queue --> Execute
    Execute --> Finish[Completed / failed / canceled]
```

## Module: Daemon, Dashboard, and Chat Channels

The daemon initializes storage, tools, agents, schedules, chat channels, and the local HTTP API. Its dashboard is embedded in the binary and served by the same localhost-only daemon. Telegram and Discord require only their bot tokens because the daemon initiates the connection. Since **v0.34.4**, the default voice-input-to-voice-output loop is paused for those channels; STT/TTS tools can still generate audio files and send them through either channel.

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
    Entry->>Exec: Run with origin and session
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
- File writes outside `$HOME` and non-allowlisted commands require explicit confirmation; approval is scoped to the session and requested path or binary.
- Command execution is validated and sandboxed (`sandbox-exec` on macOS and `bwrap` on Linux).
- Credentials, including provider and MCP OAuth tokens, are stored in the operating-system keychain rather than the repository.

## Persistence Layout

```mermaid
flowchart LR
    Config[~/.config/agenvoy/config.json] --> Runtime[Runtime settings]
    Config --> Sessions[Session directories]
    Sessions --> History[history.json]
    Sessions --> Summary[summary.json]
    SQLite[~/.config/agenvoy/.store/history.db] --> Search[History search]
    Store[~/.config/agenvoy/.store] --> Knowledge[Knowledge / error memory / tool cache]
    Tools[~/.config/agenvoy/tools] --> Registry[Tool registry]
    Skills[~/.config/agenvoy/skills] --> Scanner[Skill scanner]
    MCP[~/.config/agenvoy/mcp.json] --> MCPClient[MCP clients]
    Schedules[crons.json / tasks.json] --> Scheduler[Scheduler]
```

---

©️ 2026 [邱敬幃 Pardn Chiu](https://www.linkedin.com/in/pardnchiu)
