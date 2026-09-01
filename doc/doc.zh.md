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
go build -tags fts5 -ldflags "-X github.com/pardnchiu/agenvoy/internal/runtime.CurrentVersion=dev" -o agen ./cmd/app/
./agen
```

### 使用 Makefile

```bash
make build
agen
```

`make build` 會將二進位檔安裝至 `/usr/local/bin/agen`（因此需要 `sudo`），並以 `extensions/` 的內容重建 `~/.config/agenvoy/skills/.system` 與 `~/.config/agenvoy/tools/.system`。

| Target                                  | 作用                                                                          |
| --------------------------------------- | ----------------------------------------------------------------------------- |
| `make build`                            | 以目前 git tag 為版本建置、安裝到 `/usr/local/bin/agen`、更新內建 skill／工具 |
| `make app`                              | `stop` → `build` → 啟動 TUI                                                   |
| `make cli <input>` / `make run <input>` | 不安裝直接跑一次請求                                                          |
| `make stop`                             | 停止 daemon                                                                   |
| `make update`                           | 執行官方更新程式                                                              |
| `make test`                             | `go test -v -count=1 ./...`                                                   |

### 直接執行

```bash
go run ./cmd/app/
```

## 設定

Agenvoy 使用 `~/.config/agenvoy/` 保存執行期資料，並將憑證存放於作業系統 keychain；不要將 API key 或 token 寫入專案檔案或 Git。

### 常用憑證

| Keychain 項目                                        | 用途                                                  |
| ---------------------------------------------------- | ----------------------------------------------------- |
| `OPENAI_API_KEY`                                     | OpenAI 與 KuraDB                                      |
| `CLAUDE_API_KEY`、`GROK_API_KEY`、`DEEPSEEK_API_KEY` | 對應模型供應商                                        |
| `TELEGRAM_TOKEN`、`DISCORD_TOKEN`                    | 聊天機器人整合                                        |
| `GEMINI_API_KEY`                                     | 語音回覆與 `transcribe_media`；未設定時該工具不會註冊 |

### Runtime 設定

主要設定檔是 `~/.config/agenvoy/config.json`。`limits` 欄位由程式載入，缺漏欄位會自動補上內建預設值。

| 設定                                |    預設值 | 說明                          |
| ----------------------------------- | --------: | ----------------------------- |
| `limits.max_tool_iterations`        |     `128` | 單次 Agent 工作的工具迭代上限 |
| `limits.agent_send_timeout_seconds` |     `600` | 模型請求逾時秒數              |
| `limits.max_history_messages`       |      `24` | 保留的近期歷史訊息數          |
| `limits.max_history_bytes`          | `5242880` | 歷史訊息大小上限（位元組）    |

套件內建預設值（目前不會從 `config.json` 讀取）：

| 常數                    |       預設值 | 說明                                                      |
| ----------------------- | -----------: | --------------------------------------------------------- |
| `MaxSessionTasks`       | `NumCPU × 4` | 每個 session 的並行工作數上限；超出的任務排隊等待而非失敗 |
| `MaxSubagentTimeoutMin` |         `30` | Subagent 逾時分鐘數                                       |
| `MaxResumeWaitMin`      |         `60` | Pending resume 等待回答的分鐘數                           |

```json
{
  "limits": {
    "max_tool_iterations": 128,
    "agent_send_timeout_seconds": 600
  }
}
```

### TUI 執行模式

目前 runtime 內建 10 個模型供應商，另有 `compat` 項目可接本機或自訂的 OpenAI 相容端點（Ollama、LM Studio、自架 gateway）。

當輸入區為空時，按下 `Shift+F` 可切換 fast mode。啟用時，標題列會顯示 `[fast]`。Fast mode 只存在於目前行程，不會保存至 `config.json`；它會透過 `go-llm-router` v0.5.1 傳遞 `provider.ModeFast`，讓支援的 provider backend 要求更快速的服務層級。關閉 fast mode 時則使用預設模式。

### Agent 選擇與確認路由

請求符合 Skill 時，dispatcher 會收到該 Skill 的說明作為選擇提示，因此模型選擇會反映目前任務契約，而不只依賴使用者輸入文字。

互動請求也會攜帶來源前綴。CLI 確認僅由 TUI 接收，Web 請求由 Web 確認串流處理，Telegram 與 Discord 則由各自對應的頻道 listener 接收。非 TUI 確認會在五分鐘後逾時，避免某個頻道攔截或長期占用其他頻道的提示。

### 受限路徑與受限指令

`$HOME` 以外的路徑與白名單外的指令不會直接被拒絕：`boundary.Resolve` 與 `tools.RestrictedCommands` 會收集它們並發出確認，且該確認同時要求作業系統密碼（TUI 內的 `sudo -v`）。核准只綁定該 session 與該路徑或該執行檔，時鐘一律沿用 sudo 自己的 timestamp，不另外維護 TTL；該 ticket 仍有效時，彈窗不會再出現密碼欄。

自動化之前要知道的事項：

- 一般工具確認會回到原始 CLI、Web、Telegram 或 Discord 頻道；TUI 只監聽 CLI 來源的請求。
- 需要作業系統驗證的受限操作，仍只會在驗證可用時執行。頻道即使核准提示，未完成驗證的受限呼叫仍會跳過，不會提權。
- 讀取不受路徑限制。沙箱約束的是寫入而非讀取，因此指令只有在作業系統本身拒絕時才會在某個路徑上失敗。

`$HOME` 一律可寫，不需要任何設定。指令要寫到 `$HOME` 以外時，agent 會在**那一次呼叫**自己附上 `write_paths`，你會看到確認框並輸入系統密碼，核准後那幾個路徑才額外綁進沙箱。

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
      "headers": { "Authorization": "Bearer ${MCP_TOKEN}" }
    }
  }
}
```

