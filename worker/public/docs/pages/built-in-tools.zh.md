# 內建工具

## 檔案操作

| 工具 | 說明 |
|---|---|
| `read_file` | 讀取文字、PDF、DOCX、PPTX、CSV/TSV 或圖片檔。必須在 `patch_file` 之前呼叫。**敏感檔案防護**：SSH keys、`.pem`、`.key`、`.env` 一律需要確認，無論是否處於 sudo 或 allowlist |
| `write_file` | 建立或完全覆寫檔案 |
| `patch_file` | 精確字串替換（支援 `replace_all`） |
| `list_files` | 列出目錄內容 |
| `glob_files` | Glob 樣式搜尋 |
| `search_files` | 對檔案內容進行 regex 搜尋 |

## 網頁（唯讀，多數可並行）

| 工具 | 可並行 | 說明 |
|---|---|---|
| `fetch_page` | ✓ | 抓取網頁（readability + 4xx/5xx 透過 ToriiDB 跳過 cache）；`save=true` 會持久化到本地檔案 |
| `search_web` | | DuckDuckGo lite endpoint，package 層級的 rate limit（2 秒間隔） |
| `search_google_news` | ✓ | Google News RSS |

## HTTP

| 工具 | 可並行 | 說明 |
|---|---|---|
| `send_http_request` | ✓ | 原始 HTTP 請求，回傳 status + headers + body。GET 自動放行；其他 method 需要確認。內建 SSRF 防護（DNS 解析後比對 loopback / private / link-local）；透過 `config.json` 的 `net_white_list` 對特定 host 放行 |
| `download_file` | ✓ | 下載二進位檔到本地磁碟（tar.gz、圖片、壓縮檔）；JSON/HTML 請改用 `send_http_request` 或 `fetch_page` |

## 媒體

| 工具 | 可並行 | 說明 |
|---|---|---|
| `transcribe_media` | ✓ | 透過 Gemini `inline_data` 進行本地音訊 / 視訊轉錄（ogg、mp3、wav、m4a、flac、aac、mp4、mov、webm、mpeg、3gp）；每次請求上限 20 MiB。*(需要 gemini credential)* |
| `generate_image` | | 透過 codex@ 訂閱額度以 gpt-image-2 產生圖片。先用 `ask_user` 確認尺寸與品質。輸出為 `[SEND_FILE:<path>]`。15 分鐘上限。*(需要 codex credential)* |

## 工具程式

| 工具 | 可並行 | 說明 |
|---|---|---|
| `calculate` | ✓ | 數學運算式求值器 |

## Agent 編排

| 工具 | 說明 |
|---|---|
| `invoke_subagent` | In-process subagent（不走 HTTP）；支援 `name` / `session_id` / `model` / `system_prompt` / `exclude_tools`。強制排除集：`invoke_subagent` 自身、`invoke_external_agent`、`cross_review_with_external_agents`、`review_result`。`AllowAll` 與 `WorkDir` 繼承自 parent ctx |
| `invoke_external_agent` | 一次性外部 CLI（claude / codex / copilot / gemini）；`readonly` flag 控制寫入權限。Subprocess timeout 上限由 `MAX_EXTERNAL_AGENT_TIMEOUT_MIN` 決定（預設 10 分鐘） |
| `cross_review_with_external_agents` | 串接四個外部 CLI，最多進行三輪 review（`MaxVerifyRounds=3`，package const）。15 分鐘硬上限 |
| `review_result` | 內部 priority-model 自我 review |
| `generate_plan` | 回傳結構化 markdown plan（需求摘要 / 前置條件 / 步驟 + 驗收 / 整體驗收 / 風險 / fallback）。使用 `exec.SelectAgent` 加上 `[plan]` prefix 觸發 P0.6 routing，導向強推理 agent。`toolDefs=nil` — 只做 plan，不執行。5 分鐘上限 |

## 互動

