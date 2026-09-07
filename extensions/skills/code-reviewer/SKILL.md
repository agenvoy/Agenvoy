---
name: code-reviewer
description: Analyze project source code and generate optimization suggestions. Use when user wants code review, performance optimization advice, security hardening recommendations, or architecture improvement suggestions.
---

> **本 Skill 為 Agenvoy 內部最佳化版本**，依 Agenvoy 的執行環境撰寫（`run_command` 的 CWD、`~/.config/agenvoy/skills/.system/` 安裝位置、`edit_skill`／`schedules`／`find_edit_tool` 等工具、subagent 與排程的觸發路徑），**不保證適配其他 AI harness**。

# Code Reviewer

AST 驅動的專案原始碼分析，產生優化建議報告（Go / Python / JavaScript / TypeScript）。

## Command Syntax

```
/code-reviewer [PROJECT_PATH] [OUTPUT_FILE]
```

### Parameters (All Optional)

| Parameter | Default | Description |
|-----------|---------|-------------|
| `PROJECT_PATH` | Current directory | 專案根目錄路徑 |
| `OUTPUT_FILE` | `.doc/code-reviewer/{yyyy-MM-dd_HH-mm}.md` | 輸出檔案路徑（相對於 `PROJECT_PATH`） |

### Output Path Rules

- **預設路徑**：`{PROJECT_PATH}/.doc/code-reviewer/{yyyy-MM-dd_HH-mm}.md`
  - 時間戳使用 24 小時制本地時間（例：`2026-04-25_14-30.md`）
  - 不存在時自動建立 `.doc/code-reviewer/` 目錄
- **顯式覆寫**：傳入 `OUTPUT_FILE` 時直接使用該路徑；路徑若含目錄請先自行建立
- **永遠不在專案根目錄落檔** — 所有產出集中於 `.doc/code-reviewer/`
- **零問題 + 零有效建議時不落檔** — 見下節「No-Op 條件」

### Examples

```bash
/code-reviewer                           # → .doc/code-reviewer/2026-04-25_14-30.md
/code-reviewer ./my-project              # → my-project/.doc/code-reviewer/2026-04-25_14-30.md
/code-reviewer . custom.md               # → ./custom.md（顯式覆寫）
```

---

## Supported Languages

| Language | Analyzer | Dependencies |
|----------|----------|--------------|
| Go | `go/ast`（via `go run` helper）+ 字串掃描 | `go` ≥ 1.21 |
| Python | 內建 `ast` 模組 | Python ≥ 3.10 |
| JavaScript / TypeScript | 內建輕量結構掃描（brace-based 函式邊界／巢狀深度）+ 專案本地 `eslint`（可選）+ 字串掃描 | `node_modules/.bin/eslint`（可選） |

> 對應工具鏈不可用時，自動降級為字串掃描並在報告中標示。

---

## Workflow

```
1. Detect    →  偵測專案主要語言（依 go.mod / tsconfig.json / package.json / pyproject.toml）
2. Analyze   →  呼叫對應分析器（AST + 字串掃描）
3. Evaluate  →  計算指標並依嚴重度排序問題；比對 Convention Adherence 的規範檔
4. Validate  →  逐條回原始碼確認錨點；確認不了的直接刪除（見 Validation Pass）
5. Gate      →  檢查 No-Op 條件；若命中則跳過 Generate / Save，僅輸出無需處理訊息
6. Generate  →  產生優化建議報告（繁體中文），套用 Recommendation Principles 過濾
7. Save      →  `mkdir -p {PROJECT_PATH}/.doc/code-reviewer/` 後寫入 `{yyyy-MM-dd_HH-mm}.md`
```

### Validation Pass

Evaluate 產出的每一條問題與建議，在寫進報告前獨立確認一次：宣稱不存在的符號，回檔案確認它真的不存在；宣稱違反規範，確認那條規則的原文存在且作用範圍涵蓋該檔案。確認不過的**移出清單**，不降級為 Low 保留。

