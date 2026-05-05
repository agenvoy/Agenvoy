# 步驟 5：輸出格式

確保 `.doc/version-generate/` 目錄存在：

```bash
mkdir -p .doc/version-generate
```

生成 `.doc/version-generate/NEW_VERSION.md`（例：`.doc/version-generate/v1.3.0.md`），依以下模板填入。

## 結構區塊

輸出檔必須包含兩個結構化區塊，順序與內容不得調換：

| 區塊 | 內容 | 對應段落 |
|---|---|---|
| **上段** | 發布者資訊與時間 | Frontmatter（`released_by` / `date` / `contributors` / `co_authors`） |
| **中段** | 變更紀錄與對應 commit hash | `## Summary` / `## ⚠️ Breaking Changes` / `## Changes` / `## Scope` |

## Summary 撰寫規範（強制）

### 硬性上限（超過任一條即整段重寫，不放行）

| 維度 | 上限 |
|---|---|
| 句數 | **1–3 句**（用 `.` / `。` 計算） |
| 英文字數 | **≤ 80 words** |
| 中文漢字 | **≤ 120 漢字**（不含 ASCII 與標點） |
| 整段「具體名稱」總數 | **≤ 2 個**（包含檔名、模組路徑、function／type／flag、CLI 指令、URL、量化規格） |

寫完後**必須**用下面的 Self-check 逐項回測；任何一項超標都不是「微調」，是**整段重寫**。

### 必須寫什麼（主軸三句結構）

Summary 是「主軸層」，回答「這版在做什麼方向」，**不是**「這版改了什麼」。後者是 `## Changes` 與 `## Scope` 的工作。

| 句位 | 任務 | 句型範例 |
|---|---|---|
| 第 1 句 | 本版核心主題（單一動詞 + 受詞 + 概念名詞） | `Plugs X into the Y ecosystem`／`Tightens permission gating across A and B` |
| 第 2 句（選用） | **獨立**第二方向，不是第 1 句的展開 | `Closes a long-standing gap in ...` |
| 第 3 句（選用） | 輔助／順帶（doc／chore） | `Detailed documentation moves to ...` |

### 絕對禁止（NEVER — 命中即不合格）

下列任一出現於 Summary 即重寫，不接受「只列一兩個應該還好」的妥協：

1. **逐 commit 列檔名／模組路徑** — 例：`internal/toolAdapter/mcp/`、`tools/register`、`cmd/app/main.go`
2. **列舉具體 API／function／type／flag** — 例：`invoke_subagent`、`AllowAll`、`MaxBytes`、`api_*`、`mcp__<server>__<tool>`
3. **列舉 CLI 指令／URL／環境變數** — 例：`agen mcp [list|add|remove]`、`https://...`、`MAX_HISTORY_MESSAGES`
4. **量化規格** — 例：「1 MiB」、「90 天 TTL」、「128 iterations」、「3 輪上限」
5. **括號塞清單** — 例：`the MCP client adapter (stdio + HTTP/SSE transports, global + per-session config merge, agen mcp CLI)`
6. **冒號／破折號／分號當小標** — 用 `... (...): X 與 Y；Z` 變相把 commit 內容串成一段
7. **中英版各自加料** — 中文不得比英文多任何具體名詞；發現中文比英文多訊息一律以英文為準回頭刪
8. **複述 Changes／Scope 的句子** — 同一句話只能出現一次，Summary 該抽象就抽象
9. **「Adds／Fixes／Updates A, B, C, D, E ...」式列舉** — 三個以上同性質動作必須抽象成單一概念

**為何嚴格：** Summary 一旦寫成清單，讀者必須先解析括號才能 grasp release 走向，等於把 `## Changes` 抄一遍 — 兩個區段語意重疊、`## Changes` 失去存在意義。Release note 的訊息密度應**遞減**：Summary（主軸）→ Changes（一句話／commit）→ Scope（路徑 + tag）。

### Self-check（寫完逐項勾，未通過不放行）