| 工具 | 說明 |
|---|---|
| `ask_user` | 自由文字 / 單選 / 多選 / `secret` 遮罩輸入提示；active 時透過 `pending` registry 路由，否則 fallback 到 stdin（CLI）或非互動式指引 |
| `store_secret` | 透過遮罩輸入擷取一個值並直接寫入 keychain — **該值永遠不會進入 LLM context、history 或 logs**。Schema **不**接受 `value` 參數；agent 只看得到 `name` + 說明 |
| `install_dependence` | 跨平台安裝缺失的系統 binary（僅 TUI/CLI）。若已在 PATH 則略過。Sandbox 會擋 sudo，因此此工具會繞過 sandbox。語言層級套件（pip/npm/cargo/gem）→ 輸出指令供使用者手動執行 |

## 記憶

| 工具 | 說明 |
|---|---|
| `search_chat_history` | 對當前 session 的 history 進行關鍵字 + 語意雙軌搜尋 |
| `remember_error` | 記錄一個工具錯誤及其解法 / 策略 |
| `search_error_history` | 對 error memory 進行跨 session 語意搜尋 |
| `read_error` | 依 key 讀取特定的錯誤條目 |

## 診斷

| 工具 | 說明 |
|---|---|
| `read_log` | 回傳 daemon.log 近期的 WARN/ERROR 行（最近 `h` 小時） |
| `report_error` | 掃描 daemon.log 的 WARN/ERROR 行並上傳到 report.agenvoy.com。Fire-and-forget |

## RAG

透過 KuraDB child process 進行外部文件 RAG。當 `~/.config/kuradb/endpoint` 不存在時，這些工具會**逐 turn 動態排除** — KuraDB 關閉時 LLM 永遠不會看到它們。

| 工具 | 說明 |
|---|---|
| `list_rag` | 列出可用的 KuraDB 資料庫（例如 `notes`、`inbox`、`code`） |
| `search_rag` | 依關鍵字（`mode=keyword`、`gse` 斷詞、支援中文）或語意（`mode=semantic`、OpenAI `text-embedding-3-small`）搜尋資料庫 |

當 `list_rag` / `search_rag` 工具載入後，system prompt 會強制任何資訊查詢的**第一波**工具呼叫為 `list_rag` + `search_rag`。外部 web/search 工具退為次要（補缺口），而非 fallback 或替代。

## 渲染

| 工具 | 說明 |
|---|---|
| `render_page` | 覆寫當前 session canvas 的渲染 HTML 頁面；瀏覽器分頁透過 SSE 自動 reload |

## Channel

跨 session 推播工具與 channel 格式參考。每個工具同時 gate 在 `cfg.{T,D}Enabled` 與 keychain credential 的存在。

| 工具 | 說明 |
|---|---|
| `list_chatbot` | 列出指定平台（`platform=telegram` 或 `platform=discord`）已授權的 chat。*(需要 telegram 或 discord)* |
| `send_to_chatbot` | 依 `target_id` 發送格式化訊息到已授權的 chat。需要 `platform` 參數。Telegram：HTML + transient client。Discord：markdown + transient client。*(需要 telegram 或 discord)* |
| `format_chatbot` | `AlwaysLoad=true`；回傳指定平台的完整格式化參考（Telegram HTML 或 Discord markdown）。*(需要 telegram 或 discord)* |

## 輸出標記（channel-specific 行為）

任何工具或 LLM 回應的輸出文字都會針對以下標記做後處理：

| 標記 | 行為 |
|---|---|
| `[SEND_FILE:<path>]` | Channel runtime 自動附加該檔案（Telegram → 依副檔名拆分 photo/document，Discord → 統一 `SendFiles` 每則訊息批次 10 個） |
| `[SEND_VOICE:<text>]` | 僅 Telegram。透過 Gemini TTS 合成，以 OGG voice 發送。Run.go 以 **async** 方式觸發上傳（`go func` 搭配 `context.WithoutCancel`）；reply 文字立即回傳。失敗 → `slog.Error` + chat 通知（絕不靜默） |

標記 regex + 去重 + `os.Stat` 過濾住在 `internal/utils/utils.go`。Telegram 專屬的 photo/document 拆分 wrapper 在 `internal/runtime/telegram/fileMarker.go`。Push hooks（`telegram.PushTelegramResult` / `discord.PushDiscordResult`）呼叫同一個 extractor。

