# MCP Server

透過 stdio pipe 啟動時，`agen` 會作為 MCP server 執行。任何相容 MCP 的 agent（Claude Code、Codex、OpenCode 等）都可連線並使用 Agenvoy 的工具 — 包含即時建立新工具。

## 外部 agent 能獲得什麼

- **沙箱化執行** — 所有 script 工具都在 OS 原生 sandbox（macOS `sandbox-exec` / Linux `bwrap`）內執行，隔離 `~/.ssh`、`~/.aws`、`.env`、`*.pem` 等敏感路徑
- **自動建立工具** — 當現有工具都無法涵蓋需求時，agent 呼叫 `script_tool_generate_guide` 取得 build contract，再 `write_tool` → `test_tool` 建立新的 Python script 工具。工具會被持久化並可跨 session 重用
- **共享工具庫** — 任何 agent（Agenvoy 內部、Claude Code、Codex 等）建立的工具都會存至 `~/.config/agenvoy/tools/script/`，並對所有已連線的 agent 可用。建立一次，處處可用
- **即時資料存取** — `api_public_api_list` 索引免費的公開 API；agent 挑選其一，圍繞它 scaffold 出一個 script 工具，並以真實資料而非訓練知識的臆測作答

## 快速設定

TUI：`/mcp install` → 選擇你的 agent

各 agent 的手動配置：

**Claude Code** — `~/.claude.json`
```json
{ "mcpServers": { "agenvoy": { "command": "agen" } } }
```

**Codex** — `~/.codex/config.toml`
```toml
[mcp_servers.agenvoy]
command = "agen"
```

**OpenCode** — `~/.config/opencode/opencode.jsonc`
```json
{ "mcp": { "agenvoy": { "type": "local", "command": ["agen"] } } }
```

## 通用 MCP client 設定

對於上方未列出的任何 MCP client，唯一的要求是：

- **Transport**：stdio（透過 stdin/stdout 的 JSON-RPC）
- **Command**：`agen`
- **Args**：無
- **前置條件**：`agen` binary 位於 `$PATH`（`curl -fsSL https://agenvoy.com/scripts/install.sh | bash`）

Server 使用 [MCP 協定版本 `2024-11-05`](https://spec.modelcontextprotocol.io/specification/2024-11-05/)，支援 `tools/list`（含 `listChanged` 通知）與 `tools/call`。無需驗證 — server 以目前使用者身分在本地端執行。

各 MCP client 常見的配置模式：

```json
{
  "<servers_key>": {
    "agenvoy": {
      "command": "agen"
    }
  }
}
```

其中 `<servers_key>` 依 client 而異（`mcpServers`、`mcp_servers`、`mcp` 等）。部分 client 需要明確的 `"type": "stdio"` 或 `"type": "local"` 欄位。請查閱你的 client 文件。

## 公開的工具

| 工具 | 用途 |
|---|---|
| `script_*` / `api_*` / `ext_*` | 使用者建立的工具與擴充工具（自磁碟自動探索） |
| `write_tool` | 將 tool.json 或 script.py 寫入 script 工具目錄 |
| `test_tool` | 以範例輸入在 sandbox 中執行 script 工具 |
| `patch_tool` | 在工具檔案內做字串取代修正 |
| `remove_tool` | 將 script 工具移至垃圾桶 |
| `list_tools` | 列出 server 公開的所有工具 |
| `script_tool_generate_guide` | 回傳 Script Tool Contract（命名、範本、執行流程、檢查清單） |
| `api_public_api_list` | 依類別瀏覽免費公開 API 以供建立工具 |

工具 CRUD（`write_tool`、`test_tool`、`patch_tool`、`remove_tool`）與 Agenvoy 內部 runtime 共用 — 相同 handler、相同 schema，透過 `toolRegister` 橋接。無重複實作。

## Hot reload

Server 透過 `fsnotify` 監看工具目錄。當工具被建立、修改或刪除時，server 會自動重新掃描並送出 `notifications/tools/list_changed` — client 無需重連即可刷新工具清單。

***

> [!NOTE]
> 本文件由 Claude 讀取完整原始碼後自動生成。
