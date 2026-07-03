# Session 與 Agent

## Session

Session 是 Agenvoy 的核心單位。每個 session 擁有自己的對話 context、記憶、agent 人格與工具配置。

儲存路徑：`~/.config/agenvoy/sessions/<sid>/`

| 檔案 | 用途 |
|---|---|
| `bot.md` | Agent 人格定義（YAML frontmatter + markdown 內文） |
| `status.json` | 目前執行狀態與 active task 清單 |
| `action.log` | 工具呼叫稽核日誌；達 1 MB 時 rotate（截斷至 768 KB） |
| `mcp.json` | Session 範圍的 MCP server 配置 |

History、summary 與 config flag 存於 ToriiDB（`DBSessionHist`、`DBSessionSummary`、`DBConfig`），而非 per-session JSON。

### Session prefix 與生命週期

| Prefix | 生命週期 |
|---|---|
| `cli-*` | 永久（由 `agen session new` / `make cli` 建立） |
| `http-*` | 永久（由 `POST /v1/send` 帶 `persist=true` 建立） |
| `dc-*` | 永久（Discord channel） |
| `tg-*` | 永久（Telegram chat — per-chat，該 chat 內所有使用者共用） |
| `temp-*` | idle 30 分鐘後回收（`POST /v1/send` 與 subagent session 的預設值） |

清理透過 cron 每 30 分鐘執行一次（並於啟動時執行一次），僅針對 `temp-*` prefix — `cli-*`、`http-*`、`dc-*` 與 `tg-*` 永不自動回收。

## bot.md — Agent 人格

每個 session 可宣告自己的人格：

```markdown
***
name: mobile-builder
***

You are an expert mobile application architect specializing in
SwiftUI, Jetpack Compose, and React Native...
```

Frontmatter 的 `name` 同時作為查找鍵（`GetSessionIDByName`）；內文於每一輪 render 進 system prompt 的 `## Bot Persona` 區塊。`agen session config` 會在 `$EDITOR` 中開啟目前 session 的 bot.md。

## Agent routing

三種方式決定由哪個 agent 處理任務：

**1. 自動** — dispatcher LLM 分析輸入，並透過 `SelectAgent()` 挑選最合適的 provider。

**2. `:name` 一次性覆寫**（CLI / TUI）— 在任何輸入前加上 `:session-name`，即可將單一命令派送至指定 session，**而不**改變 primary pointer：

```
:mobile-builder build me a SwiftUI login screen
```

`exec.Run` 中的解析順序：`:bot` → `MatchExternal`（`/claude` 等）→ `MatchSkillCall`（`/skill-name`）→ `Execute`。`:name` 覆寫於 `exec.Run`（CLI/TUI）與 Telegram runtime 中解析（後者會剝除 prefix，並在找不到名稱時附帶 metadata 註記 fallback）；HTTP `POST /v1/send` 與 Discord 不解讀該 prefix。

**3. `invoke_subagent` 工具** — agent 於執行期間 in-process（無 HTTP）呼叫另一個 agent，從 parent ctx 繼承 `AllowAll` 與 `WorkDir`。強制排除集為 `{invoke_subagent, invoke_external_agent, cross_review_with_external_agents, review_result}`；`ask_user` **不**被排除 — subagent 可透過共用的 pending registry 向使用者提問。

## Permission mode

| Mode | 行為 |
|---|---|
| `single-confirm` | 每個非 ReadOnly 工具呼叫都需使用者確認（`agen cli` 的預設值） |
| `always-allow` | 工具自動執行；LLM 被指示對七類真正不可逆的操作先呼叫 `ask_user` |

在 `always-allow` 下仍需明確 `ask_user` 的七類不可逆操作：

1. 對已有內容的目錄執行 `rm -rf`
2. `DROP TABLE` / `DROP DATABASE`
3. `git push --force` 至 `main`
4. 對系統路徑執行 `chmod 777`
5. 覆寫尚未讀取過的非空檔案
6. 雲端資源刪除
7. 對系統 process 執行 `shutdown` / `kill -9`

此閘門由 system prompt 強制，而非硬編碼的 Go 端 filter — 新增類別只需編輯 `configs/prompts/`。

## Per-session 併發

`MAX_SESSION_TASKS`（預設 `3`，硬上限 `10`）限制單一 session 可同時執行的 `Execute()` 呼叫數。超額的呼叫者透過 `EnterConcurrent(sid)` 等待，並在有空位後才出現於 `status.json`。