## Skill 探索

| 工具 | 說明 |
|---|---|
| `run_skill` | 將一個 skill 載入當前 loop（合成一組 tool_call/tool_result pair 進 ToolHistories） |
| `search_tools` | 搜尋已註冊的工具目錄 |
| `list_tools` | 列出所有已註冊的工具 |

## Skill 與 tool 變體（always-allowed 的 `write_file` 變體）

| 工具 | 說明 |
|---|---|
| `write_skill` | 在 `~/.config/agenvoy/skills/` 下建立或重寫檔案 |
| `patch_skill` | 對 skill 檔案進行字串替換 |
| `remove_skill` | 將 skill 目錄移到 `.Trash/` |
| `write_tool` | 在 `~/.config/agenvoy/tools/script/` 下建立或覆寫 `tool.json` 或 `script.py` |
| `patch_tool` | 對 script tool 檔案（`tool.json` 或 `script.py`）進行字串替換 |
| `test_tool` | 在 sandbox 內以 JSON input 執行 script tool 的 `script.py` |
| `remove_tool` | 將 script tool 目錄移到 `.Trash/` |

所有變體皆為 always-allowed 且限定在各自的目錄內。每次 write/patch/remove 都會自動 commit 到對應的 git repo（skills 或 tools）。`write_tool` 與 `write_skill` 支援並行呼叫。

## Git 版本控管與自我改進

| 工具 | 說明 |
|---|---|
| `git_log` | 列出 skills 或 tools 目錄的 git commit 歷史（`tag` = `skills` 或 `tools`） |
| `git_rollback` | 將 skills 或 tools 目錄回滾到指定的 git commit（`tag` = `skills` 或 `tools`） |

**自我改進 loop**：當 skill 執行產生工具錯誤（工具名稱錯誤、步驟失敗），`postSkillImprove` 會在 `Execute` 結尾同步執行。它載入內建的 `improve-skill` 定義，餵入執行 trace，重寫出錯的 SKILL.md/scripts，並自動 commit 修正。

## 系統

| 工具 | 說明 |
|---|---|
| `run_command` | 執行系統指令（argv-only schema，透過 `go-pkg/sandbox` 包裹 sandbox）；`cd` 為特例，直接 mutate `Executor.WorkDir` 而不經過 sandbox |

## Scheduler

| 工具 | 說明 |
|---|---|
| `add_schedule` | 將既有的 scheduler skill 綁定到一次性觸發時間（`target=task`）或 5 欄位 cron 運算式（`target=cron`）。Task 時間格式：`+5m`（相對）、`HH:MM`（今天）、`YYYY-MM-DD HH:MM`，或 RFC3339。**必須由 `scheduler-skill-creator` skill 呼叫，不可直接呼叫** — 直接呼叫要求該 skill 已存在於 `~/.config/agenvoy/skills/scheduler/<short>-<hash8>/`。 |
| `patch_schedule` | 依 `skill_name` 與 `target` 重新排程；只改時間，不動綁定的 SKILL 本體。 |
| `remove_schedule` | 依 `skill_name` 與 `target` 取消；綁定的 scheduler skill 目錄會移到 `.Trash/`。 |
| `list_schedule` | 列出當前 session 的 tasks 和/或 crons。`target` 接受 `task`、`cron` 或 `all`（預設）。 |

`scheduler-skill-creator` 是高階 skill，負責**建立** scheduler skill 本體並呼叫 `add_schedule` 來綁定它。新的週期性 / 一次性請求應啟動該 skill，而非直接呼叫低階工具。

Daemon 端的 runtime（`internal/runtime/scheduler.go`）以 fsnotify 監看 `~/.config/agenvoy/{tasks,crons}.json`，並在 Write / Create / Rename 時 hot-reload。逾期的 tasks 會在啟動或 reload 時自動觸發並移除；觸發透過 `runtime.SetRunner` → 在 scheduler skill 本體上執行 in-process subagent（always-allow context）。

TUI 提供三個 slash 指令來管理排程：`/cron`、`/task`（add / remove / edit），以及 `/sched-<name>`（手動觸發既有的 scheduler skill 本體）。