Agenvoy 本身也是 MCP server，走 stdio：`agen` 的 stdin 不是 TTY（有管線輸入）時就啟動 MCP server 而非 TUI。要掛進其他 agent 請自行寫入該 agent 的設定檔——Claude Code 用 `~/.claude.json`、OpenCode 用 `~/.config/opencode/opencode.jsonc`、Codex 用 `~/.codex/config.toml`：

```json
{
  "mcpServers": {
    "agenvoy": { "command": "agen" }
  }
}
```

```toml
[mcp_servers.agenvoy]
command = "agen"
```

### Session 分類與監控

TUI 的 session 選擇器會依 ID 前綴分類：`cli-` 代表本機 CLI、`tg-` 代表 Telegram、`dc-` 代表 Discord、`chat-` 代表 Web／API，`temp-` 代表短期工作。偵測到至少兩個群組時，選擇器會顯示 `all` 與各前綴分頁，並將目前 session 排在最前。Daemon 會以 `fsnotify` 監看新建立的 session 目錄，將 session ID 與設定名稱寫入 daemon log。

Session persona 現存於 history SQLite 資料庫。`self_id` 會正規化為小寫，只接受最多 32 個 ASCII 字母、數字、`_` 或 `-`，非空值必須唯一。Daemon 啟動時會把舊版每個 session 的 `bot.json`、bot markdown、`config.json` 與 `status.json` 遷移至 SQLite／state table。

Daemon 另有背景 Runtime 監控器，每 30 秒檢查 CPU、Go process 記憶體，以及到 `1.1.1.1:443` 的 TCP 連線；CPU 過高、記憶體過高、網路中斷與恢復都會寫入 daemon log，CPU 異常時也會盡可能列出前三名程序。

## 使用方式

### 啟動 TUI

```bash
agen
```

直接輸入文字即在當前 session 執行；其餘皆為斜線指令，只打 `/` 會開啟選單，其中也包含已安裝的 skill 與排程項目。

