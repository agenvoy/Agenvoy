# Execution Engine

每個請求在 `exec.Execute()` 中執行主迴圈，最多 **128 次迭代**。每次迭代：

1. 組裝 messages：`SystemPrompts` + `OldHistories` + `UserInput` + `ToolHistories`
2. 對所選 provider 呼叫 `Agent.Send()`
3. 從 response 解析 `tool_calls`
4. 透過 `toolCall.go` 分派 tool call（three-pass concurrency，見下）
5. 將結果 append 至 `ToolHistories`
6. 當無 `tool_calls` 剩餘或撞到迭代上限時停止

**無 inter-round delay** — rate-limit 保護來自 provider round-trip 延遲、same-error circuit breaker，以及 per-tool 內部限流器（例如 `search_web` 2 秒間隔、`api_*` per-name 1 秒間隔）。

## Three-pass tool concurrency

`toolCall.go` 將每一輪的 tool call 拆成三個序列 pass；只有 Pass 2 會 fan out：

| Pass | 模式 | 工作 |
|---|---|---|
| 1 — pre-flight | 序列 | Cache 命中檢查（`read_file` 略過）、stub-tool 短路、confirm gate、JSON-schema 驗證 |
| 2 — execute | 對標記 `IsConcurrent` 的 tool 並行；其餘序列 | `tools.Execute` |
| 3 — commit | 序列 | 落地 `sessionData.Tools` 與 `ToolHistories`、更新 cache、發出 `EventToolResult`、處理 review tool |

標記為 concurrent 的 tool：`read_file`、`list_files`、`glob_files`、`search_files`、`fetch_page`、`search_google_news`、`send_http_request`、`download_file`、`transcribe_media`、`calculate`、`invoke_subagent`、`search_chat_history`、`search_error_history`、`read_error`、`read_log`、`list_rag`、`search_rag`、`format_chatbot`、`list_chatbot`、`list_tools`、`list_schedule`。`search_web`、write 類 tool、`api_*` 以及 MCP tool 一律序列執行。

## Pending registry

`internal/runtime/pending.go` 是由 prefix 路由的 confirm/ask listener registry，由主 agent 與任何 in-process subagent 共用。Producer（`toolCall` confirm、`ask_user` handler、`store_secret` handler）呼叫 `Ask(ctx, req)` 並阻塞於 per-entry buffered=1 的 reply channel；每個 runtime 透過 `pending.RegisterListener(prefix)` 註冊一個 listener（TUI/CLI 用 `""` 匹配全部，Telegram daemon listener 用 `"tg-"`），並只透過 `PickNextFor(prefix)` 認領匹配的 entry。ctx 取消會移除該 entry，使過期的 producer 絕不浪費一次人為互動。

Gate `pending.HasListener(sessionID)` 檢查是否有匹配 prefix 的 listener 為該 session 註冊。這取代了舊的全域 `pending.Active atomic.Bool`，讓 Telegram、Discord、CLI 的 confirm 流程能並行運作而不互相阻塞。

## Circuit breaker

當 `Agent.Send()` 連續三次回傳相同的 error signature（例如 HTTP 429 且 request payload 相同），迴圈會中止以防止無限 retry 風暴。不同的 error signature 會重置計數器。

## 跨 turn workdir 重置

每則新的 user message 會重建 `Executor`，並透過 `os.Getwd()` 將 `data.WorkDir` 重置為 process cwd — 被 `cd` 改動的 workdir **不會**跨 turn 保留。兩道護欄防止 LLM 從歷史推斷出過期的 workdir：

- **L1（system prompt）** — `Work directory: {{.WorkPath}}` 行，加上明確提醒：先前的 `cd` 文字來自較早的 turn
- **L2（per-message）** — 每則 user message 都被包上含當前時間戳與工作目錄的 metadata header；workDir 行是最強的錨點，覆蓋任何歷史近因偏誤

TUI 透過 `stripUserMetaHeader` 在視覺上剝除該 wrapper；LLM 仍原樣收到。

***

> [!NOTE]
> 本文件由 Claude 讀取完整原始碼後自動生成。
