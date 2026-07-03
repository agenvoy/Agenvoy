# KuraDB RAG

KuraDB 是 Agenvoy 的 in-process RAG（Retrieval-Augmented Generation）provider。它以 daemon 管理的子行程執行，並向 agent 暴露兩個搜尋工具（`list_rag`、`search_rag`）。

## 它是什麼

KuraDB（[pardnchiu/KuraDB](https://github.com/pardnchiu/KuraDB)）是自行開發的本地文件索引，具備下列能力：

- 將使用者檔案（notes、inbox、code 等）索引進多個具名資料庫
- 透過 `gse` 斷詞（支援中文）提供關鍵字搜尋
- 透過 OpenAI embeddings（`text-embedding-3-small`）提供語意搜尋
- 完全在使用者機器上執行 —— 無外部服務

Agenvoy 透過本地 HTTP API 與 KuraDB 溝通（啟動時將隨機 port 寫入 `~/.config/kuradb/endpoint`）。

## 生命週期

KuraDB 位於 `internal/runtime/kuradb/`：

| 檔案 | 職責 |
|---|---|
| `kuradb.go` | 公開介面：`BinaryPath`、`EndpointExists()`、`ReadEndpoint()`、`BinaryInstalled()`、`HasOpenAIKey()`、`SetOpenAIKey()` |
| `run.go` | `RunChild(ctx)` —— `exec.Cmd` 啟動 + `StdoutPipe`/`StderrPipe` → slog；5 秒 crash backoff；health check goroutine 每分鐘輪詢 `<endpoint>/api/health`（5s timeout），連續 3 次失敗 → 自動停用 |

### Daemon 編排（`cmd/app/cmdDeamon.go::reloadKuradb`）

Daemon 透過 fsnotify 監看 `~/.config/agenvoy/config.json` 來控制 KuraDB：

1. 當 config 變更且 `kuradb_enabled=true` 時，spawn **之前**先過三道 gate：
   - `kuradb.BinaryInstalled()` —— `/usr/local/bin/kura` 必須存在
   - `kuradb.HasOpenAIKey()` —— `OPENAI_API_KEY` 必須在 keychain 中（`agenvoy` service）
2. 任一 gate 失敗 → **silent return**（不 log、不自動停用、不寫 config）
3. 通過 → 透過 `RunChild` spawn 子行程；KuraDB 將 endpoint URL 寫入 `~/.config/kuradb/endpoint`
4. Healthcheck goroutine 啟動；連續 3 次失敗 → 寫入 `kuradb_enabled=false` + 移除 endpoint 檔 + **顯式**呼叫 `reloadKuradb()`（不走 fsnotify async，以避免 200ms race window）

### Crash 復原

`RunChild` 將子行程包在 5 秒 backoff 迴圈內。Stdout/stderr 透過 `bufio.Scanner` 導入 `slog`，讓 KuraDB 錯誤落到 `daemon.log` 而非被丟棄。

## Tool 註冊

兩個 RAG 工具位於 `internal/runtime/kuradb/tool/`，並在三個進入點（`cmd/app/{main,cmdDeamon,newTUI}.go`）透過顯式 `kuradbTool.Register()` 呼叫註冊（非 `init()` —— `init()` 早於 `filesystem.Init()` 觸發，gate check 會永遠失敗）。

| Tool | 說明 |
|---|---|
| `list_rag` | 列出可用的 KuraDB 資料庫（例如 `notes`、`inbox`、`code`） |
| `search_rag` | 以關鍵字（`mode=keyword`，`gse` 斷詞）或語意（`mode=semantic`，OpenAI embeddings）搜尋資料庫 |

Tool gate 為單一條件 `cfg.KuradbEnabled` —— 每個 handler 內的 `ReadEndpoint()` 呼叫是 endpoint 於 turn 中途消失時的第二道防線。

## 每 turn 動態排除

`exec.Execute()` 在 `NewExecutor` 之後檢查 `kuradb.EndpointExists()`。當為 false 時，兩個 RAG 工具會被 append 到 `data.ExcludeTools`，既有的 filter 機制便在該 turn 將它們從 `exec.Tools` 剝除。

結果：endpoint 下線時 LLM **完全看不到** `list_rag` / `search_rag` 工具 —— 連 stub 名稱也看不到。system prompt 中「當 `list_rag` / `search_rag` 工具存在時」的條件式指引便自然失效。

**為何重要：** 若無動態排除，LLM 會在啟動 race（KuraDB 子行程 spawn 之前）看到 RAG tool stub、去呼叫它們、得到錯誤 —— 同時困惑 LLM 與使用者。

## `/feature kuradb` TUI wizard

啟用 / 停用僅透過 TUI 暴露（設計上無 CLI 子命令 —— install.sh + sudo 提示需要真實 TTY）：

```
/feature kuradb   → popup: enable | disable
```

### 啟用流程

1. Wizard 檢查 `HasOpenAIKey()`；若缺失，開啟 `popupText` 收集 key → `keychain.Set("OPENAI_API_KEY", value)`（service：`agenvoy`）
2. `tea.ExecProcess` 執行安裝腳本：
   ```
   curl -fsSL https://agenvoy.com/scripts/kuradb/install.sh | bash
   ```
   TTY 交給子行程，讓 `sudo` 提示與套件管理器輸出可正常運作
3. 驗證 `/usr/local/bin/kura` 的 `kura` binary；將 `kuradb_enabled=true` 寫入 config.json
4. Daemon 透過 fsnotify 接手 → `reloadKuradb()` spawn 子行程 → endpoint 檔出現 → 工具變為可呼叫

### 停用流程

1. `tea.ExecProcess` 執行 `sudo rm /usr/local/bin/kura`
2. 將 `kuradb_enabled=false` 寫入 config.json
3. Daemon `reloadKuradb()` 通知執行中的子行程關閉

## RAG-first prompting

當 `list_rag` / `search_rag` 工具載入時，base system prompt 要求**任何資訊查詢的第一波 tool call** 必須是 `list_rag` + `search_rag` —— 外部 web/search 工具為**次要**（用於補足缺口），而非 fallback 或替代。

此規則於 `configs/prompts/system_prompt.md` 強制執行；當 KuraDB 關閉時自動失效（因 `list_rag` / `search_rag` 不會在 tool list 中）。

## 檔案與路徑

| 路徑 | 用途 |
|---|---|
| `/usr/local/bin/kura` | KuraDB binary（由 `install.sh` 安裝） |
| `~/.config/kuradb/endpoint` | 純文字 URL，由 KuraDB 於啟動時寫入，停用時移除 |
| `~/.config/kuradb/` | KuraDB 端 config / data 目錄（由 KuraDB 自行管理） |
| Keychain `agenvoy/OPENAI_API_KEY` | 與 Agenvoy 其他使用 OpenAI 的功能共用 |

***

> [!NOTE]
> 本文件由 Claude 讀取完整原始碼後自動生成。