| 指令                            | 用途                                                                                  |
| ------------------------------- | ------------------------------------------------------------------------------------- |
| `/model`                        | 新增／移除 provider，挑選 session／dispatcher／summary 模型，設定圖片生成來源         |
| `/mcp`                          | 列出 MCP server；新增、登入、重連、查看工具、設定單一工具權限、移除                   |
| `/switch` `/new`                | 切換 session 或建立新的（名稱會檢查重複）                                             |
| `/bot`                          | 重新命名當前 session 或編輯 persona                                                   |
| `/memory`                       | 對當前 session 執行 `compact` / `reset` / `summary`                                   |
| `/dangerous`                    | 刪除 session，或編輯 skill／指令白名單                                                |
| `/discord` `/telegram` `/voice` | 啟用或停用各通道；token 會先驗證再存入                                                |
| `/startup`                      | 啟用或停用登入時自動啟動 daemon（macOS 走 launchd agent，Linux 走 systemd user unit） |
| `/admin-channel`                | 選擇由哪個已授權對話接收新對話驗證碼                                                  |
| `/kuradb`                       | 安裝、更新或重連 KuraDB（以 MCP server 形式）                                         |
| `/cron` `/task`                 | 新增、編輯或移除週期性與一次性排程                                                    |
| `/pending`                      | 列出並恢復中斷的任務（`ask_user`、錯誤復原）                                          |
| `/resume` `/log` `/usage`       | 重載可見對話、以 `$PAGER` 開啟 `action.log`、查看各模型 token 用量                    |
| `/key`                          | 更換已儲存的憑證                                                                      |
| `/update`                       | 抓取最新 release、重建、離開                                                          |
| `/clear` `/exit`                | 清除可見對話，或離開 TUI（daemon 繼續執行）                                           |
| `/<skill>` `/sched-<name>`      | 直接執行已安裝的 skill 或排程項目                                                     |

輸入區為空時可用的快捷鍵：

| 按鍵                  | 動作                                                        |
| --------------------- | ----------------------------------------------------------- |
| `Shift+W` / `Shift+S` | 反向／正向切換 session 模型                                 |
| `Shift+A` / `Shift+D` | 切換 reasoning 等級                                         |
| `Shift+F`             | 切換 fast mode                                              |
| `Shift+T`             | 切換指令模式——輸入內容以 shell 指令在當前目錄執行，不經沙箱 |
| `Shift+U`             | 查看 provider 額度與餘額                                    |
| `Shift+M`             | 列出已註冊模型                                              |

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

### Web 與檔案回應顯示

後端會在 result、SSE、pending 與 multilog handler 間保留 `[SEND_FILE:…]` 標記，讓頻道 consumer 仍可取得檔案傳遞 metadata。Web dashboard 只在顯示訊息文字時移除標記，因此使用者看到的是回應內容，不會看到內部傳輸標記。

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

| 指令     | 語法                   | 說明                               |
| -------- | ---------------------- | ---------------------------------- |
| TUI      | `agen`                 | 開啟或連接本機 daemon 的互動式 TUI |
| 停止     | `agen stop`            | 停止 daemon                        |
| 更新     | `agen update`          | 執行官方更新腳本                   |
| Daemon   | `agen --daemon`        | 直接啟動 daemon                    |
| MCP      | 非 TTY stdin 的 `agen` | 從 stdin 提供 MCP JSON-RPC         |

## HTTP API 參考

Daemon 只綁定 `127.0.0.1`。標示 **local** 的 endpoint 另外要求請求來源必須是 `127.0.0.1`／`::1`（`localhostOnly()` 守衛）——這些會動到 credential、設定檔或行程生命週期，設計上是給同機器的 dashboard 用，不是給遠端 client 呼叫。

**Agent 執行**

| Method | Path                   | 說明                                        |
| ------ | ---------------------- | ------------------------------------------- |
| `POST` | `/v1/send`             | 執行 Agent                                  |
| `POST` | `/v1/chat/completions` | OpenAI 相容且 stateless 的 chat completions |
| `GET`  | `/v1/info/version`     | 編譯時寫入的版本（`{version, dev}`）；未帶 tag 的建置 `dev` 為 true |
| `GET`  | `/v1/log`              | SSE：不帶參數時只送 daemon `slog`（`EventDaemonLog`，`source` 為層級），與 TUI 標題列同一份來源；內含新對話驗證碼，故 daemon frame 僅對 loopback 來源附加。`?sessions=a,b` 於同一條連線加上該些 session 的事件，`replay=0` 略過回放，`daemon=0` 去掉 daemon frame。遠端來源必須帶 `sessions` |
| `GET`  | `/v1/mcp/tools`        | 列出已連線 MCP server 註冊的工具（`mcp__*`）  |

