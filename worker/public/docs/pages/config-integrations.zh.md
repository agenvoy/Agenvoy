# Integration Config

## MCP config

兩層；session 覆寫 global。完整 schema 與 `${VAR}` 展開行為見 MCP Integration。

## Provider config

Provider 定義位於 `configs/jsons/providors/`（拼法為刻意保留）。憑證絕不存於 JSON — 而存於 OS keychain 的 `agenvoy` service 下。

### Compat provider URL 儲存拆分

`compat` provider URL 採 **two-storage** 模型：

| 內容 | 位置 | 原因 |
|---|---|---|
| URL（例如 `http://host:8000/v1`） | `~/.config/agenvoy/config.json` 的 `compats[].URL` | 非機密、可由 user 編輯 |
| API key（`COMPAT_<NAME>_API_KEY`） | OS keychain | 機密 |

URL 慣例遵循 Zed：user 輸入至 `/v1` 為止的 URL（例如 `http://localhost:11434/v1`），`compat/send.go` 僅附加 `/chat/completions`。`compat.New` 透過 `session.GetCompatURL(instanceName)` 讀取 URL — **非** keychain。不存在 `COMPAT_<NAME>_URL` keychain key（刻意移除：一個歷史 bug 曾讓 TUI 寫入 config 而 runtime 讀 keychain，導致總是 fallback 到 localhost）。

## KuraDB

啟用狀態為 config.json 中的 `kuradb_enabled: bool`。透過 TUI 的 `/feature kuradb` 切換（無 CLI 子指令 — install.sh + sudo 需要真實 TTY）。完整生命週期見 KuraDB RAG。

| Key | 位置 |
|---|---|
| `kuradb_enabled` | `config.json` |
| `OPENAI_API_KEY` | keychain（`agenvoy` service） — 與 semantic search 共用 |
| Endpoint URL（runtime） | `~/.config/kuradb/endpoint`（明文，每次 spawn 隨機 port） |
| Binary | `/usr/local/bin/kura`（install.sh 中 hardcode） |

## Telegram / Discord 啟用

| Key | 位置 |
|---|---|
| `telegram_enabled` / `discord_enabled` | `config.json` |
| `TELEGRAM_TOKEN` / `DISCORD_TOKEN` | keychain（`agenvoy` service） |
| 授權的 chat ID | `~/.config/agenvoy/.telegram`（每行一個 chat ID，於 6 位數 OTP 驗證成功後寫入） |
| 授權的 Discord channel | 透過 guild mention + per-server `d_allowed` config 設定 |

## 哪些東西刻意**不**存放於此

一些刻意的非儲存位置：

- **Provider API key** — 絕不存於 `config.json`；一律存 keychain
- **MCP 憑證** — 在 `mcp.json` 中使用 `${VAR}` 佔位符，實際值放在 env var（或透過 shell init 存入 keychain）
- **由 `store_secret` 捕獲的機密** — 僅落於 keychain；絕不進入 LLM context、history、action.log 或 tool args
- **Session history** — 存於 ToriiDB，絕不存於 per-session JSON 檔（此於 ToriiDB v0.5.0 遷移時變更）
- **Tool call 結果** — 僅 in-memory cache；不跨重啟持久化（error_memory 與 conversation_history 除外）
