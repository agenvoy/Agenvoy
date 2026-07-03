# Sandbox

每次 `run_command`、script 工具與 scheduler script 的執行都由 `go-pkg/sandbox` 包裹：

| 平台 | 機制 |
|---|---|
| Linux | bubblewrap（`bwrap`） |
| macOS | `sandbox-exec` |

Sandbox 限制：不可特權執行、受限的檔案系統寫入範圍、可配置的網路存取、可配置的 CPU/記憶體上限。

## 三個呼叫者，單一進入點

Sandbox 恰有三個呼叫者，全部直接呼叫 `sandbox.Wrap(ctx, binary, args, workDir, opt)`：

1. `run_command` 工具 — 任意使用者發出的 shell 命令
2. `toolAdapter/script/execute` — script 工具擴充（`script_*`）
3. `scheduler/script/script` — scheduler 驅動的 script

呼叫者與 `sandbox.Wrap` 之間沒有 wrapper 層。新增行為（例如新的資源上限）意味著貢獻至 `go-pkg/sandbox`，而非在 agenvoy 中加 shim。

## Policy 注入

三個 JSON 檔案定義 policy：

| 檔案 | 用途 |
|---|---|
| `configs/jsons/denied_map.json` | Sandbox 拒絕暴露的路徑 |
| `configs/jsons/exclude_list.json` | 從 listing/walking/searching 中排除的路徑 |
| `configs/jsons/white_list.json` | 允許的路徑 |

`cmd/app/main.go init()` 透過以下方式一次性注入 policy：

```go
sandbox.New(configs.DeniedMap)
filesystem.New(Policy{DeniedMap: ..., ExcludeList: ...})
```

`go-pkg/sandbox` 與 `go-pkg/filesystem` 都會自動對 policy 強制 `IsDenied` — 呼叫端無需檢查。

## 檔案系統寫入 guard

`go-pkg/filesystem` 的寫入 API（`WriteFile`、`WriteJSON`、`AppendText`、`CheckDir`）全部在內部強制 `IsDenied`。任何繞過 `go-pkg/filesystem`、直接透過 `os.WriteFile` 寫入的 agenvoy code 都會**逃脫 policy** — 這是禁止的。

`internal/filesystem` package 僅保留路徑計算與 domain wrapper（例如 `MCPPath`、`MCPSessionPath`）。它不重複 read/write 邏輯。

## 子程序 argv-only schema

`run_command` 只接受 `argv: string[]`（minItems 1）。它**不**接受帶自動 tokenize 的 `command: string`。此零解析做法移除了 agent 層的 shell-injection surface。

Shell 功能（pipe、redirect）需要 LLM 明確發出 `["sh", "-c", "cmd | pipe"]`。Allowlist 檢查會檢視 `argv[0]` 的 basename，且對 `sh -c` 也檢視內層第一個 token（透過 `strings.Fields(argv[2])[0]`）。Denylist 掃描則在 `strings.Join(argv, " ")` 上執行。

## 子程序 timeout

外部 CLI 呼叫（`invoke_external_agent`、`cross_review_with_external_agents`）有受環境變數控制的硬上限：

- `MAX_EXTERNAL_AGENT_TIMEOUT_MIN` — 預設 `10`，硬上限 `60`

Subagent 呼叫（`invoke_subagent`）含 slot-wait 時間，有：

- `MAX_SUBAGENT_TIMEOUT_MIN` — 預設 `10`，硬上限 `60`

這些防止失控的子程序執行無限期阻塞 parent。
