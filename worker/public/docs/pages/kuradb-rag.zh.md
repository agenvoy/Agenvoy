# KuraDB RAG

KuraDB 是一個獨立的 RAG（Retrieval-Augmented Generation）daemon，Agenvoy 透過本地 HTTP 與它溝通。它是獨立的長駐行程（`kura`），由使用者自行啟動/停止 —— Agenvoy 不 spawn 也不擁有它的生命週期。Agenvoy 向 agent 暴露兩個呼叫 KuraDB API 的工具（`list_rag`、`search_rag`）。

## 它是什麼

KuraDB（[pardnchiu/KuraDB](https://github.com/pardnchiu/KuraDB)）是自行開發的本地文件索引，具備下列能力：

- 將使用者檔案（notes、inbox、code 等）索引進多個具名資料庫
- 透過 `gse` 斷詞（支援中文）提供關鍵字搜尋
- 透過 OpenAI embeddings（`text-embedding-3-small`）提供語意搜尋
- 完全在使用者機器上執行，`kura` 自行 daemonize —— 無外部服務

Agenvoy 透過本地 HTTP API 與 KuraDB 溝通。endpoint URL 由 `kura` 於啟動時寫入 `~/.config/kuradb/endpoint`；PID 與啟動時間寫入 `~/.config/kuradb/runtime.uid`。

## Agenvoy 端介面（`internal/runtime/kuradb/kuradb.go`）

Agenvoy 不管理 KuraDB 行程，只觀察它：

| 函式 | 用途 |
|---|---|
| `IsInstalled()` | `/usr/local/bin/kura` 存在且可執行 |
| `IsRunning()` | 讀 `~/.config/kuradb/runtime.uid`，檢查記錄的 PID 是否仍存活（`os.FindProcess` + signal 0） |
| `Version()` | 讀 Go 在建置時戳進 binary 的 module 版本（`debug/buildinfo`）——即 release tag，因為 `install.sh` 是從乾淨的 tag checkout 建置 |
| `Health(ctx, onFail)` | 每分鐘 tick 一次；一次 strike 需**同時**滿足 `IsRunning()` 與存活的 `GET <endpoint>/api/health` —— 連續 3 次 strike 呼叫 `onFail` |
| `SyncOpenAIKey(value)` | 把 OpenAI key 寫進**另一個**獨立的 OS keychain 條目、service 為 `kuradb`（`security`/`secret-tool`，非 go-pkg keychain）—— `kura` 是獨立行程，讀不到 Agenvoy 的 keychain namespace |
| `Remove()` | 刪除 endpoint 檔（用於在判定 KuraDB 已下線時讓 tool call 快速失敗） |

### Health gating（`cmd/app/cmdDeamon.go::reloadKuradb`）

當 config 變更時（fsnotify 監看 `~/.config/agenvoy/config.json`），若 `kuradb_enabled=true` 且 `kuradb.IsInstalled()` 且 Agenvoy keychain 中有 `OPENAI_API_KEY`：把 key 同步進 KuraDB 的 keychain 條目，並啟動一個 `Health` goroutine。任一 gate 失敗都是 silent no-op —— 不 log、不自動停用、不寫 config。連續 3 次 health strike → `disableKuradb()`：寫入 `kuradb_enabled=false`、移除 endpoint 檔、再次呼叫 `reloadKuradb()`（顯式呼叫，非透過 fsnotify watcher，以避免 race window）。

## Tool 註冊

兩個 RAG 工具位於 `internal/runtime/kuradb/tool/`，僅在 daemon 啟動時註冊一次（`cmd/app/cmdDeamon.go::kuradbTool.Register()`，非 `init()` —— `init()` 早於 `filesystem.Init()` 觸發，gate check 會永遠失敗）。

| Tool | 說明 |
|---|---|
| `list_rag` | 列出可用的 KuraDB 資料庫（例如 `notes`、`inbox`、`code`） |
| `search_rag` | 透過 KuraDB 統一的 `/api/search` 搜尋資料庫 —— 預設關鍵字（`gse` 斷詞）與語意（OpenAI embeddings）同時執行；帶 `?target=keyword`/`?target=semantic` 可收斂成單一模式 |

註冊 gate 為單一條件 `cfg.KuradbEnabled`，僅在啟動時檢查一次 —— binary 變為可用後要重新啟用，需重啟 `agen`。`kuradbGet()`（`tool/register.go` 內共用的 HTTP helper）是每次呼叫的第二道防線：它每次呼叫都重新解析 endpoint 檔，KuraDB 未執行時回傳明確錯誤。

## 每 turn 動態排除

`exec.Execute()` 在建好 executor 之後檢查 `~/.config/kuradb/endpoint` 是否存在（`go_pkg_filesystem_reader.Exists`）。當不存在時，`list_rag`、`search_rag` 會被 append 到 `data.ExcludeTools`，既有的 filter 機制便在該 turn 將它們從工具清單剝除。

結果：endpoint 檔消失時 LLM **完全看不到** `list_rag` / `search_rag` —— 連 stub 名稱也看不到。

**為何重要：** 若無動態排除，LLM 會在 KuraDB 停止期間仍看到 RAG tool stub、去呼叫它們、得到錯誤 —— 同時困惑 LLM 與使用者。

## `/feature kuradb` TUI wizard

僅透過 TUI 暴露（設計上無 CLI 子命令 —— install.sh + sudo 提示需要真實 TTY）。選項清單反映當前狀態：

```
/feature kuradb
  未安裝 → enable
  已安裝 → update、disable，以及 start（若已停止）或 stop（若執行中）
```

popup 標題列顯示 `kura <version>  ● running (<endpoint>)` 或 `○ stopped`。

### 啟用 / 更新流程

1. 若 Agenvoy keychain 中尚無 `OPENAI_API_KEY`，開啟 `popupText` 收集 → `keychain.Set`（Agenvoy 自己的 keychain）+ `kuradb.SyncOpenAIKey`（KuraDB 獨立的 keychain 條目）
2. `tea.ExecProcess` 執行 `curl -fsSL https://kuradb.agenvoy.com/scripts/install.sh | bash`，TTY 交給子行程讓 `sudo` 提示與套件管理器輸出可正常運作，接著跑 `kura add agenvoy`
3. 驗證 `kura` binary 落在 `/usr/local/bin/kura`；將 `kuradb_enabled=true` 寫入 config.json

### 啟動 / 停止流程

`start` 執行裸的 `kura`（它會自行 fork 到背景並在就緒後返回）；`stop` 執行 `kura stop`（SIGTERM，逾寬限期後改 SIGKILL）。兩者都不動 `kuradb_enabled` —— 只影響已設定好的 daemon 是否在跑。

### 停用流程

`tea.ExecProcess` 執行 `sudo rm -f /usr/local/bin/kura`，接著將 `kuradb_enabled=false` 寫入 config.json。

## RAG + 即時網路併用

`configs/prompts/system_prompt.md` 要求任何非閒聊的資訊查詢都須同時以 `search_rag`（若可用）與即時網路查詢（`search_web` / `search_google_news`）為依據 —— RAG 是基礎 context，只要答案涉及真實世界實體或時效性，即時網路就是強制項。兩者互不作為對方的 fallback；僅閒聊或純本地專案操作可跳過兩者。

## 檔案與路徑

| 路徑 | 用途 |
|---|---|
| `/usr/local/bin/kura` | KuraDB binary（由 `install.sh` 安裝） |
| `~/.config/kuradb/endpoint` | 純文字 URL，由 KuraDB 於啟動時寫入，Agenvoy 於 health check 失敗後移除 |
| `~/.config/kuradb/runtime.uid` | JSON `{uid, pid, started_at}`，Agenvoy 的 `IsRunning()` 讀取此檔 |
| `~/.config/kuradb/` | KuraDB 端 config / data 目錄（由 KuraDB 自行管理） |
| Keychain `agenvoy/OPENAI_API_KEY` | Agenvoy 自己的副本，透過 `/feature kuradb` wizard 輸入 |
| Keychain `kuradb/OPENAI_API_KEY` | KuraDB 自己的副本，由 `SyncOpenAIKey` 保持同步，因 `kura` 是獨立行程 |

Agenvoy 自己的 updater（`static/scripts/update.sh`）在已安裝 `kura` 時，也會在完成前一併更新它。

***

> [!NOTE]
> 本文件由 Claude 讀取完整原始碼後自動生成。