```
[ ] 英文 ≤ 80 words（用 `wc -w` 或一個一個數）
[ ] 中文 ≤ 120 漢字（去掉 ASCII 與標點再數）
[ ] 句數 1–3 句（數 `.` 與 `。`）
[ ] 「具體名稱」（檔名／模組／API／flag／指令／URL／量化規格）總數 ≤ 2
[ ] 第 1 句是「主題綱領」而非「細節枚舉」
[ ] 中英版主軸一致；中文不比英文多任何專名／量化
[ ] 沒有任何 NEVER 條目命中
```

### 對照範例

**❌ 反例**（命中 NEVER #1, #2, #3, #4, #5；110+ words；4 個括號塞清單）

> Adds the MCP client adapter (stdio + HTTP/SSE transports, global + per-session config merge, `agen mcp [list|add|remove]` CLI) so any MCP server can plug tools straight into the runtime under the `mcp__<server>__<tool>` namespace. Tightens `tools/register` to drop empty / duplicate names at the registry boundary, removes the `api_*` prefix exemption from the confirm gate, and caps each MCP tool result at 1 MiB so a single response cannot overrun provider limits. Detailed documentation moves out of `doc/` into GitHub Wiki; README cross-links now point at `https://github.com/agenvoy/Agenvoy/wiki/...`.

**為何錯：** 列了 5 個模組／API、3 個量化規格、寫死 URL、括號塞 4 段清單。讀者必須解析才能 grasp，已不是 summary 而是 changes 縮排版。

**✅ 正例**（43 words；3 句；0 個檔名／API／URL；具體名稱僅「MCP」「GitHub Wiki」共 2 個概念名詞 — 屬主題不屬細節）

> Plugs Agenvoy into the MCP ecosystem with a dual-transport client adapter that merges global and per-session config. Tightens the tool registry boundary and closes a confirm-gate gap that let user-defined API endpoints execute silently. Detailed documentation moves from the repo into GitHub Wiki.

**為何對：** 三個獨立主軸（接入 MCP／收斂 gating／文件搬家），全部用概念詞（boundary、gap、ecosystem）取代具體名稱。細節留給 `## Changes` 與 `## Scope`。

**✅ 正例**（取自 `v0.20.0` 收斂後）

> Session lifecycle gains a friendly-name layer with three routing paths. Runtime adds a process-singleton lock plus per-session status and rotating action log. Internal utilities upgraded.

## Scope 撰寫規範（強制）

`## Scope` 段內容**一律英文**，**不附中文翻譯區塊**。每行格式：

```
- `<path>` — <TAG1>, <TAG2>, ...
```

`<path>` 為原始檔／目錄路徑（保留斜線與副檔名），`<TAG>` 用大寫 commit tag（FEAT／FIX／UPDATE／SECURITY 等）。**禁止**：
- 中文補充說明
- 中文標點（全形括號／頓號／全形斜線「／」）—— 多檔列舉一律 ASCII `, ` 分隔
- 完整檔名清單（同目錄多檔合併為單行）
- 與 `## Changes` 重複的句子

範例（取自 `v0.20.0`）：
```
- `internal/session/` — FEAT, ADD (`bot.go`, `match.go`, `status.go`)
- `internal/agents/exec/` — FEAT (`run.go`, `getSession.go`)
```

附帶檔名清單可用，但限同模組內 ≤ 5 檔；超過則省略改寫成「`internal/foo/` — FEAT (multiple files)」。

## 範本

````markdown
---
version: v1.3.0
previous: v1.2.3
date: 2026-04-25
type: minor
breaking: false
released_by:
  name: Pardn Chiu
  email: chiuchingwei@gmail.com
  github: "@pardnchiu"
contributors:
  - Pardn Chiu <chiuchingwei@gmail.com>
co_authors:
  - Claude <noreply@anthropic.com>
compare: https://github.com/pardnchiu/skill-version-generate/compare/v1.2.3...v1.3.0
---

> v1.2.3 -> v1.3.0

## Summary

