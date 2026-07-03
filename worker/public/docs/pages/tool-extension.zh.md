# Tool Extension

Agenvoy 支援四種在 built-in 集合之外新增 tool 的方式：從 capability gap 自動生成、script tool、API tool 與 MCP tool。

## 自動生成（Capability Gap）

當 user request 需要即時外部資料（天氣、匯率、股票、geocoding、翻譯等）且無現有 tool 涵蓋時，agent 當場建立該 tool，隨即執行以回答。無需寫程式。

system prompt 的 `Capability Gap` 段落驅動此序列：

| 步驟 | 動作 |
|---|---|
| 1. 找到合適的 API | `api_public_api_list(type=category)` 挑選相關類別，選出最佳候選（偏好 no-auth + HTTPS），再 `fetch_page` 文件 |
| 2. 建立 script tool | `mkdir` tool 目錄，再 `write_file` 一個 `tool.json`（name、description、schema）+ `script.py`（stdin JSON 轉 HTTP call 轉 stdout JSON） |
| 3. 執行並回答 | 將 user query pipe 進新 script；若失敗則修正並重試（最多 3 次） |

建立後，tool 持久化於 `~/.config/agenvoy/tools/script/<name>/`，並在所有未來 session 中可用。需認證的 API 透過生成 script 中的 `store_secret` + keychain 整合處理。

關鍵限制：

- agent 絕不可用裸 `send_http_request` 或 inline `python3 -c` 回答；必須將可重用 script 寫入磁碟
- `fetch_page` 僅允許用於讀取 API 文件，不用於抓取回答資料
- 生成的 `tool.json` 使用 `"always_allow": true`，使該 tool 在後續呼叫時免確認執行

## Script tool（`script_*`）

在 `extensions/scripts/<name>/` 底下放入 Python / Node.js / shell script，連同一個 `tool.json` descriptor。Agenvoy 於啟動時自動註冊為 `script_<name>`。

```
extensions/scripts/my-tool/
├── tool.json     # name, description, parameter schema, command
└── run.py        # actual script
```

## API tool（`api_*`）

在 `extensions/apis/<name>.json` 底下放入描述 REST endpoint 的 JSON 檔。它自動註冊為 `api_<name>`。每個 `api_<name>` 有自己 per-name 的 1 秒 rate limiter（`reserveAPISlot`）。

**Confirm gate** --- `api_*` tool 非 prefix-exempt 於確認。使用者可能定義破壞性 endpoint（DELETE / POST 寫入），因此 `agen cli` 對每次呼叫確認。批次自動核准請用 `agen run`。

## MCP tool（`mcp__*`）

由 MCP server 曝露的 tool 自動註冊為 `mcp__<server>__<tool>`。MCP tool output 每次呼叫上限 **1 MiB**，以將 tool result 保持在 provider 限制內。
