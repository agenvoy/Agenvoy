# Scheduler & Self-Improvement

本頁涵蓋自動化排程與自我修復機制。

## Scheduler skills（隔離 namespace）

由 scheduler 觸發的 skill 位於獨立樹狀結構，**不**被 `host.Scanner()` 掃描：

```
~/.config/agenvoy/skills/scheduler/<short>-<hash8>/SKILL.md
```

| 面向 | 一般 skill | Scheduler skill |
|---|---|---|
| 路徑 | `~/.config/agenvoy/skills/<name>/SKILL.md` | `~/.config/agenvoy/skills/scheduler/<short>-<hash8>/SKILL.md` |
| Frontmatter `name` | `<name>` | `<short>-<hash8>`（無 `scheduler-` 前綴） |
| `/<name>` 自動補全 | 有 | 無 —— 於 picker 底部以 `/sched-<name>`（warn-purple）呈現 |
| 觸發 | `MatchSkillCall` 後執行 run_skill | (a) 由 daemon `runtime.SetRunner` 觸發 cron / one-shot，(b) 從 TUI 手動 `/sched-<name>` |

### 建立流程

`scheduler-skill-creator` skill 是**新增**排程的正規入口。它會：

1. Pre-flight gate（Step 0）：若使用者訊息缺少時間 token（`+5m` / `HH:MM` 等）或任務 token，必須先呼叫 `ask_user` —— 不預設 `+10m`、不推測「大概是早上 9 點」。
2. 執行 `python3 scripts/init_scheduler_skill.py <short>` 建立帶 hash 後綴的 skill 目錄。
3. Patch SKILL.md 主體（description + task + output format）。
4. 呼叫 `add_schedule` 綁定排程。

直接呼叫 `add_schedule` 僅允許用於**重新綁定**既有排程（變更已建立之 scheduler skill 的時間）。

### 手動執行（`/sched-<name>`）

TUI 命令 picker 將 `scheduler/` 下每個目錄列為 `/sched-<name>`。選取後讀取其主體並派送給當前 agent，**帶一段 preamble**，阻止較弱的模型將 SKILL.md 形狀的主體誤讀為建立排程請求而重跑 creator。

該 preamble 強制：

- 立即執行既有 scheduler skill 並輸出結果
- **不**啟用 `scheduler-skill-creator`
- **不**執行 `init_scheduler_skill.py`
- **不**呼叫 `add_schedule`

### Daemon 觸發

`runtime.SetRunner` 註冊 `runSkill(ctx, sessionID, skillName)`。當 scheduler 觸發（cron tick 或 one-shot 到期）時，runner：

1. 透過 `filesystem.ScheduleSkillBody(skillName)` 讀取主體。
2. 確保 session 目錄存在；寫入預設 `bot.md`。
3. 呼叫 `exec.ExecWithSubagent(ctx, body, sessionID, "", "", nil)` —— 一個帶 always-allow context 的 in-process subagent。

One-shot 任務成功觸發後會被移除，skill 目錄則被丟棄。

## Self-improvement（失敗時自動修復）

當 skill 執行遇到 tool 錯誤時，Agenvoy 自動改寫 skill 定義，以在下次防止同一失敗。這是閉環演化循環 —— 無需使用者介入。

### 運作方式

1. **Trace 收集** —— 於 `Execute` 期間，每個 tool call 結果記錄為 `execStep{Tool, Error}`。若該 skill 的 session 使用 `run_skill` 啟用其他 skill，那些也會被追蹤。
2. **觸發** —— 於 `Execute` 結尾（在 defer 中），若 trace 含至少一個錯誤，`postSkillImprove` 對每個已啟用 skill 同步執行。
3. **Improvement agent** —— `postSkillImprove` 載入內建 `improve-skill` skill 主體，以執行 trace 建立 stateless session，並執行完整 `Execute` 迴圈（2 分鐘 timeout、always-allow、無互動式工具）。
4. **Auto-commit** —— 成功時，`skill.AutoCommit(ctx, "improve", skillName)` 將 `~/.config/agenvoy/skills/` 下改寫的檔案 commit 至 git，建立改進的版本化歷史。

### improve-skill 修復的內容

| 偵測到的問題 | 動作 |
|---|---|
| SKILL.md 中錯誤的 tool 名稱（例如用 `Bash` 而非 `run_command`） | 換成正確的已註冊 tool 名稱 |
| 某步驟造成重複失敗 | 加入 fallback 策略或移除該步驟 |
| 不清楚的指示造成 LLM 誤呼叫 | 以更精確措辭改寫 |
| 步驟順序造成依賴失敗 | 重排步驟 |
| 錯字或文法問題 | 修正同時保留原語言 |

### 限制

- Improvement agent 不能使用互動式工具（`ask_user`）、web 工具（`search_web`、`fetch_page`）、媒體工具（`generate_image`）或 agent 編排工具
- 僅可用檔案讀寫與命令執行 —— agent 讀取當前 skill、分析 trace、寫入修正後的檔案
- 改進範圍僅限於該 skill 自身的檔案；system prompt 與 runtime code 永不被修改
- 每次改進執行皆為無 conversation 歷史的 stateless session

### Rollback

所有 skill 變更皆自動 commit 至 git。使用 `git_log`（tag=skills）檢視歷史，或 `git_rollback`（tag=skills）還原不想要的自動修復。

---

> [!NOTE]
> 本文件由 Claude 讀取完整原始碼後自動生成。
