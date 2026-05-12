---
name: scheduler-skill-creator
description: |
  建立並排程定時觸發的 skill。**所有新增定時／週期任務、提醒、排程通知的請求必須走此 skill**，禁止直接呼叫 add_task / add_cron（那是 skill 已存在時的時間綁定工具，不該作為新建排程的入口）。

  必定觸發的訊息特徵（任一即活化）：
  - 相對延遲：「X 分鐘後」「X 小時後」「稍後」「待會」「等一下」
  - 明確時間：「X 點」「下午 X 點」「明天 X 點」「後天」「YYYY-MM-DD HH:MM」
  - 週期性：「每 X 分鐘」「每小時」「每天」「每週」「每月」「定時」「固定」
  - 提醒 / 通知意圖：「提醒我」「通知我」「告訴我」+ 時間描述

  範例觸發訊息：「5 分鐘後提醒我喝水」「每天早上 9 點抓 HN 頭條」「明天下午 3 點開會」「每 5 分鐘查台積電股價」。

  流程：解析訊息抽出「要做什麼」「何時觸發」→ 缺項用 ask_user 補問 → 生成 skill 檔案至 ~/.config/agenvoy/skills/scheduler/<short>/SKILL.md（無 scheduler- 前綴）→ 呼叫 add_task 或 add_cron 綁定時間 → 回報。
---

# Scheduler Skill 建立器

## 目的

scheduler 採 skill-based 觸發：到時間時，daemon 讀 `scheduler/<short>/SKILL.md` body 並起 in-process subagent 跑（always-allow）。本 skill 的職責 = 「從使用者意圖建出 skill 並綁定時間」，完整跑完 6 步即完成排程。

**重要：scheduler 用 skill 與一般 skill 隔離**

| 比較 | 一般 skill | scheduler 用 skill |
|---|---|---|
| 路徑 | `~/.config/agenvoy/skills/<name>/SKILL.md` | `~/.config/agenvoy/skills/scheduler/<short>/SKILL.md` |
| frontmatter `name` | `<name>` | `<short>` (**無前綴**) |
| 一般 `/<name>` 補全 | 出現 | **不出現**（scanner 不掃 scheduler/） |
| 呼叫方式 | `/<name>` 觸發 | `add_task(skill_name=<short>)` / `add_cron(skill_name=<short>)` |

## 成功標準

- 生成檔案: `~/.config/agenvoy/skills/scheduler/<short>/SKILL.md`，frontmatter `name: <short>`（無前綴）
- skill body 描述任務行為、引用具體 tool、（必要時）含 Notify 段
- 呼叫 `add_task(time, skill_name=<short>)` 或 `add_cron(time, skill_name=<short>)` 綁定時間成功
- 回報生成位置、short name、排程類型（one-shot／recurring）、下次觸發時間

## 步驟

### 1. 解析需求並補齊缺漏

從使用者訊息抽兩個元素：

- **任務**：要做什麼（行為描述）
- **時間**：何時觸發

範例解析：

| 訊息 | 任務 | 時間 |
|---|---|---|
| 每 5 分鐘提醒我台積電最新股價 | 查台積電股價並提醒 | 每 5 分鐘（recurring） |
| 明天早上 9 點提醒我開會 | 開會提醒 | 明天 09:00（one-shot） |
| 5 分鐘後叫我喝水 | 喝水提醒 | +5m（one-shot） |
| 每天抓 HN 頭條給我 | 抓 HN 頭條摘要 | 每天（recurring，需追問時段） |

兩者皆有 → 直接進步驟 2。任一缺失 → 用 `ask_user` 補問：

- **缺任務** → 「要做什麼？」（舉 1–2 個範例引導）
- **缺時間** → 「什麼時候執行？」附範例：「每 5 分鐘 / 5 分鐘後 / 明天 9 點」
- **缺時段（recurring 但沒指定幾點）** → 「每天的幾點？」

一次問一題，依需要追問。**禁止假設**，不要用「應該是早上 9 點」之類腦補。

### 2. 時間正規化 + 選 tool

| 使用者說 | 工具 | `time` 參數 |
|---|---|---|
| `X 分鐘後` | `add_task` | `+Xm` |
| `X 小時後` | `add_task` | `+Xh` |
| `今天 X 點`（24h） | `add_task` | `HH:MM` |
| `明天 / 特定日期 X 點` | `add_task` | `YYYY-MM-DD HH:MM` |
| `每 X 分鐘` | `add_cron` | `*/X * * * *` |
| `每小時` | `add_cron` | `0 * * * *` |
| `每天 X 點` | `add_cron` | `MM HH * * *` |
| `每週 N`（0=Sun, 1=Mon, ..., 6=Sat） | `add_cron` | `MM HH * * N` |
| `每月 D 日 X 點` | `add_cron` | `MM HH D * *` |

決定走 `add_task`（一次性）或 `add_cron`（週期）。

### 3. 初始化 skill 目錄（**強制走 init 腳本**）

> **禁止直接用 `write_file` 建立 SKILL.md** —— LLM 容易寫成 `<short>.md` 而非 `<short>/SKILL.md`，或誤加 `scheduler-` 前綴。必須先跑 init 腳本。

用 `run_command` 執行：

```bash
python3 scripts/init_scheduler_skill.py <short-name>
```

`<short-name>` 由步驟 1 的任務描述推導（kebab-case、**不含 `scheduler-` 前綴**）。腳本會：

