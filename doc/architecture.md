# Agenvoy - Architecture

> Back to [README](../README.md)

## Overview

Agenvoy is a local Go agent runtime that combines an interactive terminal interface, a local HTTP daemon, chatbot integrations, and MCP client/server capabilities. The runtime shares one execution engine for model routing, session-aware tools, skills, and persistent history.

```mermaid
graph TB
    User[User / Client] --> Entry[CLI or HTTP Entry]
    Entry --> TUI[TUI]
    Entry --> Daemon[Local Daemon]
    Entry --> MCPServer[MCP Server]
    TUI --> Exec[Agent Execution]
    Daemon --> Exec
    MCPServer --> Tools[Tool Registry]
    Exec --> Router[Model Router]
    Exec --> Tools
    Exec --> Sessions[Session & Memory]
    Tools --> Guard[Permission & Sandbox]
    Daemon --> Chat[Telegram / Discord]
    Tools --> MCPClient[External MCP Clients]
```

## Module: Entry Points

The `cmd/app` binary runs the TUI by default. `agen cli <input>` retains per-tool confirmation, while `agen run <input>` allows tools for that run subject to sandbox policy. `agen stop` stops the daemon, `agen update` runs the official updater, and non-terminal stdin activates the MCP server.

```mermaid
graph TB
    subgraph CLI[cmd/app]
        Args[Arguments] --> Dispatch{Mode}
        Dispatch --> TUIEntry[Interactive TUI]
        Dispatch --> Cli[cli]
        Dispatch --> Run[run]
        Dispatch --> Stop[stop]
        Dispatch --> Update[update]
        Stdin[Non-TTY stdin] --> MCP[MCP server]
    end
    TUIEntry --> TUIRuntime[TUI runtime]
    Cli --> TUIRuntime
    Run --> TUIRuntime
    TUIRuntime --> Execution[Agent execution]
    Stop --> RuntimeState[Runtime state]
    Update --> Installer[Official update script]
```

## Module: Daemon and HTTP API

The daemon initializes the filesystem, runtime limits, ToriiDB/history storage, registered tools, agents, schedulers, chatbots, and Gin routes. The HTTP API binds to `127.0.0.1` and covers two tiers: an agent-execution surface (send, chat completions, tool calls, sessions, models, SSE logs, pending-task recovery) reachable without extra checks, and a much larger config/management surface — credentials, providers, MCP servers, cron/task automation, KuraDB lifecycle, command/skill allowlists, and read-only session-artifact/error-memory inspection — gated behind an additional `localhostOnly()` middleware since it touches credentials, config files, or process state. A separate web dashboard (hosted independently, not part of this repo) is the intended client for the config tier.

```mermaid
graph TB
    subgraph Daemon[Daemon Runtime]
        Init[Filesystem & Runtime Init] --> Storage[SQLite / History Store]
        Storage --> ToolInit[Tool Registration]
        ToolInit --> AgentInit[Agent Registry & Skill Scanner]
        AgentInit --> Services[Scheduler & Integrations]
        Services --> Routes[Gin Routes]
        Config[config.json Watcher] --> Reload[Reload agents / integrations]
        Reload --> AgentInit
    end
    Routes --> ExecAPI[Agent-execution API<br/>send · chat/completions · tools · sessions · models · SSE logs]
    Routes --> ConfigAPI[Config / management API<br/>keys · providers · MCP · cron/task · kuradb · allowlists · torii inspection]
    ConfigAPI --> LocalGuard[localhostOnly guard]
    ExecAPI --> Client[CLI / TUI / remote agents]
    LocalGuard --> Dashboard[Web dashboard, separate repo]
```

## Module: Agent Execution and Routing

A request is matched to a skill or a configured model. The executor builds system prompts and a session, sends messages to the selected model, loops through tool calls, trims context when needed, and moves to fallback agents when a send attempt fails.

