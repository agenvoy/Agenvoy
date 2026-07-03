# Skill 系統

Skill 是可載入的 markdown 指令包，能將 agent 切換至特定執行模式（例如 commit message 生成、code review、README 生成）。

## Skill 格式

Skill 是一個帶有 YAML frontmatter 作為 metadata 的 markdown 檔案：

```markdown
***
name: code-reviewer
description: Deep code review covering quality, security, and architecture
version: 1.0.0
***

You are now a strict code reviewer...
```

Frontmatter 的 `name` 是觸發關鍵字。內文會在 skill 啟用時 render 進 system prompt。

## 觸發路徑

### `/skill-name` slash command

在任何輸入前加上 `/<skill-name>`：

```
/code-reviewer review the diff in this PR
```

當 `MatchSkillCall` 命中時，agenvoy 會直接合成一組 `run_skill` `tool_call` 與對應的 `tool_result`（內含 skill 內文）進 `ToolHistories` — 與自然語言啟用路徑逐位元相同，保留 prefix cache。

若使用者帶入 args（`/code-reviewer review src/parser.go`），user message 會剝除 `/<skill-name>` prefix，只留下 args。若無 args，user message 保留字面的 `/<skill-name>`，讓 LLM 仍能看到啟用 context。

### 自然語言啟用

若 agent 在執行中判定某任務需要 skill，它會直接呼叫 `run_skill`。這是 LLM 發起的路徑，並使用相同的 render pipeline。

> Skill 啟用被設計為**工具呼叫**（lazy load），而非啟動時的預先挑選 — 這避免了為不需要 skill 的任務付出 skill 內文的 token。

### 單一對話中的多 skill

一個對話可依序啟用多個 skill。每次 `run_skill` 呼叫都會 append 至現有的指令堆疊；後續 skill 透過 system-prompt 區塊排序來增補或覆寫先前的 skill。

## User message 是 binding context

`skill_execution.md` Mandatory Principle #5：觸發 skill 的 user message 是 **binding context，而非雜訊**。LLM 將其視為使用者提供的參數／提示，並編織進輸出。

具體而言：

- SKILL.md 描述預設行為
- User message 覆寫或增補預設行為
- 「SKILL.md 中的步驟即命令」**並非**僵硬的單向解讀

範例：`/readme-generate private MIT` — SKILL.md 定義 README 結構；user message 指定 private mode + MIT license，兩者皆覆寫預設值。

## Skill 位置

Skill 位於 `extensions/skills/<name>/` 下：

```
extensions/skills/code-reviewer/
├── SKILL.md            # The skill definition (frontmatter + body)
└── ...                 # Optional supporting scripts / templates
```

Agenvoy 於啟動時掃描此目錄。System prompt 的 `## Skills` 區塊由 `skillTool.ListBlock` 填入，讓 LLM 得知有哪些 skill 可用。

## Skill 執行 prompt

執行迴圈由 `configs/prompts/skill_execution.md` 驅動，它承載每個 skill 都須遵守的規則（輸出紀律、工具名稱對映、mandatory principle）。

工具名稱對映範例：為 Anthropic SDK 撰寫的外部 skill 可能引用 `AskUserQuestion`；agenvoy 透過 `skill_execution.md` 中的 **Tool Name Mapping** 表自動將其對映至 `ask_user`。無需在 Go 端註冊 alias。
