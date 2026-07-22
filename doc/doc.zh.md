# Agenvoy - 技術文件

> 返回 [README](./README.zh.md)

## 前置需求

- Go 1.25.1 或更新版本
- macOS 或支援 Go、SQLite 與 `go-pkg/sandbox` 相依套件的環境
- 至少一組模型供應商憑證；透過 TUI 設定 API key 或 OAuth
- Telegram、Discord、語音、圖片與 KuraDB 功能各自需要對應憑證

## 安裝

### 官方安裝程式

```bash
curl -fsSL https://agenvoy.com/scripts/install.sh | bash
agen
```

### 從原始碼建置

```bash
git clone https://github.com/pardnchiu/agenvoy.git
cd agenvoy
go build -tags fts5 -ldflags "-X github.com/pardnchiu/agenvoy/internal/runtime/tui.projectVersion=dev" -o agen ./cmd/app/
./agen
```

### 使用 Makefile

```bash
make build
agen
```

`make build` 會將二進位檔安裝至 `/usr/local/bin/agen`，因此需要 `sudo` 權限。

### 直接執行

```bash
go run ./cmd/app/
```

## 設定

Agenvoy 使用 `~/.config/agenvoy/` 保存執行期資料，並將憑證存放於作業系統 keychain；不要將 API key 或 token 寫入專案檔案或 Git。

### 常用憑證

| Keychain 項目 | 用途 |
|---|---|
| `OPENAI_API_KEY` | OpenAI 與 KuraDB |
| `GEMINI_API_KEY` | Gemini、語音轉錄與圖片功能 |
| `CLAUDE_API_KEY`、`GROK_API_KEY`、`DEEPSEEK_API_KEY` | 對應模型供應商 |
| `TELEGRAM_TOKEN`、`DISCORD_TOKEN` | 聊天機器人整合 |

### Runtime 設定

主要設定檔是 `~/.config/agenvoy/config.json`。`limits` 欄位由程式載入，缺漏欄位會自動補上內建預設值。

| 設定 | 預設值 | 說明 |
|---|---:|---|
| `limits.port` | `17989` | 本機 HTTP daemon 連接埠 |
| `limits.max_tool_iterations` | `128` | 單次 Agent 工作的工具迭代上限 |
| `limits.agent_send_timeout_seconds` | `600` | 模型請求逾時秒數 |
| `limits.max_history_messages` | `8` | 保留的近期歷史訊息數 |
| `limits.max_session_tasks` | `3` | 每個 session 的並行工作數上限 |

```json
{
  "limits": {
    "port": "17989",
    "max_tool_iterations": 128,
    "agent_send_timeout_seconds": 600
  }
}
```

### MCP Client

在 `~/.config/agenvoy/mcp.json` 登錄 stdio 或 streamable HTTP MCP server：

```json
{
  "servers": {
    "local-tools": {
      "command": "node",
      "args": ["/absolute/path/server.js"]
    },
    "remote-tools": {
      "url": "http://127.0.0.1:8000/mcp",
      "headers": {"Authorization": "Bearer ${MCP_TOKEN}"}
    }
  }
}
```

## 使用方式

### 啟動 TUI

```bash
agen
```

TUI 可管理 session、模型、Skill、工具權限、MCP，以及 Telegram、Discord、語音與 KuraDB 整合。

### 互動與自動執行

```bash
# 每次工具呼叫均需確認
agen cli '找出目前專案的 Go 模組並摘要說明'

# 僅本次工作自動允許工具
agen run '檢查最近 Git 變更並產生摘要'
```

`run` 不會繞過 sandbox、denied-path 規則、工具排除或 runtime limits。

### 管理 daemon

```bash
agen stop
agen update
```

直接啟動 `agen` 時，TUI 會在需要時啟動本機 daemon。

### stdin MCP Server

當 stdin 不是終端機時，Agenvoy 會啟動 newline-delimited JSON-RPC MCP server：

```bash
printf '%s\n' \
  '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}' \
  '{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}' | agen
```

支援 `initialize`、`notifications/initialized`、`tools/list`、`tools/call` 與 `ping`。

### HTTP API

Daemon 預設只監聽 `127.0.0.1:17989`：

```bash
curl --fail-with-body -sS \
  -H 'Content-Type: application/json' \
  -d '{"content":"列出目前可用工具","persist":false,"allow_all":false}' \
  http://127.0.0.1:17989/v1/send
```

`/v1/chat/completions` 為 stateless endpoint；需在每次請求中帶入延續對話所需的 `messages`。

## 命令列參考

| 指令 | 語法 | 說明 |
|---|---|---|
| TUI | `agen` | 開啟或連接本機 daemon 的互動式 TUI |
| 互動執行 | `agen cli <input...>` | 工具執行前要求確認 |
| 自動執行 | `agen run <input...>` | 本次工作自動允許工具 |
| 停止 | `agen stop` | 停止 daemon |
| 更新 | `agen update` | 執行官方更新腳本 |
| Daemon | `agen --daemon` | 直接啟動 daemon |
| MCP | 非 TTY stdin 的 `agen` | 從 stdin 提供 MCP JSON-RPC |

## HTTP API 參考

| Method | Path | 說明 |
|---|---|---|
| `POST` | `/v1/send` | 執行 Agent，支援 SSE、session、model 與工具排除 |
| `POST` | `/v1/chat/completions` | OpenAI 相容且 stateless 的 chat completions |
| `GET` | `/v1/tools` | 列出工具 |
| `POST` | `/v1/tool/:tool_name` | 直接呼叫工具 |
| `GET` | `/v1/sessions` | 列出 session 與狀態 |
| `GET` | `/v1/models` | 列出模型 |
| `GET` | `/v1/session/:session_id/status` | 查詢 session 狀態與用量 |
| `GET` | `/v1/session/:session_id/pending` | 列出待完成工作 |

## 工具參考

| 工具 | 用途 |
|---|---|
| `read_files`、`list_files`、`glob_files`、`search_files` | 批次讀取、列舉與搜尋檔案 |
| `write_file`、`patch_file` | 建立、完整覆寫或精準修改檔案 |
| `run_command` | 在 shell 驗證與 sandbox 約束下執行命令 |
| `ask_user`、`write_todo` | 互動式輸入與多步驟進度追蹤 |
| `search_tools` | 搜尋已註冊工具 |
| `invoke_subagent` | 委派單一子任務 |

## 架構入口

請參閱完整的 [Architecture](./architecture.md) 與繁體中文 [架構](./architecture.zh.md)。

***

©️ 2026 [邱敬幃 Pardn Chiu](https://www.linkedin.com/in/pardnchiu)
