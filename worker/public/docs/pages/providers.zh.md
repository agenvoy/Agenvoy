# Providers

Agenvoy 在統一的 `Agent.Send()` 介面後支援十個 LLM provider。

## 支援清單

| Provider | Config name | 說明 |
|---|---|---|
| Anthropic Claude | `claude` | Messages API；預設啟用 parallel tool use |
| OpenAI | `openai` | Chat Completions / Responses API |
| OpenAI Codex | `codex` | OAuth 登入（用你的 ChatGPT / Codex 帳號，無需 API key）；SSE streaming；自動 prompt-cache key（`sha256(instructions)`） |
| Google Gemini | `gemini` | gemini-2.x / 3.x 家族 |
| GitHub Copilot | `copilot` | 需 GitHub OAuth（one-shot 登入流程） |
| Nvidia NIM | `nvidia` | Llama、Mistral 與其他 open-weight 託管 model |
| xAI Grok | `grok` | grok-4 / grok-3 家族含 `grok-code-fast-1`；non-streaming HTTP client |
| DeepSeek | `deepseek` | `deepseek-chat`（tool use）與 `deepseek-reasoner`（CoT，temperature 停用）；non-streaming HTTP client |
| OpenRouter | `openrouter` | Aggregator——透過單一 API key routing 至多 provider 的 200+ model |
| Compat | `compat` | 任何自訂 OpenAI-compatible endpoint |

> **Model discovery** — 自 v0.27.3 起，model 清單於 `agen model add` / TUI `/model global` 期間從各 provider 的 API 即時抓取。`configs/jsons/providors/` 底下的靜態 JSON catalog 已移除。

## Provider 設定

```bash
agen model add          # Interactive provider/model add
agen model remove       # Interactive provider/model remove
agen model list         # List registered models
agen model dispatcher   # Choose the dispatcher model
agen model reasoning    # Set dispatcher reasoning effort: low / medium / high / xhigh
```

憑證（API key、OAuth token）儲存於 OS keychain 的 `agenvoy` service 下，絕不以明文 JSON 存放。

## Dispatcher model

Dispatcher LLM 決定由哪個 worker model 處理各任務。它在 `Execute()` 進入 iteration loop 前透過 `SelectAgent()` 被呼叫，接收 user input 加上任何 matched skill 的提示。

透過 `agen model dispatcher`（model 選擇）與 `agen model reasoning`（reasoning effort）設定。

## Streaming

僅 `openaiCodex` 使用 SSE 進行 response streaming（`parseSSEStream` 依 `item_id` 累積 `argsBuf`）。其他 provider 每個 turn 一次性收到完整 response。

## Parallel tool calls

- **Claude Messages API** — parallel tool use 預設開啟
- **OpenAI Responses API** — `parallel_tool_calls=true` 保持開啟
- Agenvoy 執行 engine 仍序列化 commit（Pass 3），並遵守 per-tool concurrency marker

## Prompt caching

`openaiCodex/send.go` 計算 `sha256(instructions)` 並作為 `prompt_cache_key` 送出。Anthropic 與 OpenAI 皆在 >=1024 token 時支援自動 prefix caching，因此無需顯式 cache marker。

## 新增自訂 OpenAI-compatible endpoint

使用 `compat` provider type，並指向任何接受 OpenAI Chat Completions schema 的 endpoint。URL 慣例沿用 Zed：**輸入到 `/v1` 為止的 URL**（例如 `http://192.168.1.10:4000/v1`，Ollama 預設 `http://localhost:11434/v1`）。`compat/send.go` 只附加 `/chat/completions`。

```
/providor → name: VLLM
            URL:  http://192.168.1.10:4000/v1
            API key: <bearer token, or blank>
            Model: gemma3-27b-it          (becomes compat[VLLM]@gemma3-27b-it)
```

### 儲存拆分（URL vs key）

