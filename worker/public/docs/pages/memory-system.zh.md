# Memory System

Agenvoy 的 memory 層針對 conversation memory 有三個層級，另加一個跨 session 的 error memory 層。

| 層級 | 後端 | 範圍 |
|---|---|---|
| 1. Context window（16 則訊息 + summary） | `history.json` + `summary.json` | session |
| 2. 語意搜尋（近期） | ToriiDB `DBSessionHist`（vector） | session |
| 3. 全文封存（全部歷史） | 透過 go-sqlkit 的 SQLite FTS5 | session |
| Error memory | ToriiDB `error_memory`（90 天 TTL） | 跨 session |

## 三層 conversation memory

### 1. Context window（`max_history_messages`，預設 16）

每個 session 完整保留最近 N 則訊息並餵入 LLM context window。更舊的內容仍留在 `history.json`，但不送給 LLM。

滾動式 summary（`summary.json`）將較舊的對話濃縮，並於每個 turn 開頭注入 system prompt，讓較舊 context 得以在 N 則訊息窗口之外存續。

**增量游標：** `summary.meta.json`（per-session）持有 `last_message_time`（格式 `YYYY-MM-DD HH:MM:SS`，由訊息內容中的 timestamp 擷取）。每次 `summary.Generate` 呼叫時：

1. `filterAfterTime(histories, cursor)` 只保留 `t > cursor` 的訊息
2. 每個 chunk 執行**一次** `generatePass` LLM call（system prompt 已含 `{{.Summary}}`=舊 summary，因此合併在生成期間完成 —— 無獨立 `mergePass`，避免 2 倍成本）
3. 成功時，游標推進至該 chunk 的最大 timestamp + `SaveSummary` 觸發 mtime gate
4. `generatePass` 失敗 → `return`（不對後續 chunk 計費；下一次 cron tick 重試）

### 2. 語意搜尋 —— ToriiDB（近期對話）

`search_chat_history` 工具搭配 `mode=semantic` 透過 ToriiDB `db.VSearch` 執行 vector 相似度搜尋。每次命中觸發 context window 擴展：命中前 2 則 + 命中後 1 則。

ToriiDB 條目於 `history.json` compaction 期間清理 —— 早於 compact cutoff 的條目被移除，讓 ToriiDB 專注於近期對話。較舊資料存於 SQLite（層級 3）。

### 3. 全文封存 —— SQLite FTS5（全部歷史）

每則寫入 `history.json` 的訊息都會透過 go-sqlkit **雙寫**至 SQLite（`~/.config/agenvoy/.store/history.db`）。即使 `history.json` 已 compact，SQLite 始終持有完整對話歷史。

`search_chat_history` 工具搭配 `mode=keyword` 對 SQLite 封存執行 FTS5 全文搜尋 + 對近期條目執行 ToriiDB 子字串比對，並合併結果。

**Compaction：** 當 `history.json` 超過 `max_history_bytes`（預設 5 MiB）時，最舊訊息在完整的 user+assistant pair 邊界上裁剪至 80%。cutoff timestamp 記錄於 SQLite `session_meta.start_at`，使 keyword 搜尋排除 `history.json` 已存在的條目（避免重複）。早於 cutoff 的 ToriiDB 條目亦被移除。

**Backfill：** 首次遇到（SQLite 對某 session 無資料但 `history.json` 有內容）時，整份既有歷史 backfill 進 SQLite。

**Timestamps：** 以 UTC unix 奈秒儲存。訊息內容中的 timestamp 透過 `time.ParseInLocation`（本地時區）解析並轉為 UTC 儲存。搜尋查詢使用 `time.Now().UnixNano()`（已為 UTC）。

### 搜尋路由

| `mode` 參數 | 來源 | 使用情境 |
|---|---|---|
| `semantic`（預設） | ToriiDB VSearch | 「我們討論過關於 X 的什麼？」—— 基於語意 |
| `keyword` | SQLite FTS5（封存）+ ToriiDB 子字串（近期） | 「找出含 'sandbox' 的訊息」—— 精確文字 |

### 跨 session error memory

Tool 失敗、解決路徑與放棄的策略以 **90 天 TTL** 存於 `error_memory`，跨 session 存續。命中時（透過關鍵字 `Contains` 或 `db.VSearch`），該條目的 TTL 透過 `db.Expire` 續期。

當同一 tool 名稱在未來 session 失敗時，`toolCall.go` 自動查詢 `error_memory` 並將相關條目作為 hint 注入下一個 assistant turn：

| 記錄結果 | Hint 行為 |
|---|---|
| `resolved` | Agent 必須套用已記錄的解決方式 |
| `failed` / `abandoned` | Agent 必須避開已記錄的策略 |

## 儲存佈局

| 儲存 | 內容 | 生命週期 |
|---|---|---|
| `history.json` | 近期訊息（hot，LLM 每 turn 讀取） | 5 MiB 時自動 compact |
| ToriiDB `DBSessionHist` | 帶 embeddings 的近期訊息 | compact 時清理（移除早於 cutoff 的條目） |
| SQLite `messages` | 曾寫入的所有訊息（雙寫） | reset / remove-session 時清除 |
| SQLite `session_meta` | `start_at` —— compact cutoff timestamp | reset / remove-session 時清除 |
| `summary.json` | 滾動式 summary blob | reset 後存續 |
| ToriiDB `error_memory` | 帶解決 metadata 的 tool 錯誤記錄 | 90 天 TTL（命中時續期） |

## Reset / remove 行為

| 操作 | `history.json` | ToriiDB `DBSessionHist` | SQLite（messages + meta） | `summary.json` |
|---|---|---|---|---|
| Compact（自動） | 裁剪至 80% | 移除早於 cutoff 的條目 | 不動（已含全部資料） | 不動 |
| Reset（`/reset`） | 刪除 | 清除 | 清除 | 保留 |
| Remove session | 刪除目錄 | 清除 | 清除 | 刪除目錄 |

## Migration 注意

Session 與 error memory 過去存於 per-session JSON 檔。自 ToriiDB v0.5.0 起改存於 embedded store。勿重新引入 JSON 路徑。

***

> [!NOTE]
> 本文件由 Claude 讀取完整原始碼後自動生成。
