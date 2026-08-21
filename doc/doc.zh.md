# Agenvoy - 技術文件

> 返回 [README](./README.zh.md)

## 前置需求

- Go 1.25.1 或更新版本
- macOS 或支援 Go、SQLite 與 `go-pkg/sandbox` 相依套件的環境
- 至少一組模型供應商憑證；透過 TUI 設定 API key 或 OAuth

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
| `CLAUDE_API_KEY`、`GROK_API_KEY`、`DEEPSEEK_API_KEY` | 對應模型供應商 |
| `TELEGRAM_TOKEN`、`DISCORD_TOKEN` | 聊天機器人整合 |

### Runtime 設定

主要設定檔是 `~/.config/agenvoy/config.json`。`limits` 欄位由程式載入，缺漏欄位會自動補上內建預設值。

| 設定 | 預設值 | 說明 |
|---|---:|---|
| `limits.max_tool_iterations` | `128` | 單次 Agent 工作的工具迭代上限 |
| `limits.agent_send_timeout_seconds` | `600` | 模型請求逾時秒數 |
| `limits.max_history_messages` | `24` | 保留的近期歷史訊息數 |
| `limits.max_history_bytes` | `5242880` | 歷史訊息大小上限（位元組） |

套件內建預設值（目前不會從 `config.json` 讀取）：

| 常數 | 預設值 | 說明 |
|---|---:|---|
| `MaxSessionTasks` | `NumCPU × 4` | 每個 session 的並行工作數上限；超出的任務排隊等待而非失敗 |
| `MaxSubagentTimeoutMin` | `30` | Subagent 逾時分鐘數 |
| `MaxResumeWaitMin` | `60` | Pending resume 等待回答的分鐘數 |

```json
{
  "limits": {
    "max_tool_iterations": 128,
    "agent_send_timeout_seconds": 600
  }
}
```

### TUI 執行模式

當輸入區為空時，按下 `Shift+F` 可切換 fast mode。啟用時，標題列會顯示 `[fast]`。Fast mode 只存在於目前行程，不會保存至 `config.json`；它會透過 `go-llm-router` v0.4.0 傳遞 `provider.ModeFast`，讓支援的 provider backend 要求更快速的服務層級。關閉 fast mode 時則使用預設模式。

### 圖像生成已移除

圖像生成支援目前暫時移除，等待 router 整合重新設計。`image2` 指令、`enable_image2` 設定旗標、`generate_image` 工具及相關註冊路徑目前均不可用。

- 從既有設定中移除 `enable_image2`。
- 停止呼叫 `/image2` 與 `generate_image`。
- 若仍需要圖像輸出，請使用外部圖像生成整合。

### MCP Client