| 項目 | 位置 | API |
|---|---|---|
| URL | `~/.config/agenvoy/config.json` `compats[].URL` | `session.UpsertCompat` / `session.GetCompatURL` |
| API key | OS keychain | `keychain.Set("COMPAT_<NAME>_API_KEY", value)` |

`compat.New` 透過 `session.GetCompatURL(instanceName)` 讀取 URL。沒有 `COMPAT_<NAME>_URL` keychain key（刻意移除）。

### 已測試的 compat 目標

| 目標 | 可用 | 說明 |
|---|---|---|
| Ollama | 是 | 預設 `http://localhost:11434/v1` |
| LM Studio | 是 | |
| vLLM | 是 | tool use 需 `--enable-auto-tool-choice --tool-call-parser <name>` |
| llama.cpp server | 是 | |
| LiteLLM proxy | 是 | virtual key 作為 Bearer token |
| Groq / Together / DeepInfra / OpenRouter / Fireworks | 是 | |
| Azure OpenAI | 否 | 需 `api-key` header（非 `Bearer`）+ `?api-version=` query——不支援 |
| Reasoning-only model（o1、deepseek-r1、QwQ） | 部分 | compat 硬寫 `temperature: 0.2`；部分 server 回 422 |

## Send timeout（3 層）

Send 側 timeout 有三個獨立層，各捕捉不同的 failure mode：

| 層 | 值 | 捕捉 | 位置 |
|---|---|---|---|
| **Transport** `ResponseHeaderTimeout` | `15s`（僅 SSE） | Backend 在回傳 header 前卡住（健康的 SSE <1s 回傳；高負載 <=5s；15s 對齊 `ProbeTimeout`） | `openaiCodex/new.go::newHTTPClient()` |
| **`http.Client.Timeout`** | `10m` | 完整 request（header + body） | per-provider client |
| **`execute.go::AgentSendTimeout`** | env `AGENT_SEND_TIMEOUT_SECONDS`，預設 `600s` | 透過 `context.WithTimeout` 的 exec-layer 上限 | `internal/agents/exec/execute.go` |

所有 provider 用 `10m` client timeout。僅 codex SSE transport 加一層 `ResponseHeaderTimeout`；non-SSE provider 與 compat 省略之。

### HTTP client factory 拆分

| Provider 類別 | Factory | Config |
|---|---|---|
| Cloud non-SSE（claude / copilot / gemini / nvidia / openai / openrouter） | `provider.NewHTTPClient()` | `Timeout=10m` |
| Cloud SSE（openaiCodex） | `openaiCodex/new.go::newHTTPClient()` | `Timeout=10m` + `ResponseHeaderTimeout=15s` |
| Local / self-hosted（compat） | inline `&http.Client{Timeout: 10 * time.Minute}` | **無** `ResponseHeaderTimeout`——Ollama / vLLM / llama.cpp cold-start 可能在回 header 前 hold 30-90s；15s 會 100% false-positive |

Local compat 設計上**不**走 factory。Cold-start 容忍度對 self-hosted backend 不可妥協。

### Retry 語意

- `sendFailCount` 對 timeout/network error **無條件**累加（payload 未達 model；signature 比對無意義）
- 對 content-level error（parse 失敗、帶 body 的 4xx、garbage response），retry 為 sig-based——相同 payload signature 遞增計數；不同 signature 重置之
- `sendFailCount >= MaxRetry`（預設 3）觸發 MaxRetry-exhausted path，發出 `sendText` + `EventDone` 加上 branch-specific 訊息（timeout / context-length / generic）
- retry 期間（`sendFailCount < MaxRetry`）僅發 `slog.Warn`；不浮現 chat event（避免吵雜的「retrying 1/3, 2/3」洗頻——只有最終結果會抵達使用者）

OAuth device-code polling（`copilot/login.go`）每次 poll 用獨立的 `http.Client{Timeout: 30s}`——zero timeout 會讓 GitHub OAuth backend 掛住並鎖死整個登入流程。

***

> [!NOTE]
> 本文件由 Claude 讀完完整原始碼後自動生成。