**模型**

| Method          | Path                            | 說明                                                 |
| --------------- | ------------------------------- | ---------------------------------------------------- |
| `GET`           | `/v1/models`                    | 列出已註冊模型（OpenAI `{data:[…]}` 格式,含 `auto`） |
| `GET`           | `/v1/models/*id`                | 讀取單一已註冊模型                                   |
| `POST` `DELETE` | `/v1/models` `/v1/models/*name` | **local** — 新增／移除模型                           |
| `GET` `POST`    | `/v1/model`                     | **local** — 模型路由：`dispatcher`／`summary`／`image`，讀取時另附 `image_options`。三個欄位放的東西不同類：`dispatcher` 與 `summary` 填已註冊的模型名（`prefix@model`），`image` 填的是 provider endpoint（`openai`／`codex`／`grok`／`grok-oauth`／`gemini`）——因為各 provider 的圖片模型寫死在 `go-llm-router` 內，選的是來源不是模型；`image_options` 只列出當前有憑證的 provider。`POST` 為部分更新：未帶（或 `null`）的欄位不動，`""` 清除，`image` 另接受 `off` 作為 `""` 的別名。模型未註冊、provider 不存在、或該 provider 沒有憑證一律拒絕且整批不寫。兩個動詞回傳同一種物件 |

**Session**

| Method                | Path                                           | 說明                                                                                                                                                                                  |
| --------------------- | ---------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `GET`                 | `/v1/sessions`                                 | 列出 session 與狀態                                                                                                                                                                   |
| `POST`                | `/v1/session`                                  | **local** — 建立 session，`{prefix}` 預設 `cli-`                                                                                                                              |
| `GET` `POST` `DELETE` | `/v1/session/:id`                              | **local** — 單一 session 的完整狀態：`id`／`self_id`／`name`／`rule`／`state`／`model`／`reasoning`／`levels`／`count`。`POST` 為部分更新，`self_id`／`name`／`rule`／`model`／`reasoning` 皆選填，未帶（或 `null`）的欄位不動；`model: ""` 重設為 `auto`，`reasoning` 須為 `levels` 之一。`GET` 與 `POST` 回傳同一種物件，`self_id` 重複回 409。`DELETE` 移除 session 目錄、歷史、狀態與向量。`GET` 另接受 `?chat=1` 附上原始 action log（放在 `chat`）與 `?usage=1` 附上 24h/7d/28d 各模型 token 用量（放在 `usage`，與 TUI `/usage` 畫面同一套聚合邏輯）；兩者預設關閉，因為 log 可能很大 |
| `POST`                | `/v1/session/:id/event`                        | **local** — 對某 session 的事件串流手動發布事件                                                                                                                                       |
| `GET`                 | `/v1/session/:id/task`                      | 列出可恢復的待完成（`ask_user`／confirm）工作；仍在執行中的不列入——執行期間每 3 秒刷新 ToriiDB 的 `action:<session_id>:<task_hash>`（TTL 5 秒），視窗關閉或程序被砍的任務 5 秒內會重新出現 |
| `GET`                 | `/v1/session/:id/task/:task_hash/questions` | 取得待完成工作的問題內容                                                                                                                                                              |
| `POST`                | `/v1/session/:id/task/:task_hash/resume`    | 回答待完成工作並恢復執行                                                                                                                                                              |
| `DELETE`              | `/v1/session/:id/task/:task_hash`           | 直接捨棄待完成工作，不回答                                                                                                                                                            |
| `POST`                | `/v1/session/:id/cancel/:once_id`              | 取消單一執行中的任務；該 id 不在本行程執行中時回 404                                                                                                                                |
| `POST`                | `/v1/session/:id/confirm/:once_id`          | 回覆等待中的工具確認:`{approve, remember?, allow_turn?, abort?, reason?}`。受限路徑與白名單外指令無法由此核准——它們需要只有 TUI 收得到的系統密碼驗證,沒帶驗證的核准會被退回為 skipped |
| `POST`                | `/v1/session/:id/memory`                       | **local** — 對該 session 執行一項記憶操作，由 `action` 決定：`summary` 重建滾動摘要並回 `count`；`compact` 丟掉較舊的訊息並回 `removed`；`reset` 清空對話並回 `removed`，且必須帶 `mode`——`summary` 保留滾動摘要，`all` 連摘要一起清 |
| `GET`                 | `/v1/session/:id/task/history`                 | **local** — 該 session 已完成的任務清單（新到舊），每列 `{task_hash, end_at, objective, model, reasoning}`；`?keyword=` 對 objective 與紀錄內容做過濾                                 |
| `GET`                 | `/v1/session/:id/task/:task_hash/history`      | **local** — 單一已完成任務的完整 action 紀錄，以 JSON 字串放在 `content`；該 hash 沒有紀錄時回 404                                                                                     |

