# REST API

由 `make app` 啟動。HTTP server 僅綁定 `127.0.0.1` — 區網 client 無法連到 daemon。CORS middleware 以來源白名單控管跨來源存取（`web.agenvoy.com` 協作 dashboard 所需）。

## Endpoints

| Endpoint | 說明 |
|---|---|
| `POST /v1/chat/completions` | OpenAI 相容 chat completions（stateless） |
| `POST /v1/send` | 傳送訊息；body `{sid?, persist?, text}` |
| `GET /v1/sessions` | 列出所有 session 及狀態 |
| `GET /v1/session/:sid/status` | 讀取 `status.json`（session 不存在則 404） |
| `GET /v1/session/:sid/log` | `action.log` 的 SSE stream（1 秒 ticker，`: ping` heartbeat） |
| `GET /v1/log?sessions=a,b,c` | 多工 SSE — 單一連線串流多個 session 的事件，每個事件以 `session` 欄位標記 |
| `GET /v1/session/:sid/pending` | 列出某 session 的待處理 confirm/ask 任務 |
| `GET /v1/session/:sid/pending/:hash/questions` | 取得特定待處理任務的問題 |
| `POST /v1/session/:sid/pending/:hash/resume` | 提交答案以恢復待處理任務 |
| `POST /v1/session/:sid/event` | 發布 session 事件（僅 localhost） |
| `GET /v1/tools` | 列出已註冊工具 |
| `POST /v1/tool/:tool_name` | 直接呼叫工具 |
| `GET /v1/key` | 從 keychain 讀取值（僅 localhost） |
| `POST /v1/key` | 寫入值至 keychain |

## `POST /v1/send` 語意

| `persist` | `sid` | 結果 |
|---|---|---|
| `false`（預設） | 空 | 建立 `temp-<uuid>`，閒置 30 分鐘後回收 |
| `true` | 空 | 建立 `http-<uuid>`，永久保留 |
| 任意 | 有提供 | 使用所給的 sid（`persist` 被忽略） |
