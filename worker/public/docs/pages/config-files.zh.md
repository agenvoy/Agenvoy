# 設定

## 檔案佈局

```
~/.config/agenvoy/
├── config.json                       Main config (active session, dispatcher_model, kuradb_enabled, t_enabled, d_enabled, compats[])
├── usage.json                        Token usage tracker
├── runtime.uid                       Server-mode singleton lock (daemon-only writer)
├── mcp.json                          Global MCP servers
├── allow_skill                       Global skill always-allow list (one name per line)
├── .telegram                         Authorized Telegram chat IDs (one per line, written after OTP success)
├── scheduler/
│   ├── tasks.json                    One-shot scheduled tasks
│   └── crons.json                    Recurring cron tasks
├── download/                         Inbound chat attachments + outbound generated images (agenvoy-img-<uuid>.png)
├── skills/scheduler/                 Isolated scheduler skill dirs (<short>-<hash8>/SKILL.md)
└── sessions/
    └── <sid>/
        ├── bot.md                    Agent persona (frontmatter + body)
        ├── status.json               Active task list / state
        ├── action.log                Tool call audit trail (1 MB rotate, 768 KB target; foreign-process lines prefixed)
        ├── summary.meta.json         {last_message_time: YYYY-MM-DD HH:MM:SS} — incremental summary cursor
        ├── input_history             Per-session TUI input history
        └── mcp.json                  Session-scoped MCP servers

~/.config/kuradb/
└── endpoint                          Plaintext URL (random port), written by KuraDB on spawn, removed on disable

<project-root>/.agenvoy/
└── allow_skill                       Project-scoped skill always-allow list (union with global at load time)
```

History、summaries、error memory 與 config flags 住在 ToriiDB，位於 `~/.config/agenvoy/.store/`（由 ToriiDB 自身管理，非使用者直接可編輯）。

## 專案設定

```
configs/
├── jsons/
│   ├── providors/                    Provider catalogs (note: spelling is intentional)
│   │   ├── claude.json
│   │   ├── openai.json
│   │   ├── codex.json
│   │   ├── gemini.json
│   │   ├── copilot.json
│   │   └── nvidia.json
│   ├── denied_map.json               Sandbox denied paths
│   ├── exclude_list.json             Listing/walking exclude paths
│   └── white_list.json               Allowed paths
└── prompts/
    ├── system_prompt.md              Main system prompt template
    ├── skill_execution.md            Skill execution discipline
    ├── summary_prompt.md             Summary generation prompt
    ├── summary_merge_prompt.md       Summary merge prompt
    ├── summary_context.md            Summary context injection
    ├── discord_system_prompt.md      Discord interface system prompt
    └── telegram_system_prompt.md     Telegram interface system prompt
```

`compat` provider 條目在透過 `agen model add` 加入後，會與靜態 catalog 並存。

## 環境變數

透過 `cmd/app/main.go init()` 中的 `godotenv` 從 repo-root 的 `.env` 載入。

| 變數 | 必填 | 預設 | 說明 |
|---|---|---|---|
| `MAX_HISTORY_MESSAGES` | 否 | `16` | 每個 turn 送出的最大 history 訊息數 |
| `MAX_TOOL_ITERATIONS` | 否 | `16` | 每次請求的 tool-call 迭代上限 |
| `MAX_SKILL_ITERATIONS` | 否 | `128` | skill 執行期間的 tool-call 迭代上限 |
| `MAX_EMPTY_RESPONSES` | 否 | `8` | 放棄前可容忍的連續空回應數 |
| `MAX_SESSION_TASKS` | 否 | `3`（上限 `10`） | 每個 session 的並行上限；超額 tasks 進 queue |
| `MAX_SUBAGENT_TIMEOUT_MIN` | 否 | `10`（上限 `60`） | `invoke_subagent` 總 timeout（分鐘） |
| `MAX_EXTERNAL_AGENT_TIMEOUT_MIN` | 否 | `10`（上限 `60`） | 外部 CLI subprocess timeout（分鐘） |
| `AGENT_SEND_TIMEOUT_SECONDS` | 否 | `600` | Exec 層對 `Agent.Send` 的上限；在 provider 呼叫外層包 `context.WithTimeout`。主要與 codex SSE 相關（10m client timeout）；非 SSE provider 則 `Client.Timeout=5m` 會先觸發 |
| `OPENAI_API_KEY` | 否 | — | 啟用透過 `text-embedding-3-small` 的語意搜尋與 KuraDB embedding |