```mermaid
graph TB
    subgraph Execution[Agent Execution]
        Input[User input] --> Match[Match skill]
        Match --> Resolve[Resolve primary & fallback agents]
        Resolve --> Session[Build AgentSession]
        Session --> Prompt[Build system prompts]
        Prompt --> Send[Send to model]
        Send --> Response{Response}
        Response -->|Tool call| ToolExec[Tool executor]
        ToolExec --> Send
        Response -->|Context limit| Trim[Trim / compact]
        Trim --> Send
        Response -->|Send failure| Fallback[Fallback agent]
        Fallback --> Send
        Response -->|Final text| Output[Events & response]
    end
```

## Module: Tool Registry and Sandbox

Built-in tools and discovered API, script, extension, and MCP tools enter one registry (`internal/runtime/toolAdapter` plus `internal/runtime/mcp`). Before execution, file and command operations pass through denied-path checks, allow rules, confirmation gates, shell validation, and sandbox enforcement. Reasoning rules are fetched on demand through the single `reasoning_guide(topic=...)` tool.

```mermaid
graph TB
    subgraph Tools[Tool System]
        Builtins[Built-in tools] --> Registry[Tool registry]
        Adapters[API / Script / Extension adapters] --> Registry
        MCPDiscovery[MCP discovery] --> Registry
        Registry --> Executor[Tool executor]
        Executor --> Paths[Path & permission checks]
        Executor --> Allow[Allow / confirmation gate]
        Executor --> Shell[Shell AST validator]
        Paths --> Sandbox[Sandbox]
        Allow --> Sandbox
        Shell --> Sandbox
        Sandbox --> Result[Tool result]
    end
```

## Module: Sessions, History, and Pending Work

Sessions persist configuration, model selection, message history, summaries, logs, usage, and pending interactive work. History appends deltas to `history.json` and mirrors searchable content to SQLite. Pending questions retain task metadata and resume through registered channel handlers.

```mermaid
graph TB
    subgraph Sessions[Session & Memory]
        Request[Request] --> Config[Session config]
        Request --> History[history.json delta append]
        History --> SQLite[SQLite history index]
        History --> Summary[Summary metadata]
        Request --> Logs[action.log / usage.log]
        Pending[ask_user / confirmation] --> Meta[Pending task metadata]
        Meta --> Resume[Resume handler]
        Resume --> Request
        Reset[Reset] --> History
        Reset --> SQLite
        ResetAll[ResetAll] --> Summary
    end
```

## Module: Task Lifecycle, Concurrency, and Cancellation

Every execution registers itself in `status.json` — and registers its cancel function — *before* competing for a per-session concurrency slot, so a task queued behind the limit stays observable and cancellable instead of blocking invisibly. Each task records the PID of the process running it; any reader that finds a task whose PID is no longer alive treats it as stale and clears it, so a killed or crashed process cannot leave a session permanently marked online.

Concurrency is per session (`MaxSessionTasks`, defaulting to four times the CPU count). Sessions never block or cancel one another, and exceeding the limit queues a task rather than rejecting it.

```mermaid
graph TB
    subgraph Lifecycle[Task Lifecycle]
        Start[Execute] --> Register[Register task and PID in status.json]
        Register --> CancelReg[Register cancel func under task ID]
        CancelReg --> Gate{Concurrency slot free}
        Gate -->|Yes| Run[Run agent loop]
        Gate -->|No| Queue[Queued: visible and cancellable]
        Queue --> Run
        Run --> Terminal[Completed / Failed / Canceled]
        Terminal --> Clear[Remove task from status.json]
    end
    CancelAPI[POST /v1/session/:id/cancel/:task_id] --> Registry[Task ID to cancel func registry]
    Registry --> Run
    StaleCheck[Reader finds dead PID] --> Clear
```

## Module: Chat and MCP Integrations

Telegram and Discord use a shared event pipeline with channel-specific authorization, attachment handling, pending confirmations, formatting, and push delivery. External MCP servers are consumed through stdio or streamable HTTP via the official `modelcontextprotocol/go-sdk` client in `internal/runtime/mcp`; tool-list change notifications trigger re-registration, and server instructions are injected into the agent system prompt. Agenvoy can also expose local tools as a stdin JSON-RPC MCP server (`mcp.NewServer()`).