**Channel**

| Method | Path                                         | 說明                                                                                                                                                                                                                                                    |
| ------ | -------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `GET`  | `/v1/channel`                                | **local** — 所有 channel 讀取合在同一個物件：`telegram` 與 `discord` 各帶 `{enabled, username, has_token}`，`admin` 帶 `{channel, authorized, chats:[{value,type,id,name}]}`。`chats` 取自 `.telegram` / `.discord` 授權檔（先 tg 後 dc），`value` 可直接回送 `POST`；`authorized` 標示現行轉發目標是否仍在名單內（手打的 ID 會是 `false`） |
| `POST` | `/v1/channel/telegram` `/v1/channel/discord` | **local** — `{action:"enable"\|"disable", token?}`。enable 只存 token 並切換設定 flag,刻意不做 TUI 那套 `GetMe` 驗證——daemon 既有的設定檔監看機制會自動重連 bot 並填回使用者名稱                                                                        |
| `GET` | `/v1/channel/:channel/chats` | **local** — `telegram` / `discord` 已完成驗證的對話,來源是 `.telegram` / `.discord` 授權檔。只有 bot 執行中才有意義,建議在 `status` 回報 `enabled` 後才取用 |
| `DELETE` | `/v1/channel/:channel/chat` | **local** — `{id}`。從該授權檔移除一筆對話,該對話需重新驗證才能再與 bot 對話。id 不在名單上回 404 |
| `POST` | `/v1/channel/admin`                          | **local** — `{value:"tg@<chatID>"\|"dc@<channelID>"\|""}`。設定新對話驗證碼的轉發目標,空字串清除;`value` 必填(省略回 400,避免誤送空 body 靜默清除)。只驗格式,不檢查該 ID 是否已在授權名單——未授權時 `NotifyAdminCode` 會 log warning 並讓驗證碼留在日誌 |

**檔案與憑證**

| Method         | Path              | 說明                                                                                   |
| -------------- | ----------------- | -------------------------------------------------------------------------------------- |
| `GET` `PUT`    | `/v1/file`        | **local** — 讀取／寫入檔案                                                             |
| `GET`          | `/v1/file/open`   | **local** — 以系統預設程式開啟檔案／URL                                                |
| `GET`          | `/v1/file/locate` | **local** — 依純檔名搜尋候選路徑（`name`,可選 `dir=1`、`child`、`size`、`mtime` 過濾） |
| `GET`          | `/v1/workdir`     | **local** — 解析並驗證工作目錄（`?path=`）,回傳絕對路徑                                |
| `GET` `DELETE` | `/v1/key`         | **local** — 查詢／刪除 keychain 中單一憑證                                             |
| `GET` `POST`   | `/v1/keys`        | **local** — 列出／設定憑證                                                             |

**Provider**