MCP client 與 server 位於 `internal/runtime/mcp`，並使用官方 [`modelcontextprotocol/go-sdk`](https://github.com/modelcontextprotocol/go-sdk)。在 `~/.config/agenvoy/mcp.json` 登錄 stdio 或 streamable HTTP MCP server。Client 會訂閱工具清單變更通知，遠端 server 更新目錄時會重新註冊工具；server instructions 會注入 agent system prompt：

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

Daemon 只監聽 `127.0.0.1:17989`，連接埠固定不可調整；Open WebUI 部署後同樣固定在 `17990`，由 `/webui` 反向代理：

```bash
curl --fail-with-body -sS \
  -H 'Content-Type: application/json' \
  -d '{"content":"列出目前可用工具","persist":false,"allow_all":false}' \
  http://127.0.0.1:17989/v1/send
```

`/v1/chat/completions` 為 stateless endpoint；需在每次請求中帶入延續對話所需的 `messages`。`reasoning_effort` 接受 `none` `low` `medium` `high` `xhigh` `max`（另支援別名 `minimal` `extra` `ultra`）；未帶或無法識別的值退回該 session 的 reasoning 設定。

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

Daemon 只綁定 `127.0.0.1`。標示 **local** 的 endpoint 另外要求請求來源必須是 `127.0.0.1`／`::1`（`localhostOnly()` 守衛）——這些會動到 credential、設定檔或行程生命週期，設計上是給同機器的 dashboard 用，不是給遠端 client 呼叫。

**Agent 執行**

| Method | Path | 說明 |
|---|---|---|
| `POST` | `/v1/send` | 執行 Agent |
| `POST` | `/v1/chat/completions` | OpenAI 相容且 stateless 的 chat completions |
| `GET` | `/v1/log` | SSE：跨所有 session 的事件串流 |
| `GET` | `/v1/tools` | 列出工具 |
| `POST` | `/v1/tool/:tool_name` | 直接呼叫工具 |

**模型**

| Method | Path | 說明 |
|---|---|---|
| `GET` | `/v1/models` | 列出已註冊模型 |
| `POST` `DELETE` | `/v1/models` `/v1/models/*name` | **local** — 新增／移除模型 |
| `GET` `POST` | `/v1/model/dispatcher` | **local** — 讀取／設定 dispatcher 模型 |
| `GET` `POST` | `/v1/model/summary` | **local** — 讀取／設定 summary 模型 |

**Session**

| Method | Path | 說明 |
|---|---|---|
| `GET` | `/v1/sessions` | 列出 session 與狀態 |
| `POST` `PUT` `DELETE` | `/v1/session` | **local** — 建立／重新命名／刪除 session |
| `POST` | `/v1/session/:id/model` | 設定該 session 的模型 |
| `GET` | `/v1/session/:id/status` | 查詢 session 狀態與用量 |
| `GET` | `/v1/session/:id/log` | SSE：單一 session 的事件串流 |
| `POST` | `/v1/session/:id/event` | **local** — 對某 session 的事件串流手動發布事件 |
| `GET` | `/v1/session/:id/pending` | 列出待完成（`ask_user`／confirm）工作 |
| `GET` | `/v1/session/:id/pending/:task_hash/questions` | 取得待完成工作的問題內容 |
| `POST` | `/v1/session/:id/pending/:task_hash/resume` | 回答待完成工作並恢復執行 |
| `DELETE` | `/v1/session/:id/pending/:task_hash` | 直接捨棄待完成工作，不回答 |
| `POST` | `/v1/session/:id/cancel/:task_id` | 取消單一執行中的任務。`task_id` 由 `/status` 取得；設計上只做逐一取消，沒有一次砍全部的版本 |
| `GET` `POST` | `/v1/session/:id/persona` | **local** — 讀取／設定 session persona |
| `POST` | `/v1/session/:id/compact` | **local** — 背景壓縮歷史（fire-and-forget,立即回 `202 Accepted`） |
| `GET` | `/v1/session/:id/daemon` | **local** — `daemon.log` 中提到該 sessionID 的行（best-effort grep,非真正的 per-session 檔） |
| `GET` | `/v1/session/:id/action` | **local** — 該 session 的 `action.log` 全文 |
| `GET` | `/v1/session/:id/usage` | **local** — 24h/7d/28d 各模型 token 用量（與 TUI `/usage` 畫面同一套聚合邏輯） |
| `GET` | `/v1/session/:id/history` | **local** — 列出已歸檔的完成 pending task 檔案 |
| `GET` | `/v1/session/:id/history/*file` | **local** — 讀取單一歸檔的 pending task 檔案 |

**Channel**

| Method | Path | 說明 |
|---|---|---|
| `GET` | `/v1/channel/status` | **local** — Telegram／Discord 啟用狀態、bot 使用者名稱、是否已存 token |
| `POST` | `/v1/channel/telegram` `/v1/channel/discord` | **local** — `{action:"enable"\|"disable", token?}`。enable 只存 token 並切換設定 flag,刻意不做 TUI 那套 `GetMe` 驗證——daemon 既有的設定檔監看機制會自動重連 bot 並填回使用者名稱 |

**檔案與憑證**

| Method | Path | 說明 |
|---|---|---|
| `GET` `PUT` | `/v1/file` | **local** — 讀取／寫入檔案 |
| `GET` | `/v1/file/open` | **local** — 以系統預設程式開啟檔案／URL |
| `GET` `DELETE` | `/v1/key` | **local** — 查詢／刪除 keychain 中單一憑證 |
| `GET` `POST` | `/v1/keys` | **local** — 列出／設定憑證 |

**Provider**

| Method | Path | 說明 |
|---|---|---|
| `GET` | `/v1/providers` | **local** — 列出 provider 及其可用操作 |
| `GET` | `/v1/provider/:provider/check` | **local** — 該 provider 是否已有憑證 |
| `POST` | `/v1/provider/:provider/key` | **local** — 設定 API key |
| `GET` | `/v1/provider/:provider/oauth` | **local** — SSE device-code OAuth 流程 |
| `GET` | `/v1/provider/:provider/models` | **local** — 列出該 provider 可用模型 |

**MCP**

| Method | Path | 說明 |
|---|---|---|
| `GET` `POST` | `/v1/mcp` | **local** — 列出／新增 MCP server |
| `POST` | `/v1/mcp/remove` | **local** — 移除 MCP server |
| `GET` | `/v1/mcp/status` | **local** — 各 server 連線狀態 |
| `GET` | `/v1/mcp/health` | **local** — 各 server health probe |
| `POST` | `/v1/mcp/reconnect` | **local** — 重連全部 MCP client 並重新註冊工具 |

**排程與自動化**

| Method | Path | 說明 |
|---|---|---|
| `GET` | `/v1/schedule/*skill` | **local** — 讀取 scheduler skill 內容 |
| `GET` `DELETE` | `/v1/cron` | **local** — 列出／刪除 cron 項目 |
| `POST` | `/v1/cron/run` | **local** — 立即觸發 cron 項目（`202 Accepted`） |
| `GET` `DELETE` | `/v1/task` | **local** — 列出／刪除單次任務 |
| `POST` | `/v1/task/run` | **local** — 立即觸發任務（`202 Accepted`） |

**KuraDB 與白名單**

| Method | Path | 說明 |
|---|---|---|
| `GET` `POST` | `/v1/kuradb` | **local** — 查詢狀態／enable／disable／start／stop／restart。安裝／解除安裝仍只走 TUI（需要真實終端機處理 `sudo`／安裝腳本的互動提示）,此 API 只切換 `enabled` flag 並控制已安裝好的 `kura` 行程 |
| `GET` `POST` | `/v1/allowlist/cmd` | **local** — 列出／新增指令白名單（append-only,需重啟 daemon 生效） |
| `GET` `POST` | `/v1/allowlist/skill` | **local** — 列出／切換 skill 白名單（`scope=global\|project`） |

**查閱**

| Method | Path | 說明 |
|---|---|---|
| `GET` | `/v1/torii/error` | **local** — 查閱工具錯誤記憶；`tool`／`keyword` 皆未帶時回傳無過濾全表掃 |

## 工具參考

註冊表內建 25 個工具，另有 3 個在前置條件成立時才註冊。涵蓋多種相關動作的工具以 `mode` 參數區分，而不是拆成多個名稱。

| 分類 | 工具 | 用途 |
|---|---|---|
| 工具系統 | `find_tools` | 發現既有工具並載入其 schema（`mode=search\|list`） |
| | `edit_tool` | 建立、修正或丟棄工具定義（`mode=write\|patch\|remove`） |
| | `test_tool` | 上線前在沙箱內執行 script 工具 |
| Skill | `run_skill` | 載入具名 skill 的參考素材 |
| | `edit_skill` | 編寫 skills 目錄底下的檔案（`mode=write\|patch\|remove`） |
| 排程 | `schedules` | 查詢、改期或取消定時與週期任務（`mode=list\|patch\|remove\|write`） |
| 檔案 | `find_files` | 以目錄、檔名樣式或內容定位（`mode=list\|glob\|search`） |
| | `read_files` | 批次讀取文字、PDF、DOCX、PPTX、CSV 與圖片 |
| | `edit_file` | 建立、修改、移置或還原檔案（`mode=write\|patch\|remove\|restore`） |
| | `file_history` | 工具改過的每個檔案的版本紀錄（`mode=list\|read`） |
| 執行環境 | `run_command` | 在工作目錄以沙箱約束執行二進位 |
| | `open_file` | 以系統預設應用開啟檔案 |
| | `download_file` | 下載二進位資產至磁碟 |
| | `install_dependence` | 安裝缺少的系統執行檔 |
| Agent 協調 | `subagents` | 將子任務委派到獨立 session（`mode=invoke\|list`） |
| | `write_todo` | 使用者即時看得到的任務清單 |
| | `ask_user` | 暫停提問，回答後自動續跑 |
| 網路 | `search_web` | DuckDuckGo 結果與 Google News 標題一次取得 |
| | `fetch_page` | 取得完整頁面內容（markdown／html／json） |
| | `http_request` | 原始 HTTP 呼叫，含 multipart 上傳 |
| 狀態 | `chat_history` | 本 session 的執行紀錄與對話（`mode=list\|read\|search`） |
| | `error_history` | 跨 session 保留的工具失敗紀錄（`mode=search\|read\|write`） |
| | `reasoning_guide` | 依 `topic` 取得完整推理規則 |
| 基礎支援 | `calculate` | 算術、單位與匯率換算 |
| | `store_secret` | 遮蔽輸入並存入 keychain |
| 條件註冊 | `transcribe_media` | 音訊／影片轉文字——需 `GEMINI_API_KEY` |
| | `list_chatbot`、`send_to_chatbot` | 跨頻道推送——需啟用 Telegram 或 Discord |

只有 `find_tools`、`reasoning_guide`、`run_command`、`find_files`、`read_files` 會帶完整 schema 送出；其餘工具初始只送名稱與描述，參數在首次使用時經 `find_tools(mode=search)` 載入，讓初始工具 payload 維持在完整註冊表的三分之一左右。

## 架構

請參閱完整的 [Architecture](./architecture.md) 與繁體中文 [架構](./architecture.zh.md)。

## License

本專案以 [Apache License 2.0](../LICENSE) 授權。

***

©️ 2026 [邱敬幃 Pardn Chiu](https://www.linkedin.com/in/pardnchiu)