```mermaid
graph TB
    subgraph Integrations[Integrations]
        Telegram[Telegram] --> Auth[Authorization & session match]
        Discord[Discord] --> Auth
        Auth --> Attachments[Save attachments / optional transcription]
        Attachments --> ChatRun[Run agent]
        ChatRun --> Events[Agent events]
        Events --> Format[Channel formatter]
        Format --> Reply[Reply / status / push]

        MCPConfig[mcp.json] --> Transport{Transport}
        Transport --> SDKClient[go-sdk client]
        SDKClient --> MCPTools[Registered MCP tools]
        SDKClient --> Refresh[tools/list_changed refresh]
        Refresh --> MCPTools
        SDKClient --> Instructions[Server instructions → system prompt]
        ExternalClient[External MCP client] --> LocalMCP[stdin JSON-RPC server]
        LocalMCP --> Tools[Local tool registry]
    end
```

## Data Flow

```mermaid
sequenceDiagram
    participant User
    participant TUI as TUI / HTTP
    participant Exec as Agent Executor
    participant Router as Model Router
    participant Tools as Tool Executor
    participant Store as Session Store

    User->>TUI: Submit request
    TUI->>Exec: Run with session context
    Exec->>Store: Load history and summary
    Exec->>Router: Send prompt and tool definitions
    Router-->>Exec: Model response
    alt Tool call
        Exec->>Tools: Validate and execute
        Tools-->>Exec: Tool result
        Exec->>Router: Continue
    else Final response
        Exec->>Store: Append history and usage
        Exec-->>TUI: Publish final events
        TUI-->>User: Render reply
    end
```

## State Machine

```mermaid
stateDiagram-v2
    [*] --> Initialized
    Initialized --> Ready: Tools and agents loaded
    Ready --> Selecting: Request received
    Selecting --> Queued: No concurrency slot
    Queued --> Running: Slot released
    Selecting --> Running: Agent resolved
    Running --> WaitingConfirmation: Tool confirmation
    WaitingConfirmation --> Running: Approved or skipped
    Running --> WaitingUser: ask_user pending
    WaitingUser --> Running: Answers received
    Running --> Compacting: Context limit
    Compacting --> Running: Trimmed
    Running --> Fallback: Send failure
    Fallback --> Running: Fallback selected
    Running --> Completed: Final response
    Running --> Canceled: Cancel requested
    Queued --> Canceled: Cancel requested
    Running --> Failed: Unrecoverable error
    Completed --> Ready
    Canceled --> Ready
    Failed --> Ready
    Ready --> [*]: Shutdown
```

## Security Boundaries

- The HTTP daemon binds to `127.0.0.1`; selected endpoints apply an additional localhost-only guard.
- File operations use denied-path and sensitive-file checks before execution.
- Command execution is subject to allow rules, AST-based shell validation, and sandbox policies.
- `run` mode bypasses confirmation only for its request; sandbox and denied-path protections still apply.
- Credentials are stored through the operating-system keychain integration, not in the repository.

## Persistence Layout

```mermaid
flowchart LR
    Config[~/.config/agenvoy/config.json] --> Limits[Runtime limits]
    Config --> Sessions[Session directories]
    Sessions --> History[history.json]
    Sessions --> Summary[summary.json]
    Sessions --> Pending[pending metadata]
    Sessions --> Status[status.json: active tasks and owning PID]
    SQLite[~/.config/agenvoy/.store/history.db] --> Search[History search]
    MCP[~/.config/agenvoy/mcp.json] --> MCPClients[MCP clients]
    Tools[~/.config/agenvoy/tools] --> Registry[Tool registry]
    Skills[~/.config/agenvoy/skills] --> Scanner[Skill scanner]
```

***

©️ 2026 [邱敬幃 Pardn Chiu](https://www.linkedin.com/in/pardnchiu)