| Method | Path                            | 說明                                   |
| ------ | ------------------------------- | -------------------------------------- |
| `GET`  | `/v1/providers`                 | **local** — 列出 provider 及其可用操作 |
| `GET` | `/v1/providers/usage` | **local** — `codex`、`grok-oauth`、`copilot` 的剩餘額度（`kind:"percent"`）與 `openrouter`、`deepseek` 的剩餘餘額（`kind:"balance"`）,平行取得,上限 15 秒。成功的結果在 ToriiDB 快取 3 分鐘並帶 `cached:true`;`?refresh=1` 清除快取重讀,存入 API key 或完成 OAuth 也會自動清掉該 provider 的快取。沒有憑證的 provider 回 `error` 而非 `value`,且不進快取 |
| `POST` | `/v1/provider/:provider/key`    | **local** — 設定 API key               |
| `GET`  | `/v1/provider/:provider/oauth`  | **local** — SSE device-code OAuth 流程 |
| `DELETE` | `/v1/provider/:provider/oauth` | **local** — 清除已儲存的 provider 登入（`codex`、`copilot`、`grok-oauth`）。token 的 keychain 鍵名由 OAuth 套件自己持有（`CODEX_OAUTH_TOKEN` 與各自的舊名）,因此改走它們的 `ClearToken`,而非 `DELETE /v1/key` |
| `GET`  | `/v1/provider/:provider/models` | **local** — 列出該 provider 可用模型   |

**MCP**

| Method       | Path                     | 說明                                                                                                                                                                                                                        |
| ------------ | ------------------------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `GET` `POST` | `/v1/mcp`                | **local** — 列出／新增 MCP server。`GET` 另回 `oauth: {name: bool}`,標示各 HTTP server 是否已持有 token                                                                                                                     |
| `POST`       | `/v1/mcp/remove`         | **local** — 移除 MCP server                                                                                                                                                                                                 |
| `GET`        | `/v1/mcp/status`         | **local** — 各 server 連線狀態                                                                                                                                                                                              |
| `POST`       | `/v1/mcp/reconnect`      | **local** — 重連全部 MCP client 並重新註冊工具                                                                                                                                                                              |
| `GET`        | `/v1/mcp/oauth?name=X`   | **local** — 單一 HTTP MCP server 的 SSE OAuth 登入,與 `/v1/provider/:provider/oauth` 同形狀:先送 `{"url":…}` 供瀏覽器開啟,結束送 `{"done":true,"ok":…}`(登入後重連失敗時附 `reconnect_error`)。10 分鐘逾時,客戶端斷線即中止 |
| `POST`       | `/v1/mcp/oauth/callback` | **local** — `{name, url}`。瀏覽器連不到 daemon 的 `localhost:17988` loopback listener 時,把 redirect URL 貼回來,code 由 query 取出。該 server 沒有等待中的登入回 400                                                        |
| `POST`       | `/v1/mcp/oauth/client`   | **local** — `{name, client_id, client_secret?, redirect_uri?}`。給拒絕動態註冊的 server 用的預先註冊 client;`redirect_uri` 預設 `http://localhost:17988/callback`,須與 provider console 完全一致。寫入前先清掉既有 token    |
| `DELETE`     | `/v1/mcp/oauth`          | **local** — `{name}`。同時清除該 server 的 token 與 client 註冊                                                                                                                                                             |

**Rule、知識與 Skill**

| Method                  | Path                  | 說明                                                                                     |
| ----------------------- | --------------------- | ---------------------------------------------------------------------------------------- |
| `GET`                   | `/v1/rules`           | **local** — 列出 `prompts/` 底下的 session prompt rule（`.md`）                          |
| `GET`                   | `/v1/rule/*name`      | **local** — 讀取單一 rule                                                                |
| `POST` `PATCH` `DELETE` | `/v1/rule`            | **local** — 建立／更新（可帶 `rename`）／刪除 rule                                       |
| `GET`                   | `/v1/knowledges`      | **local** — 列出 operator 筆記（名稱、大小、`updated_at`）,資料存於 ToriiDB 而非檔案系統 |
| `GET`                   | `/v1/knowledge/*name` | **local** — 讀取單筆筆記                                                                 |
| `POST` `PATCH` `DELETE` | `/v1/knowledge`       | **local** — 建立／更新／刪除筆記,未給名稱時以首行為名                                    |
| `GET`                   | `/v1/skills`          | **local** — 列出已安裝的 skill                                                           |