外部 CLI agents（`codex` / `gh` / `claude` / `gemini`）透過 `exec.LookPath` 自動偵測；把 binary 裝到 `PATH` 上即可啟用，無需 env flag。

數值變數會 clamp 到文件列出的上限；值 `<= 0` 則 fallback 到預設。

## bot.md 格式

```markdown
***
name: <session display name>     # used by :name routing and invoke_subagent name param
***

<persona content as free-form markdown>
```

本體會在每個 turn 渲染進 system prompt 的 `## Bot Persona` 區塊。未設定時 frontmatter `name` 預設為 session id。

`agen session new <name>` 會同時寫入 session 目錄與一個 `name` 等於 `<name>` 的 bot.md。`agen session switch <name>` 依 bot.md 的 `name` 查找 session（僅 frontmatter，不 fallback 到 sid）。

## Permission mode

Active 的 permission mode（`single-confirm` 相對於 `always-allow`）由 entry point 決定：

| Entry | Mode |
|---|---|
| `agen cli` | `single-confirm`（`AllowAll=false`） |
| `agen run` | `always-allow`（`AllowAll=true`） |
| Discord / REST | `always-allow` |
| Telegram | `single-confirm`（`AllowAll=false`；confirm gate 使用 Telegram inline-keyboard SendSelect） |
| Subagent | 繼承 parent ctx |

該 mode 會渲染進 system prompt 的 `## Permission Mode` 之下。沒有全域 env var 可覆寫它。

## MCP 設定

兩層；session 覆寫 global。完整 schema 與 `${VAR}` 展開行為請見 MCP Integration 頁面。

## Provider 設定

Provider 定義住在 `configs/jsons/providors/`（拼寫為刻意）。Credentials 永不住在 JSON — 它們住在 OS keychain 的 `agenvoy` service 下。

### Compat provider URL 儲存拆分

`compat` provider URL 採用**雙儲存**模型：

| 內容 | 位置 | 原因 |
|---|---|---|
| URL（例如 `http://host:8000/v1`） | `~/.config/agenvoy/config.json` 的 `compats[].URL` | 非機密、使用者可編輯 |
| API key（`COMPAT_<NAME>_API_KEY`） | OS keychain | 機密 |

URL 慣例遵循 Zed：使用者輸入 URL 到 `/v1` 為止（例如 `http://localhost:11434/v1`），`compat/send.go` 只附加 `/chat/completions`。`compat.New` 透過 `session.GetCompatURL(instanceName)` 讀取 URL — **不是** keychain。沒有 `COMPAT_<NAME>_URL` 的 keychain key（刻意移除：曾有一個歷史 bug，TUI 寫入 config 而 runtime 讀 keychain，總是 fallback 到 localhost）。

## KuraDB

Enabled 狀態為 config.json 中的 `kuradb_enabled: bool`。透過 TUI 的 `/feature kuradb` 切換（無 CLI 子命令 — install.sh + sudo 需要真實 TTY）。完整生命週期請見 KuraDB RAG 頁面。

| Key | 位置 |
|---|---|
| `kuradb_enabled` | `config.json` |
| `OPENAI_API_KEY` | keychain（`agenvoy` service）— 與語意搜尋共用 |
| Endpoint URL（runtime） | `~/.config/kuradb/endpoint`（明文，每次 spawn 隨機 port） |
| Binary | `/usr/local/bin/kura`（install.sh 中 hardcoded） |

## Telegram / Discord 啟用

| Key | 位置 |
|---|---|
| `telegram_enabled` / `discord_enabled` | `config.json` |
| `TELEGRAM_TOKEN` / `DISCORD_TOKEN` | keychain（`agenvoy` service） |
| Authorized chat IDs | `~/.config/agenvoy/.telegram`（每行一個 chat ID，於 6 位數 OTP 驗證成功後寫入） |
| Authorized Discord channels | 透過 guild mention + per-server 的 `d_allowed` config 設定 |

## 刻意不放置之處

一些刻意的「非位置」：

- **Provider API keys** — 永不在 `config.json`；一律在 keychain
- **MCP credentials** — 在 `mcp.json` 使用 `${VAR}` placeholders，並把實際值放進 env vars（或透過 shell init 放進 keychain）
- **`store_secret` 擷取的 secrets** — 只落在 keychain；永不進入 LLM context、history、action.log 或 tool args
- **Session history** — 在 ToriiDB，永不在 per-session JSON 檔（此於 ToriiDB v0.5.0 migration 變更）
- **Tool call 結果** — 僅 in-memory cache；重啟後不持久化（除了透過 error_memory 與 conversation_history）