- 正規化 short name（lowercase、hyphen-case）
- 若 skill 目錄已存在 → 印 `[OK] already exists` 並 exit 0，**跳到步驟 5**（除非使用者明確要求修改 body 才回步驟 4）
- 否則建立 `~/.config/agenvoy/skills/scheduler/<short>/SKILL.md`，寫入含 frontmatter `name: <short>` 的 TODO 模板

判斷依據：腳本 stdout 出現 `created` 走步驟 4；出現 `already exists` 直接跳步驟 5。

### 4. 填充 skill body（**僅新建時**）

> 若步驟 3 已是既有 skill (`already exists`)，**跳過本步驟**，直接到步驟 5。

用 `patch_file` 取代模板中的 `[TODO: ...]` 段：

- `description:` ← 步驟 1 收集到的「一句話描述」
- `## 任務` ← 步驟 1 收集到的「行為細節」，引用具體 tool 名稱與參數
- `## 輸出格式` ← 期望輸出形式

caller session 為 `dc-*` 起源時，在末段追加 Discord Notify 段（見下方規則）。

### 5. 綁定時間

依步驟 2 結果呼叫：

```
add_task(time="<time_value>", skill_name="<short>")
# 或
add_cron(time="<cron_expression>", skill_name="<short>")
```

`skill_name` **不加 `scheduler-` 前綴**（內部會直查 `~/.config/agenvoy/skills/scheduler/<short>/SKILL.md` 確認存在）。session_id 內部自動取 caller `e.SessionID`，不必傳。

成功會回 `ID: <hash>` 等資訊。失敗（skill 不存在、cron 表達式錯誤、`time` 已過）就 abort 本流程，向使用者回報原因。

### 6. 回報

簡短告知：

- skill 已建立: `~/.config/agenvoy/skills/scheduler/<short>/SKILL.md`
- skill name: `<short>`（無前綴）
- 排程: `add_task` / `add_cron` 的回應內容（含下次觸發時間、ID）

## 命名規則

| 項目 | 規則 | 範例 |
|---|---|---|
| short name | lowercase / hyphen-case | `daily-hn-digest`、`tsmc-stock-watch` |
| 目錄 | `~/.config/agenvoy/skills/scheduler/<short>/` | `.../tsmc-stock-watch/` |
| frontmatter name | `<short>` | `tsmc-stock-watch` |
| add_task / add_cron skill_name | `<short>` | `tsmc-stock-watch` |

**禁止**在任何環節加 `scheduler-` 前綴。`scheduler` 已表達於目錄路徑，加前綴只會造成 `scheduler/scheduler-foo/` 之類的重複命名。

## Discord Notify 段（條件性）

caller session ID 以 `dc-` 開頭時，於 skill body 末段插入：

```markdown
## Notify

完成後將最終結果以 `send_http_request` 或 MCP discord tool 推送至 channel `<channel_id>`。
```

`<channel_id>` 從 caller session ID 推斷（`dc-<guild>-<channel>-...` 格式抽取）或直接問使用者。

caller 來源非 `dc-*`（TUI/CLI/HTTP）時跳過此段，結果留在 session history 由 caller 查看。

## 時間敏感性提醒（寫入 skill body 時注意）

被觸發的 skill 跑在獨立 subagent session，**沒有當下對話上下文**。skill body 必須：

- 不依賴「使用者剛才說了什麼」
- 不假設特定變數已被定義
- 引用具體 tool 名稱與參數（自包含可重現）
- cron 觸發時反覆執行，邏輯應 idempotent 或自帶 dedup

## 完整範例

使用者：「每 5 分鐘提醒我台積電最新股價」

**步驟 1** 解析：任務 = 查 2330.TW 股價；時間 = 每 5 分鐘 → recurring。兩者皆有，不問。

**步驟 2** 正規化：`add_cron(time="*/5 * * * *", ...)`

**步驟 3** `run_command python3 scripts/init_scheduler_skill.py tsmc-stock-watch`

**步驟 4** `patch_file` 填入：

```markdown
---
name: tsmc-stock-watch
description: 每 5 分鐘抓取台積電 2330.TW 即時股價並提醒。
---

# Tsmc Stock Watch

## 任務

呼叫 `fetch_yahoo_finance` 取 `2330.TW` 的最新報價。

## 輸出格式

`台積電 2330.TW: NT$<price> (<change>% 從昨收)` 一行。
```

**步驟 5** `add_cron(time="*/5 * * * *", skill_name="tsmc-stock-watch")`

**步驟 6** 回報：「已排程每 5 分鐘觸發 `tsmc-stock-watch`。下次觸發 HH:MM。」

## 不做的事

- **不**用 `write_file` 直接建立 SKILL.md —— 必須走 `init_scheduler_skill.py`，避免結構錯誤（`<name>.md` vs `<name>/SKILL.md`）
- **不**在 short name、frontmatter、skill_name 任何位置加 `scheduler-` 前綴
- **不**留 `[TODO: ...]` 佔位符在最終 skill —— 步驟 4 須把所有 TODO 替換為具體內容
- **不**用任意預設值補齊時間 —— 缺時間就 `ask_user` 問清楚，不要「應該是 9 點」之類腦補
- **不**跳過步驟 5 的 `add_task` / `add_cron` —— skill 建立但沒綁時間 = 排程不會觸發