**排程與自動化**

| Method         | Path                  | 說明                                             |
| -------------- | --------------------- | ------------------------------------------------ |
| `GET`          | `/v1/schedule`        | **local** — 以單一 `schedules` 陣列列出 cron 與單次任務，每筆帶 `type=cron\|task`；`?type=` 可只取其一 |
| `GET`          | `/v1/schedule/*skill` | **local** — 讀取 scheduler skill，拆成 `name`／`description`／`body`（frontmatter 已解析） |
| `POST` `PATCH` | `/v1/schedule`        | **local** — 由 `name`／`description`／`content` 建立／更新 scheduler skill（frontmatter 由後端組出）並整組重綁到 `type=cron\|task`；切換 type 時同步刪除該 skill 在另一邊的排程 |
| `DELETE`       | `/v1/schedule`        | **local** — 刪除該 skill 的排程（帶 `type` 只刪一邊，不帶則兩邊都刪）；刪除後若無其他綁定則將 skill 移入 .Trash |
| `POST`         | `/v1/schedule/run`    | **local** — 立即觸發排程（`202 Accepted`） |

**白名單**

| Method       | Path                  | 說明                                                                                                                                                                                                                                         |
| ------------ | --------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `GET` `POST` | `/v1/allowlist`       | **local** — 兩份白名單合在同一個物件，鍵為 `skill` 與 `tool`。`GET` 以 `?scope=global\|project`（`project` 需另帶 `?work_dir=`）決定 skill 區塊，`?prefix=` 縮小 tool 區塊。`POST` 收 `{skill: {name, scope?, work_dir?}}` 切換單一 skill，與／或 `{tool: {prefix, entries}}` 只替換該前綴底下的免確認項目（與 TUI `/mcp` → permission 同一支），其餘規則不受影響；每個 entry 必須以 `prefix` 開頭，出現 `prefix*` 時收斂成單一項。未帶的區塊不動 |

**設定**

| Method       | Path                  | 說明                                                                                                                                                              |
| ------------ | --------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `GET` `POST` | `/v1/config/startup`  | **local** — 讀取／設定登入時啟動。`POST` `{enable}` 寫入或刪除 launchd agent（macOS）／systemd user unit（Linux）；不會啟動或停止當前 daemon，下次登入才生效 |

**查閱**

| Method | Path              | 說明                                                                     |
| ------ | ----------------- | ------------------------------------------------------------------------ |
| `GET` `POST` `DELETE` | `/v1/toriidb/:db/*key`、`/v1/toriidb` | **local** — ToriiDB 的唯一入口。`GET` 帶純 key 回單筆，帶 `*` 樣式回全部命中，並可用 `?contains=<substr>`（比對 value，不分大小寫）與 `?after=<unix 秒>`（entry 異動時間）在伺服器端縮小範圍；`?keys=1` 只回名稱、`?search=<text>&limit=` 走向量搜尋；`POST` `{db,key,value,expire_at,vector}` 以絕對時間戳寫入，或 `{db,keys,ttl}` 以相對秒數重設既有 key 的到期；`DELETE` `{db,keys}` 刪除。daemon 以外的行程一律只能經由此端點碰 ToriiDB，所以 tool cache、對話向量、knowledge、error memory 與 pending 存活在 TUI 與 daemon 之間是共用的 |
| `GET`  | `/v1/torii/error` | **local** — 查閱工具錯誤記憶；`tool`／`keyword` 皆未帶時回傳無過濾全表掃 |

## 工具參考

註冊表內建 26 個工具，另有 3 個在前置條件成立時才註冊。涵蓋多種相關動作的工具以 `mode` 參數區分，而不是拆成多個名稱。

