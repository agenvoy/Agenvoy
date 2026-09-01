# 步驟 2：收集變更（淨差異優先）

## 2.0 核心判準：條目 = 淨差異，不是 commit

**`git diff $LATEST_TAG..HEAD` 是條目存在與否的唯一依據。** commit log 只提供屬性（hash／author／PR／變更意圖），**不決定條目要不要出現**。

Changelog 回答的是「`$LATEST_TAG` 與 `HEAD` 兩個狀態差在哪」，不是「這段期間發生過哪些事」。開發過程中的中間態、走錯又走回來的路徑，對讀者是雜訊。

| 區間內發生 | `git diff` 淨結果 | 處理 |
|---|---|---|
| 新增檔案 → 又刪除 | 無此檔 | **不列**（連 REMOVE 都不列） |
| namespace A → B → A | 無差異 | **不列** |
| 值 X → Y → X | 無差異 | **不列** |
| 同一主題跨 3 個 commit | 有淨差異 | **合併為 1 條**，描述最終狀態 |
| feat 後被 fix 修正 | 有淨差異 | 只列 FEAT（描述修正後的形態），不列該 fix |
| 純還原他人上一版功能 | 有淨差異（該功能消失） | 列 REMOVE，描述結果 |

**為何：** 歷史事故（ToriiDB v0.5.2）——區間內 Cloudflare Worker 站台被新增後又整包刪除、module namespace 從 `pardnchiu` 遷到 `agenvoy` 再遷回 `pardnchiu`，兩者淨差異皆為零；skill 仍依 commit log 產出 `FEAT: Add Cloudflare Worker landing site` 與 `REFACTOR!: Migrate namespace` 條目，並為後者要求 Migration 指引。讀者拿到的是「發生過但現在不存在」的假變更。

## 2.1 淨差異收集（先於 commit log）

```bash
# 檔案層級淨變更：A=新增 M=修改 D=刪除 R=重新命名
git diff --name-status $LATEST_TAG..HEAD

# 規模概覽，用於判斷主軸與 Scope 分組
git diff --stat $LATEST_TAG..HEAD

# 針對候選主題實際讀 diff，確認變更性質與最終形態
git diff $LATEST_TAG..HEAD -- <path>
```

先由 `--name-status` 與 `--stat` 切出「本版真正改了哪些區塊」，再進 2.2 用 commit log 為每個區塊找對應的 hash 與意圖描述。

## 2.2 commit 資料收集（屬性來源）

```bash
# 完整 commit 資料：hash|subject|author_name|author_email|body
# 使用 \x1f (Unit Separator) 分隔欄位, \x1e (Record Separator) 分隔記錄
git log $LATEST_TAG..HEAD --format='%H%x1f%s%x1f%an%x1f%ae%x1f%b%x1e'

# 區間內去重作者清單
git log $LATEST_TAG..HEAD --format='%an <%ae>' | sort -u

# Co-author trailer（pair programming / AI 協作）
git log $LATEST_TAG..HEAD --format='%(trailers:key=Co-authored-by,valueonly)' \
    | grep -v '^$' | sort -u
```

Contributors／co_authors 取**區間全部 commit**（含被抵銷的），署名不因變更被抵銷而消失。

## 2.3 對帳（強制，產出條目前執行）

逐一比對 commit 清單與淨差異，任一 commit 落入下列狀態即**丟棄**，不進 `## Changes`：

```
FOR EACH commit IN $LATEST_TAG..HEAD:
    touched = git diff-tree --no-commit-id --name-only -r <commit>
    IF touched 之檔案在 git diff $LATEST_TAG..HEAD --name-status 中全數缺席:
        DROP（新增後被刪除／改後被改回）
    IF 該 commit 的語意變更在最終 diff 中已不存在（例：遷移後遷回）:
        DROP
```

多個 commit 對應同一淨差異時**合併為單一條目**，`[hash]` 欄位列出全部 short hash（逗號分隔），描述寫**最終狀態**而非過程。

### hash 來源保留（強制）

條目描述改用淨差異，但**溯源不得因此丟失**：每條目必須列出所有對該淨差異有貢獻的 commit，包含中間過程的 commit（只要其改動有一部分留在最終 diff 中）。

```bash
# 取某路徑在區間內的全部 commit，作為該條目的 hash 來源
git log $LATEST_TAG..HEAD --format='%h' -- <path>
```

由 2.7 fallback（無 Conventional 格式）產生的條目同樣適用——先由 diff 定條目，再用上述指令補回 hash。

淨差異為零而整組丟棄者無條目可掛 hash，其軌跡由 frontmatter `compare` 連結與 git 歷史本身承載，不在 changelog 內另立區段記錄。

## 2.4 Conventional Commits 解析規則

通過 2.3 對帳的 commit，其 subject 依下列正則比對取語意：

```
^(feat|fix|docs|style|refactor|perf|test|chore|build|ci|revert|security)(\([^)]+\))?(!)?: (.+)$
```

- **Group 1**（type）→ 映射到內部標籤
- **Group 2**（scope，選填）→ 寫入 `Scope` 區段
- **Group 3**（`!`）→ 標記 BREAKING
- **Group 4**（description）→ 條目內文起點；**與淨 diff 不符時以 diff 為準**（commit message 描述的是當下意圖，可能已被後續 commit 推翻）

### 型別映射

| Conventional | 內部標籤 |
|---|---|
| `feat` | FEAT |
| `fix` | FIX |
| `perf` | PERF |
| `refactor` | REFACTOR |
| `docs` | DOC |
| `test` | TEST |
| `style` | STYLE |
| `chore` / `build` / `ci` | CHORE |
| `security` | SECURITY |
| `revert` | 依 2.0 表判定：被還原對象若也在本區間內 → 兩者皆 DROP；還原的是既有版本功能 → REMOVE |
| 任一 type + `!` 後綴 | 追加 BREAKING（須通過 2.5 淨檢查） |

## 2.5 BREAKING 偵測（淨差異驗證）

候選來源兩種：

1. Subject 含 `!` 後綴（例：`feat!: ...`）
2. commit body 含 `BREAKING CHANGE:` trailer

**命中後必須驗證該破壞在 `$LATEST_TAG..HEAD` 的淨 diff 中仍然存在**：對外介面（module path／export 簽章／CLI 參數／API route／設定欄位）與 `$LATEST_TAG` 相比確實不相容才成立。

| 驗證結果 | 處理 |
|---|---|
| 淨 diff 仍不相容 | 加入 BREAKING 區段，**強制觸發 Migration 產出**（缺失則中止） |
| 淨 diff 無差異（改出去又改回來） | **不是 BREAKING**，整條 DROP，不寫 Migration |

**為何：** 對 `$LATEST_TAG` 的使用者而言，唯一可見的是兩個 tag 之間的介面差異。區間內部曾經不相容但已還原，升級時不需要任何動作，發 Migration 指引等於要求使用者執行一段不存在的遷移。

## 2.6 PR 編號擷取

squash-merge 的 subject 通常尾綴 `(#123)`：

```
\(#([0-9]+)\)$
```

命中則記錄 PR 編號，輸出格式 `描述 (#123, @author) [a3f9c2d]`。

## 2.7 Fallback：無 Conventional 格式時

若 subject 未命中 2.4 正則，直接由 2.1 的淨 diff 判讀變更性質（LLM 讀 diff 內容），依分類規則歸類。此路徑本就以 diff 為準，無需額外對帳。
