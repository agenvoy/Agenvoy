# 工具設計與規則

## 工具設計規則

新增或編輯工具的四條強制規則（build-time 契約在 `tool_generate_guide`；後續品質由 `code-reviewer` skill 檢查 — 原本獨立的 `/tool-reviewer` skill 已被這兩者取代並移除）：

1. **name 是唯一的語意載體** — stub-tool 的首輪呼叫只看得到 name；description 與 params 在第二輪才抵達
2. **description 固定 3 行、60-200 字元** — What（核心動作）／When（觸發時機 vs 替代方案，例：`use for X; Y for Z`）／Precondition（關鍵限制，無則省略）。不放填充語、不用粗體、不放輸出 schema dump。
3. **僅限英文** — 中文只出現在面向使用者的 handler 回傳訊息
4. **選用欄位必須宣告 `default`** — handler 仍需防禦 nil/缺失

觸發條件與同類工具比較是 When 行的必要內容，不是禁止項——只寫核心動作而沒有觸發訊號的 description 視為不完整。Parameter description 須涵蓋 How／When／Example，非平凡型別（object/array/enum）若說明少於 20 字元視為不完整。

## 工具並行標記

工具有兩個獨立旗標：

- `ReadOnly` — 使用 `agen cli` 時豁免 confirm gate
- `Concurrent` — 選擇加入 Pass 2 fan-out（每次呼叫一個並行 goroutine）

加入 `Concurrent: true` 需同時滿足「無副作用」與「上游允許並行」。當前的並行工具集記載於 Core Concepts（three-pass tool concurrency）。

## 工具 timeout 矩陣

每個 adapter 有自己的 timeout，與 executor 端的上限層疊：

| Adapter | 預設 | 可設定 | 位置 |
|---|---|---|---|
| Built-in（`toolRegister.Dispatch`） | 1 分鐘 | 每個工具的 `Def.Timeout` | 工具註冊 |
| Script（`script_*`） | 5 分鐘（300s） | `tool.json` `"timeout": <seconds>` | `extensions/scripts/<name>/tool.json` |
| API（`api_*`） | 60s | `doc.Endpoint.Timeout`；硬上限 300s | `extensions/apis/<name>.json` |
| MCP HTTP | 60s `http.Client.Timeout` + 1 分鐘外層 dispatch | 無 | MCP server config |
| MCP stdio | 僅 1 分鐘外層 dispatch | 無 | MCP server config |

長時間執行的工具（script + API）每 30s 向 daemon log 發出 `running name=... elapsed=Ys/Zs` 以利可見性。

Subagent 與 external-agent 工具有各自的數分鐘級上限（`invoke_subagent` = `MAX_SUBAGENT_TIMEOUT_MIN`、`invoke_external_agent` = 10 分鐘、`cross_review_with_external_agents` = 15 分鐘、`generate_plan` / `transcribe_media` = 5 分鐘、`generate_image` = 15 分鐘）。

## 憑證自動修復

`store_secret` 設為 `AlwaysLoad: true`，因此 agent 在首輪即可見。當下游工具回傳缺 key 或無效憑證錯誤（`401` / `403` / `invalid api key` / `expired token`）時，system prompt 的 `§10 Credential auto-heal` SOP 會指示 agent 呼叫 `store_secret`（透過遮罩輸入取得新值 — 該值永不到達 LLM）並重試原工具。每個失敗工具每回合上限為兩輪 `store_secret`。