| 分類       | 工具                              | 用途                                                                                   |
| ---------- | --------------------------------- | -------------------------------------------------------------------------------------- |
| 工具系統   | `find_tools`                      | 發現既有工具並載入其 schema（`mode=search\|list`）                                     |
|            | `edit_tool`                       | 建立、修正或丟棄工具定義（`mode=write\|patch\|remove`）                                |
|            | `test_tool`                       | 上線前在沙箱內執行 script 工具                                                         |
| Skill      | `run_skill`                       | 載入具名 skill 的參考素材                                                              |
|            | `edit_skill`                      | 編寫 skills 目錄底下的檔案（`mode=write\|patch\|remove`）                              |
| 排程       | `schedules`                       | 查詢、改期或取消定時與週期任務（`mode=list\|patch\|remove\|write`）                    |
| 檔案       | `find_files`                      | 以目錄、檔名樣式或內容定位（`mode=list\|glob\|search`）                                |
|            | `read_files`                      | 批次讀取文字、PDF、DOCX、PPTX、CSV 與圖片                                              |
|            | `edit_file`                       | 建立、修改、移置或還原檔案（`mode=write\|patch\|remove\|restore`）                     |
|            | `file_history`                    | 工具改過的每個檔案的版本紀錄（`mode=list\|read`）                                      |
| 執行環境   | `run_command`                     | 在工作目錄以沙箱約束執行二進位                                                         |
|            | `open_file`                       | 以系統預設應用開啟檔案                                                                 |
|            | `download_file`                   | 下載二進位資產至磁碟                                                                   |
|            | `pkg_manage`                      | 驅動 Linux 套件管理器（install／remove／update／upgrade／search／info）；僅 Linux，全通道 |
| Agent 協調 | `subagents`                       | 將子任務委派到獨立 session（`mode=invoke\|list`）                                      |
|            | `write_todo`                      | 使用者即時看得到的任務清單                                                             |
|            | `ask_user`                        | 暫停提問，回答後自動續跑                                                               |
| 網路       | `search_web`                      | DuckDuckGo 結果與 Google News 標題一次取得                                             |
|            | `fetch_page`                      | 取得完整頁面內容（markdown／html／json）                                               |
|            | `http_request`                    | 原始 HTTP 呼叫，含 multipart 上傳                                                      |
| 狀態       | `chat_history`                    | 本 session 的執行紀錄與對話（`mode=list\|read\|search`）                               |
|            | `error_history`                   | 跨 session 保留的工具失敗紀錄（`mode=search\|read\|write`）                            |
|            | `find_knowledge`                  | 操作者自己寫的筆記，存於 ToriiDB（`mode=search\|list\|read`）；search 與 list 只回名稱 |
|            | `reasoning_guide`                 | 依 `topic` 取得完整推理規則                                                            |
| 基礎支援   | `calculate`                       | 算術、單位與匯率換算                                                                   |
|            | `store_secret`                    | 遮蔽輸入並存入 keychain                                                                |
| 條件註冊   | `transcribe_media`                | 音訊／影片轉文字——需 `GEMINI_API_KEY`                                                  |
| 條件註冊   | `generate_image`                  | 文字生成圖片並存檔——image generator 為 off 時排除                                      |
|            | `list_chatbot`、`send_to_chatbot` | 跨頻道推送——需啟用 Telegram 或 Discord                                                 |

13 個工具會帶完整 schema 送出——`ask_user`、`calculate`、`edit_file`、`fetch_page`、`find_files`、`find_knowledge`、`find_tools`、`read_files`、`reasoning_guide`、`run_command`、`run_skill`、`search_web`、`write_todo`；其餘工具初始只送名稱與描述，參數在首次使用時經 `find_tools(mode=search)` 載入，讓初始工具 payload 遠低於完整註冊表。

## 架構

請參閱完整的 [Architecture](./architecture.md) 與繁體中文 [架構](./architecture.zh.md)。

## License

本專案以 [Apache License 2.0](../LICENSE) 授權。

---

©️ 2026 [邱敬幃 Pardn Chiu](https://www.linkedin.com/in/pardnchiu)
