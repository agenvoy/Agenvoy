# KuraDB RAG

KuraDB is an independent RAG (Retrieval-Augmented Generation) daemon that Agenvoy talks to over local HTTP. It is a separate long-running process (`kura`), started/stopped by the user — Agenvoy never spawns or owns its lifecycle. Agenvoy exposes two tools to the agent (`list_rag`, `search_rag`) that call KuraDB's API.

## What it is

KuraDB ([pardnchiu/KuraDB](https://github.com/pardnchiu/KuraDB)) is a self-developed local document index that:

- Indexes user files (notes, inbox, code, …) into multiple named databases
- Provides keyword search via `gse` tokenization (Chinese-aware)
- Provides semantic search via OpenAI embeddings (`text-embedding-3-small`)
- Runs entirely on the user's machine, self-daemonizing on `kura` — no external service

Agenvoy talks to KuraDB over a local HTTP API. The endpoint URL is written by `kura` to `~/.config/kuradb/endpoint` on start; its PID and start time are written to `~/.config/kuradb/runtime.uid`.

## Agenvoy-side surface (`internal/runtime/kuradb/kuradb.go`)

Agenvoy does not manage the KuraDB process — it only observes it:

| Function | Purpose |
|---|---|
| `IsInstalled()` | `/usr/local/bin/kura` exists and is executable |
| `IsRunning()` | Reads `~/.config/kuradb/runtime.uid`, checks the recorded PID is still alive (`os.FindProcess` + signal 0) |
| `Version()` | Reads the module version Go stamped into the binary at build time (`debug/buildinfo`) — the release tag, since `install.sh` builds from a clean tag checkout |
| `Health(ctx, onFail)` | Ticks every minute; a strike requires **both** `IsRunning()` and a live `GET <endpoint>/api/health` — 3 consecutive strikes calls `onFail` |
| `SyncOpenAIKey(value)` | Writes the OpenAI key into a *separate* OS keychain entry under service `kuradb` (`security`/`secret-tool`, not go-pkg keychain) — `kura` runs as its own process and can't read Agenvoy's keychain namespace |
| `Remove()` | Deletes the endpoint file (used to force tool calls to fail fast once KuraDB is considered down) |

### Health gating (`cmd/app/cmdDeamon.go::reloadKuradb`)

On config change (fsnotify on `~/.config/agenvoy/config.json`), if `kuradb_enabled=true` and `kuradb.IsInstalled()` and an `OPENAI_API_KEY` is in Agenvoy's keychain: sync the key into KuraDB's keychain entry and start a `Health` goroutine. Any gate failure is a silent no-op — no logging, no auto-disable, no config write. 3 consecutive health strikes → `disableKuradb()`: write `kuradb_enabled=false`, remove the endpoint file, call `reloadKuradb()` again (explicit, not via the fsnotify watcher, to avoid a race window).

## Tool registration

The two RAG tools live in `internal/runtime/kuradb/tool/` and register once at daemon boot (`cmd/app/cmdDeamon.go::kuradbTool.Register()`, not `init()` — `init()` fires before `filesystem.Init()`, so the gate check would always fail).

| Tool | Description |
|---|---|
| `list_rag` | List available KuraDB databases (e.g. `notes`, `inbox`, `code`) |
| `search_rag` | Search a database via KuraDB's unified `/api/search` — keyword (`gse` tokenization) and semantic (OpenAI embeddings) run together by default; `?target=keyword`/`?target=semantic` narrows to a single mode |

Registration gate is single-condition `cfg.KuradbEnabled`, checked once at boot — re-enabling after the binary becomes available requires restarting `agen`. `kuradbGet()` (the shared HTTP helper in `tool/register.go`) is the per-call second-line defense: it resolves the endpoint file fresh on every call and returns a clear error if KuraDB isn't running.

## Per-turn dynamic exclusion

`exec.Execute()` checks whether `~/.config/kuradb/endpoint` exists (`go_pkg_filesystem_reader.Exists`) after building the executor. When absent, `list_rag` and `search_rag` are appended to `data.ExcludeTools`, and the existing filter mechanism strips them from the tool list for that turn.

The result: the LLM **never sees** `list_rag` / `search_rag` when the endpoint file is gone — not even the stub names.

**Why this matters:** without dynamic exclusion, the LLM would see RAG tool stubs whenever KuraDB is stopped, call them, and get errors — confusing both LLM and user.

## `/feature kuradb` TUI wizard

Exposed only through the TUI (no CLI subcommand by design — install.sh + sudo prompts need a real TTY). The option list reflects current state:

```
/feature kuradb
  not installed → enable
  installed     → update, disable, and start (if stopped) or stop (if running)
```

The popup header shows `kura <version>  ● running (<endpoint>)` or `○ stopped`.

### Enable / update flow

1. If `OPENAI_API_KEY` isn't in Agenvoy's keychain yet, a `popupText` collects it → `keychain.Set` (Agenvoy's own keychain) + `kuradb.SyncOpenAIKey` (KuraDB's separate keychain entry)
2. `tea.ExecProcess` runs `curl -fsSL https://kuradb.agenvoy.com/scripts/install.sh | bash` with the TTY handed to the child so `sudo` prompts and package manager output work, then `kura add agenvoy`
3. Verifies the `kura` binary landed at `/usr/local/bin/kura`; writes `kuradb_enabled=true` to config.json

### Start / stop flow

`start` runs bare `kura` (which forks into the background itself and returns once ready); `stop` runs `kura stop` (SIGTERM, falls back to SIGKILL after a grace window). Neither touches `kuradb_enabled` — they only affect whether the already-configured daemon is up.

### Disable flow

`tea.ExecProcess` runs `sudo rm -f /usr/local/bin/kura`, then writes `kuradb_enabled=false` to config.json.

## RAG + live-web pairing

`configs/prompts/system_prompt.md` requires that any non-smalltalk information query ground in **both** `search_rag` (when available) and a live-web lookup (`search_web` / `search_google_news`) — RAG is baseline context, live web is mandatory whenever the answer depends on real-world entities or recency. Neither is a fallback for the other; skip both only for smalltalk or pure local-project operations.

## Files & paths

| Path | Purpose |
|---|---|
| `/usr/local/bin/kura` | KuraDB binary (installed by `install.sh`) |
| `~/.config/kuradb/endpoint` | Plaintext URL, written by KuraDB on startup, removed by Agenvoy once health-checks fail |
| `~/.config/kuradb/runtime.uid` | JSON `{uid, pid, started_at}`, read by Agenvoy's `IsRunning()` |
| `~/.config/kuradb/` | KuraDB-side config / data dir (managed by KuraDB itself) |
| Keychain `agenvoy/OPENAI_API_KEY` | Agenvoy's own copy, entered via the `/feature kuradb` wizard |
| Keychain `kuradb/OPENAI_API_KEY` | KuraDB's own copy, kept in sync by `SyncOpenAIKey` since `kura` is a separate process |

Agenvoy's own updater (`static/scripts/update.sh`) also updates `kura` when it's already installed, right before it finishes.

***

> [!NOTE]
> This document was auto-generated by Claude after reading the full source code.
