# Agenvoy Wiki

**Agenvoy** 讓 AI 真正為你所用——一個 always-on、model-agnostic、自我改進的個人 AI 助理 runtime。單一 Go daemon 對接多個 LLM provider（Claude / GPT / Gemini 自動 routing），透過 Telegram / Discord / TUI / browser 觸及你，並讓 agent 依你所需建構並持久化自己的 tool。

## 亮點

- **九個 LLM provider** — Claude、OpenAI、Codex（OAuth）、Gemini、GitHub Copilot、Nvidia NIM、DeepSeek、xAI Grok、Compat（任何 OpenAI-compatible endpoint，Zed 風格 `/v1` URL）
- **Dispatcher-based routing** — dispatcher LLM 將各任務 routing 至最適配的 worker（Claude 寫程式、Gemini 處理影片、GPT 做研究）
- **Three-pass 並行 tool dispatch** — read tool 並行扇出；write tool 為安全維持序列
- **多層 memory** — rolling summary（增量、timestamp-cursored）+ 16-message 近期 history + keyword/semantic 雙搜尋 + 90 天 TTL 的 cross-session error memory
- **原生文件 RAG** — KuraDB in-process 子進程（`list_rag` / `search_rag`），透過 TUI 的 `/feature kuradb` 啟用
- **Skill 系統** — 可載入的 markdown skill pack，由 `/skill-name` 或 `run_skill` 觸發；scheduler skill 隔離於 `~/.config/agenvoy/skills/scheduler/<short>-<hash8>/`
- **OS sandbox** — Linux bubblewrap / macOS sandbox-exec；tool 於隔離環境執行
- **MCP client** — stdio + HTTP/SSE；tool 自動注入為 `mcp__<server>__<tool>`
- **Chat 平台整合** — Telegram（6 位數 OTP 首次接觸驗證）+ Discord（原生 select menu / modal）；透過 `send_to_chatbot` 的 cross-session push
- **語音與附件** — `[SEND_VOICE:text]` 送至 Gemini TTS（OGG/OPUS）；inbound 附件存至 download dir
- **Sub-agent 與 external agent** — `invoke_subagent`（in-process）+ `invoke_external_agent`（codex / copilot / claude / gemini CLI）+ `cross_review_with_external_agents`（並行 review）
- **Scheduler** — cron / one-shot 任務、fsnotify hot-reload、output push 回 Telegram/Discord
- **Send-timeout 3 層系統** — Transport `ResponseHeaderTimeout=10s`、`Client.Timeout` 5m / 10m（SSE）、exec 層 `AgentSendTimeout` 600s 含 retry

## 原始碼

- Repository：[pardnchiu/Agenvoy](https://github.com/pardnchiu/Agenvoy)
