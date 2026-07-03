# MCP Client

Agenvoy 的 MCP client 讓 agent 得以呼叫任何 MCP server 所暴露的 tool。

## 設定層級

兩層 — session 層覆寫 global 層：

```
~/.config/agenvoy/mcp.json                        <- global
~/.config/agenvoy/sessions/<sid>/mcp.json         <- session-scoped
```

### JSON 格式

```json
{
  "servers": {
    "github": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-github"],
      "env": { "GITHUB_TOKEN": "${GITHUB_TOKEN}" }
    },
    "remote-api": {
      "url": "https://api.example.com/mcp",
      "headers": { "Authorization": "Bearer ${TOKEN}" }
    }
  }
}
```

`command` 與 `url` 互斥。`env`、`headers`、`args` 內的 `${VAR}` / `$VAR` 佔位符會在啟動時以 `os.Expand` 展開。

## Transport 類型

| 類型 | 設定鍵 | 使用情境 |
|---|---|---|
| stdio | `command` + `args` + `env` | 本地 CLI（`npx`、原生 binary） |
| HTTP + SSE | `url` + `headers` | 遠端服務、長時運行的 MCP server |

stdio 使用透過 stdin/stdout 的 line-delimited JSON-RPC。HTTP transport 會針對每個 response 自動偵測 `Content-Type: text/event-stream`，否則退回純 JSON。

## CLI 管理

```bash
agen mcp list             # List all configured MCP servers (global + per-session)
agen mcp add              # Interactive add via promptui
agen mcp remove           # Interactive remove (with scope label)
```

`agen mcp add` 逐步引導：

1. Server 名稱
2. 類型 — Local (stdio) / Remote (HTTP)
3. 類型專屬欄位（command/args/env 或 url/headers）
4. Scope — Global / 選一個 session

Scope 僅寫入一個檔案 — global 寫 `~/.config/agenvoy/mcp.json`，session 寫對應的 `~/.config/agenvoy/sessions/<sid>/mcp.json`。不做跨檔搬移。

## Tool 命名

MCP 暴露的 tool 以下列格式自動註冊：

```
mcp__<server_name>__<tool_name>
```

範例：`mcp__github__create_issue`、`mcp__sqlite-notes__read_query`。

## 結果大小上限

每個 MCP tool 結果上限為 **1 MiB**。超過時，結果會被截斷並附上標記：

```
[mcp output truncated: <total> bytes total, <kept> kept; consider LIMIT / filter / pagination]
```

這避免觸發 OpenAI Responses API 的 10 MB 單一 tool 輸出上限而引發 same-signature 的 retry 風暴。對大型 table 執行 SQLite `SELECT *` 會撞到此限制 — 請加上 `LIMIT` / `WHERE`。

## Confirm 行為

MCP tool 走最保守的預設：

- `agen cli` — 逐一確認每個 MCP tool call
- `agen run` — 自動核准
- 無 per-server `read_only` 開關 — Agenvoy 不對第三方 server 授予信任，因為其行為無法驗證（一個 Slack MCP 可能靜默送出訊息，一個 Filesystem MCP 可能靜默寫入檔案）

批次操作請用 `agen run`。臨時使用則接受逐次 confirm 的成本。

## 生命週期

- **啟動**：`runApp` / `runAgent` 在 `buildAgentRegistry()` **之前**呼叫 `mcp.New(ctx, sid)` 再 `RegisterAll(ctx)`，並註冊 `defer Close()`
- **Per-server 失敗**：server 啟動失敗或 `ListTools` 失敗時記錄 warning 並跳過；絕不阻斷核心功能
- **啟動時快照**：session ID 於首次 resolve 時鎖定；切換 session 需重啟以重新載入 server 清單

## 推薦 server

零認證、本地執行（不需 API key）：

| Server | 用途 |
|---|---|
| `mcp-server-sqlite` | 對本地 `.db` 檔執行 SQL |
| `@modelcontextprotocol/server-memory` | 持久化知識圖譜 |
| `@playwright/mcp` | 瀏覽器自動化（會下載 chromium） |
| `@modelcontextprotocol/server-postgres` | 本地 Postgres 連線 |
| `mcp-server-time` | 時區轉換 / 相對時間 |

避免註冊能力與內建 tool 重疊的 MCP server（例如 `filesystem`、`git`、`fetch`、`shell`）— 重複只會膨脹 LLM 的 tool 清單。