**為何：** 假陽性侵蝕整份報告——讀者被一條錯的建議浪費時間之後，對的那幾條也會被跳過。這一關與嚴重度排序是兩個獨立的軸。

### Convention Adherence

專案根目錄或子目錄存在 `CLAUDE.md` / `AGENTS.md` 時，把「違反這些規範」列為與架構 / 效能 / 安全並列的一個維度。

- **作用範圍**：一個檔案只受**它自己所在目錄與各層父目錄**的規範檔約束。`internal/note/new.go` 比對 `internal/note/CLAUDE.md`、`internal/CLAUDE.md`、根目錄的 `CLAUDE.md`；`page/CLAUDE.md` 與它無關。
- **引用原文**：每條違規逐字引用被違反的那一行，並標出規範檔路徑。引不出原文代表那不是規範違反，是個人偏好，刪掉。
- **消音優先**：程式碼旁有 `//nolint` / `# noqa` / 註解說明取捨時，視為作者已知並做過決定，不列——除非該取捨與規範原文直接衝突。

### No-Op 條件（同時滿足時不產檔）

1. `issue_counts` 的 critical / high / medium / low 皆為 0
2. 套用 Recommendation Principles 後，架構 / 效能 / 安全 / 規範遵循四段**皆無有效建議**（即都會寫「未觀察到需處理事項」）
3. 未觀察到超標 metric（見 `scripts/recommendation_principles.md` 例外欄位定義）

命中時的行為：

- **不**建立 `.doc/code-reviewer/` 目錄
- **不**寫入報告檔
- 對使用者輸出一行訊息：`無需處理：{language} 專案 {name}（{file_count} 檔 / {function_count} 函式）未觀察到可執行建議`
- 若使用者顯式指定 `OUTPUT_FILE`，視為強制產檔請求，仍寫入（內容可為「未觀察到需處理事項」的最小報告）

### Go-Specific Preprocessing

對每個非測試 `.go` 檔案自動執行 `gofmt -s -w`（失敗時靜默略過）。

---

## Step 1: Analyze Project

```bash
python3 ~/.claude/skills/code-reviewer/scripts/analyze_code.py /path/to/project
```

Output: JSON 包含：
- `language`: 主要語言
- `files`: 檔案列表
- `functions`: 函式資訊（名稱、簽章、行數、文件註解狀態）
- `issues`: 偵測到的問題
- `issue_counts`: 各嚴重度計數
- `metrics`: 程式碼指標（總行數、平均函式長度、最大巢狀深度）
- `dependencies`: 依賴套件

---

## Reference Documents

執行各階段時讀取對應參考檔：

| 階段 | 參考檔 | 用途 |
|---|---|---|
| Analyze / Evaluate | [`scripts/analysis_categories.md`](scripts/analysis_categories.md) | 偵測類別、嚴重度對照 |
| Generate | [`scripts/recommendation_principles.md`](scripts/recommendation_principles.md) | 建議產出的硬性規則與自我檢查 |
| Save | [`scripts/output_format.md`](scripts/output_format.md) | 報告結構範本與撰寫規則 |

---

## Validation Checklist

- [ ] 專案成功偵測語言並分析
- [ ] AST 工具鏈可用時使用 AST；否則降級為字串掃描並標註
- [ ] 每個問題包含檔案位置與建議
- [ ] 報告依嚴重度排序
- [ ] **已跑過 Validation Pass**；錨點確認不了的建議已移除而非降級
- [ ] 規範違反逐條可引用 `CLAUDE.md` / `AGENTS.md` 原文，且該規範檔位於該檔案的路徑或父路徑
- [ ] **已套用 `scripts/recommendation_principles.md` 自我檢查，無違反項目**
- [ ] **已檢查 No-Op 條件**；命中時跳過建立目錄與寫檔，僅輸出無需處理訊息
- [ ] 若產檔：`.doc/code-reviewer/` 目錄已建立（若不存在）
- [ ] 若產檔：報告寫入 `.doc/code-reviewer/{yyyy-MM-dd_HH-mm}.md`（或使用者指定的 `OUTPUT_FILE`）
