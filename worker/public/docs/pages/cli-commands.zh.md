# CLI 指令

## 頂層分派

`agen` 解析 `os.Args[1]` 並分派到下列子指令之一；不帶子指令執行時會 attach TUI（若無 daemon 執行中則 fork-exec 一個 daemon）。

```bash
agen                                           # Attach TUI; spawn daemon if not running
agen model   {add|remove|list|dispatcher|reasoning}
agen session {new|switch|config} [name]
agen mcp     {list|add|remove}
agen cli     <input...>                        # one-shot, requires per-tool confirm
agen run     <input...>                        # one-shot, auto-approves all tools
agen stop                                      # Stop the running daemon
agen update                                    # Download latest release & rebuild
```

### `agen model`

| 子指令 | 行為 |
|---|---|
| `add` | 互動式 provider/model 新增（將憑證寫入 keychain） |
| `remove`（別名 `rm`） | 互動式 provider/model 移除 |
| `list` | 列出已註冊的 model |
| `dispatcher` | 選擇 dispatcher model |
| `reasoning` | 設定 dispatcher reasoning effort：`low` / `medium` / `high` / `xhigh` |

### `agen session`

| 子指令 | 行為 |
|---|---|
| `new <name>` | 建立 `cli-<uuid>` session，寫入其 `bot.md`（frontmatter `name=<name>`），切換 primary pointer |
| `switch [name]` | 切換 primary pointer；不帶 `name` 時開啟互動式選單，並高亮當前 session（Enter = 維持不變） |
| `config [name]` | 以 `$EDITOR` 開啟目標 session 的 `bot.md`；不帶 `name` 時開啟選單 |

### `agen mcp`

| 子指令 | 行為 |
|---|---|
| `list` | 列出所有已設定的 MCP server（global + per-session） |
| `add` | 互動式新增 — 名稱、transport（Local stdio / Remote HTTP）、欄位、scope（Global / 選一個 session） |
| `remove` | 互動式移除，含 scope 標籤 |

### `agen stop`

對執行中的 daemon 送 SIGTERM（5 秒緩衝期後改送 SIGKILL）；清除 `~/.config/agenvoy/runtime.uid`。若無存活的 daemon，印出 `No daemon running.` 並以 0 退出。

### `agen update`

一律覆寫式更新至最新 release。下載 `https://agenvoy.com/scripts/update.sh` 至 `/tmp/agenvoy-update-*.sh` 檔案，透過 `bash` 執行，並在完成時移除暫存檔（SIGINT/SIGTERM 亦會清理）。該腳本將最新 tag clone 至 `mktemp -d "${TMPDIR:-/tmp}/agenvoy-update.XXXXXX"`，執行 `make build`，並印出一個指向 `agen` 供下次啟動的摘要框。替換後 daemon 仍持有舊 inode — 執行 `agen stop` 並重新 attach 以接上新 build。

## `make` 捷徑

```bash
make build                      # Compile and install to /usr/local/bin/agen
make app                        # Full stack (TUI + Discord + Telegram + REST API)
make stop                       # Stop the running daemon
make update                     # = agen update
make cli <input...>             # agen cli <input...>
make run <input...>             # agen run <input...>
make model   [add|remove|list|dispatcher|reasoning]
make session [new|switch|config] [name]
make mcp     [list|add|remove]
```

## 輸入前綴

`exec.Run()` 中的解析順序（僅限 CLI / TUI / Telegram — Discord 與 HTTP 不解析 `:name`）：

1. **`:name`** — session override（一次性路由，不改動 primary pointer）
2. **`MatchExternal`** — 外部 CLI agent 分派（`/claude`、`/codex` 等）
3. **`MatchSkillCall`** — skill 啟用（`/<skill-name>`）

### `:name` session override

```bash
make cli ":ship-v0.20 /commit-generate"
```

可與 skill 及外部 agent 組合 — 順序由左至右解析（先 `:bot`、再 external、再 skill、最後 execute）。

### 外部 CLI 前綴

| 前綴 | 模式 | 底層 flag |
|---|---|---|
| `/claude` | 唯讀 | `claude -p --disallowedTools=Edit,Write,NotebookEdit` |
| `/claude-allow` | 寫入 | `claude -p --permission-mode acceptEdits` |
| `/codex` | 唯讀 | codex CLI（預設 sandbox）+ `--output-last-message` + `--skip-git-repo-check` |
| `/codex-allow` | 寫入 | codex CLI `--dangerously-bypass-approvals-and-sandbox` |
| `/gh` 或 `/copilot` | 唯讀 | `gh copilot -s`（無寫入變體） |
| `/gemini` | 唯讀 | `gemini --approval-mode plan --skip-trust` |
| `/gemini-allow` | 寫入 | `gemini --yolo --skip-trust` |

### Skill 前綴

任何註冊於 `extensions/skills/<name>/` 下的 skill 皆由 `/<name>` 觸發：

```bash
make cli "/commit-generate"
make cli "/readme-generate private MIT"
```

`/<skill-name>` 之後的 user message 參數會作為 binding context 傳入。

## Auto 模式

按 **Shift+Tab** 切換 auto 模式。當前模式顯示於 TUI 左下角：

- `[safe]`（預設） — tool call 執行前需 user 確認
- `[auto]` — 所有 tool call 自動核准（`allowAll = true`）；sandbox 與 validator 仍然生效

Auto 模式為 session-local，TUI 重啟時重置。亦可於啟動時透過 `agen --allow-all` 設定。

## 環境變數

完整清單見 Configuration 頁面。
