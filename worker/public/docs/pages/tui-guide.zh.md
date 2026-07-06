# TUI Guide

單一 bubbletea textarea（`internal/runtime/tui`）；slash command 開啟 transient popup，關閉後乾淨回到 prompt。

## 快捷鍵

| 按鍵 | 動作 |
|---|---|
| `Ctrl+S` | 送出當前 textarea 內容（Enter 插入換行；`Alt+Enter` 同樣插入換行） |
| `/` | 開始 slash-command 過濾（popup picker —— Up / Down 導覽、Tab / Enter 自動補全進 textarea、Esc 關閉） |
| `Up` / `Down`（空白 / 單行輸入時） | 瀏覽輸入歷史（per-session `input_history` 檔） |
| `Esc` | 取消執行中的 exec（若執行中）或關閉當前 popup |
| `Ctrl+C` | 離開 TUI（daemon 持續執行） |

TUI 自動 tail 當前 session 的 `action.log`（外部行程寫入以 warn-purple 的 `▌ ` 為前綴）。僅單 session 視圖 —— 多 session dashboard 已封存。

## Auto mode

按 **Shift+Tab** 切換 auto mode。當前模式顯示於 TUI 左下角：

- `[safe]`（預設） —— tool call 執行前需使用者確認
- `[auto]` —— 所有 tool call 自動核准（`allowAll = true`）；sandbox 與 validator 仍套用

Auto mode 為 session-local，TUI 重啟時重置。亦可於啟動時透過 `agen --allow-all` 設定。

## Slash commands

| 命令 | 說明 |
|---|---|
| `/switch` | 挑選 session（當前已預選；底部有 `(new session)` 哨兵項）。 |
| `/new [name]` | 建立 session；選填 name 將其釘進 registry（會檢查衝突）。 |
| `/bot [name body...]` | 編輯 bot 人設 —— 雙 popup 表單（先 name 後多行 body），或以 inline `parts>=3` 走快速路徑。 |
| `/model [global\|session\|dispatch\|summary\|reasoning]` | `global` —— 從 registry 新增 / 移除，`session` —— 從 `cfg.Models` 挑選，`dispatch` —— 設定 dispatcher model，`summary` —— 設定 summary model（或 `(use dispatcher)` 以退回），`reasoning` —— 設定 reasoning 深度（`low` / `medium` / `high`）。 |
| `/mcp [add\|remove]` | MCP server 設定的鏈式 popup 表單；重啟 daemon 以套用。 |
| `/feature [voice\|image2\|kuradb]` | `voice` —— 啟用 / 停用語音訊息處理，`image2` —— 啟用 / 停用 gpt-image-2 生成，`kuradb` —— 切換 KuraDB RAG。 |
| `/discord [enable\|disable]` | 切換 Discord bot 連線（TUI 內 popup 鏈：token 輸入、驗證、keychain 寫入、daemon fsnotify reload）。 |
| `/telegram [enable\|disable]` | 切換 Telegram bot 連線（與 `/discord` 相同的 TUI 內 popup 鏈；第一個傳訊給 bot 的 chat 須通過 in-chat 6 位數 OTP，隨後 chat ID 會 append 到 `~/.config/agenvoy/.telegram`）。 |
| `/cron [add\|remove\|edit]` | 週期性排程。`add` —— 多行需求，派送 `/scheduler-skill-creator <requirement>`（skill 透過 `ask_user` 詢問缺少的 when/what）。`remove` —— 列出、確認、`runtime.RemoveCron` + 丟棄 skill 目錄。`edit` —— 列出、需求、agent 選擇 `patch_schedule(target=cron)` 或改寫 SKILL body。Picker 為 session-scoped —— 僅顯示 `session_id == currentSessionID` 的條目。 |
| `/task [add\|remove\|edit]` | One-shot 任務（比照 `/cron`；使用 `add_schedule` / `patch_schedule` / `remove_schedule` 搭配 `target=task`）。Session-scoped picker。 |
| `/sched-<name>` | 於 slash picker 中在一般 skill 之後呈現（warn-purple 標籤）—— 挑選既有 scheduler skill 並派送其主體，帶明確的「execute, do NOT activate scheduler-skill-creator」preamble。依 session 過濾 —— 僅顯示綁定至當前 session task/cron 條目的 skill。 |
| `/sudo` | 切換當前 session 的 sudo mode（1 小時 TTL）。繞過 confirm gate —— system-dir floor 路徑仍被封鎖。存於 ToriiDB，不落磁碟。 |
| `/reset` | 重置當前 session 歷史（兩階段確認 popup）。清除 `history.json`、`action.log` 與 DB keys；保留 `bot.md`、`summary.json`、`status.json`。 |
| `/dangerous [remove-session\|allow-skill\|allow-cmd\|allow-report]` | `remove-session` —— 刪除當前 session（雙重確認），`allow-skill` —— 標記 skill 為 always-allow（繞過 confirm gate），`allow-cmd` —— 將 binary append 至 `white_list`，`allow-report` —— 啟用 / 停用 error report 上傳。 |
| `/history` | 重載可見 transcript —— 清畫面、重印 header、渲染 session `action.log` 最後 100 條。 |
| `/log` | 於 `$PAGER` 開啟原始 `action.log`（fallback `less -Rf +G`，跳至底部）。`\x1F` marker 展開為換行以利閱讀。 |
| `/cmd` | 直接在當前 workDir 執行 shell 命令（`sh -c`）。 |
| `/update` | 確認後，透過 `tea.ExecProcess` 執行 `agen stop && agen update`、退出（以 `agen` 重新 attach 以接上新 binary）。 |
| `/clear` | 僅清除終端顯示 —— memory 不動。 |
| `/exit`、`/quit` | 離開 TUI。 |

***

> [!NOTE]
> 本文件由 Claude 讀取完整原始碼後自動生成。