One short paragraph (1–3 sentences, ≤ 80 words / ≤ 120 漢字) capturing the **theme** of the release. State what changed at the conceptual level, not commit-by-commit.
<details>
<summary>翻譯</summary>
版本主軸的簡述（1–3 句、≤ 120 漢字）。陳述本次概念層的變動，**不要**逐 commit 列出。
</details>

## ⚠️ Breaking Changes

> 僅當 `breaking: true` 時出現；非破壞性版本**省略整段**。

### `AuthClient.Login()` signature changed

**Before:**
```go
Login(username, password string) error
```

**After:**
```go
Login(ctx context.Context, creds Credentials) error
```

**Migration:**
```go
// Old
client.Login("alice", "secret")
// New
client.Login(ctx, Credentials{User: "alice", Pass: "secret"})
```

<details>
<summary>翻譯</summary>

`AuthClient.Login()` 簽章變更，需改傳 `context.Context` 與 `Credentials` 結構體。

</details>

## Changes

### FEAT
- Add OAuth2 login flow (#142, @alice) [a3f9c2d]

<details>
<summary>翻譯</summary>

- 新增 OAuth2 登入流程

</details>

### FIX
- Fix nil pointer in token refresh (#145, @bob) [b8e4d1f]

<details>
<summary>翻譯</summary>

- 修正 token refresh 時的 nil pointer

</details>

### REFACTOR
- Extract auth middleware into dedicated package (#143, @alice) [c2a7e9b]

<details>
<summary>翻譯</summary>

- 將 auth middleware 抽取至獨立 package

</details>

### PERF
- Cache JWK endpoint responses (#147, @alice) [d4f1c8e]

<details>
<summary>翻譯</summary>

- 快取 JWK endpoint 回應

</details>

### SECURITY
- Patch CVE-2026-1234 in session cookie handling (#150, @bob) [e9c3b2a]

<details>
<summary>翻譯</summary>

- 修補 session cookie 處理的 CVE-2026-1234

</details>

### DOC
- Document OAuth2 setup in README (#144, @alice) [f1a8d5c]

<details>
<summary>翻譯</summary>

- 於 README 新增 OAuth2 設定說明

</details>

### CHORE
- Bump golang.org/x/oauth2 to v0.25.0 (#146, @dependabot) [a7b3f2e]

<details>
<summary>翻譯</summary>

- 升級 golang.org/x/oauth2 至 v0.25.0

</details>

## Scope

- `pkg/auth/` — FEAT, REFACTOR, SECURITY
- `pkg/cache/` — PERF
- `docs/` — DOC

***

Generated by [SKILL](https://github.com/pardnchiu/skill-version-generate)
````

## Frontmatter 欄位規格

| 欄位 | 來源 | 必要 |
|---|---|---|
| `version` | `NEW_VERSION` | ✅ |
| `previous` | `LATEST_TAG` | ✅ |
| `date` | 今日日期（`YYYY-MM-DD`） | ✅ |
| `type` | `major` / `minor` / `patch` / `none` | ✅ |
| `breaking` | `true` / `false` | ✅ |
| `released_by.name` | `$RELEASER_NAME` | ✅ |
| `released_by.email` | `$RELEASER_EMAIL` | ✅ |
| `released_by.github` | `$RELEASER_GITHUB` | 缺失則省略該行 |
| `contributors` | 區間去重作者清單（`Name <email>`） | ✅ |
| `co_authors` | `Co-authored-by` trailer 去重清單 | 空則省略 |
| `compare` | `$REMOTE_URL/compare/$LATEST_TAG...$NEW_VERSION` | 非 GitHub remote 則省略 |

## 條目格式

每條變更尾綴 `(#PR, @author) [short_hash]`，缺失欄位省略對應括號：

- 完整：`Add OAuth2 login flow (#142, @alice) [a3f9c2d]`
- 無 PR：`Add OAuth2 login flow (@alice) [a3f9c2d]`
- 無作者 handle：`Add OAuth2 login flow (#142) [a3f9c2d]`
- 最小：`Add OAuth2 login flow [a3f9c2d]`
