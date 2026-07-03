# 給開發者

## 差異化重點

- **基於 dispatcher 的智慧路由** — 由 dispatcher model 將每個任務路由到最適合的 worker（Claude 負責寫程式、Gemini 負責影片、GPT 負責研究），而非強迫單一 model 處理所有事情。
- **會自行建立並持久化工具的 agent** — 當缺少某個工具時，agent 會在 `extensions/` 寫入一支 script 或 API，並在下次執行時將其載入為原生工具；同時也支援 MCP server。
- **橫跨所有 channel 的單一 runtime** — Telegram、Discord、TUI、Web 與 cron 全都掛接到同一個 daemon；session、記憶與工具集皆為共用，而非每個介面各自重建。

## Agenvoy vs 主流產品：完整比較

### 1. 總覽

| | **Agenvoy** | **OpenClaw** | **Hermes Agent** | **Claude Code** | **Codex CLI** | **Gemini CLI** |
|--|--|--|--|--|--|--|
| **語言** | Go | TypeScript | Python | TypeScript | Rust + TypeScript | TypeScript |
| **授權** | Apache 2.0 | MIT | MIT | Proprietary | Apache 2.0 | Apache 2.0 |
| **作者** | 個人（pardnchiu） | 社群 | NousResearch | Anthropic | OpenAI | Google |
| **主要用途** | 多平台 AI Agent 框架 | 多平台 AI Agent | 多平台 AI Agent | 終端機程式輔助 | 終端機程式輔助 | 終端機程式輔助 |
| **架構** | Daemon + TUI + Chat | Daemon + TUI + Chat | Daemon + TUI + Chat | CLI session | CLI session | CLI session |

***

### 2. AI Provider 支援

| | **Agenvoy** | **OpenClaw** | **Hermes Agent** | **Claude Code** | **Codex CLI** | **Gemini CLI** |
|--|--|--|--|--|--|--|
| Claude | ✅ | ✅ | ✅ | ✅ 僅此 | ❌ | ❌ |
| OpenAI / GPT | ✅ | ✅ | ✅ | ❌ | ✅ 僅此 | ❌ |
| Gemini | ✅ | ✅ | ✅ | ❌ | ❌ | ✅ 僅此 |
| Codex (OpenAI OAuth) | ✅ | ✅ | ✅ | ❌ | ✅ | ❌ |
| GitHub Copilot | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ |
| Nvidia NIM | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ |
| OpenAI-compat | ✅ | ✅ Ollama/LM Studio | ✅ OpenRouter 200+ | ❌ | ❌ | ❌ |
| DeepSeek | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ |
| xAI (Grok) | ✅ API key | ✅ | ✅ OAuth + API key | ❌ | ❌ | ❌ |
| Mistral | ❌ | ✅ | ⚠️ 透過 OpenRouter（無專屬） | ❌ | ❌ | ❌ |
| Dispatcher 路由 | ✅ 專屬 dispatcher model | ❌ | ❌ | ❌ | ❌ | ❌ |

***

### 3. Runtime 與前端

| | **Agenvoy** | **OpenClaw** | **Hermes Agent** | **Claude Code** | **Codex CLI** | **Gemini CLI** |
|--|--|--|--|--|--|--|
| TUI | ✅ bubbletea | ✅ `openclaw tui` | ✅ React Ink | ✅ ink | ✅ | ✅ |
| CLI | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| HTTP API / Web UI | ✅ gin | ✅ dashboard / webchat | ✅ Web Dashboard | ❌ | ❌ | ❌ |
| Daemon 模式 | ✅ 原生 `--daemon` | ✅ systemd/launchd | ✅ gateway daemon | ❌ | ❌ | ❌ |
| Session Canvas (HTML+SSE) | ✅ `render_page` | ❌ | ❌ | ❌ | ❌ | ❌ |
| 具名 session | ✅ | ⚠️ workspaces / per-agent session | ✅ session picker | ❌ | ❌ | ❌ |

***

### 4. 聊天平台整合

| | **Agenvoy** | **OpenClaw** | **Hermes Agent** | **Claude Code** | **Codex CLI** | **Gemini CLI** |
|--|--|--|--|--|--|--|
| Telegram | ✅ 原生 daemon | ✅ 原生 daemon | ✅ 原生 daemon | ⚠️ Channels MCP（需 active session） | ❌ | ❌ |
| Discord | ✅ 原生 daemon | ✅ 原生 daemon | ✅ 原生 daemon | ⚠️ Channels MCP（需 active session） | ❌ | ❌ |
| iMessage | ❌ | ✅ BlueBubbles | ✅ BlueBubbles | ⚠️ Channels MCP（僅 macOS） | ❌ | ❌ |
| LINE | ⚠️ alpha（linebot branch） | ✅ | ✅ | ❌ | ❌ | ❌ |
| WhatsApp / Slack | ❌ | ✅ 24+ 平台 | ✅ 24+ 平台 | ❌ | ❌ | ❌ |
| 常駐接收（免 session） | ✅ daemon | ✅ | ✅ | ❌ | ❌ | ❌ |
| 跨 session 傳送（任一 session 送至 chat） | ✅ `send_to_chatbot` | ❌ | ⚠️ `send_message` 工具 | ❌ | ❌ | ❌ |
| 首次接觸驗證 | ✅ 6 位數 OTP（crypto/rand） | ✅ pairing code（dmPolicy: pairing） | ✅ pairing code（`gateway/pairing.py`） | ❌ | ❌ | ❌ |
| 原生平台 UI（按鈕 / 選單 / modal） | ✅ inline keyboard / select menu / modal | ⚠️ 文字型選項 | ⚠️ 文字型選項 | ❌ | ❌ | ❌ |

> **平台層**：Agenvoy 的 Telegram 與 Discord 整合建構於 pardnchiu/go-bot 之上，獨立維護且開源。go-bot 封裝了 bot 協定細節 — Agenvoy 只實作業務邏輯。

> **關鍵差異**：Claude Code Channels 需要 active session。OpenClaw 與 Hermes 有 daemon，但聊天內的確認為文字型。Agenvoy 使用原生平台 UI — Telegram inline keyboard 與 Discord select menu / modal。Agenvoy 的跨 session 傳送讓任何 session 類型（CLI/TUI/HTTP/排程 script）都能推送至特定的 Telegram/Discord chat — 競品僅部分暴露此能力。

***

### 5. Telegram 功能比較

| 功能 | **Agenvoy** | **OpenClaw** | **Hermes Agent** | **Claude Code Channels** |
|---------|-------------|-------------|------------------|--------------------------|
| 文字回覆 | ✅ | ✅ | ✅ | ✅ |
| 語音回覆（TTS） | ✅ Gemini TTS | ✅ ElevenLabs/Hume | ✅ Edge TTS/ElevenLabs | ❌ |
| 傳送檔案 | ✅ `[SEND_FILE:]` | ✅ | ✅ | ❌ |
| 接收附件 | ✅ photo/doc/voice/video | ✅ | ✅ | ❌ |
| 語音轉文字（STT） | ✅ Gemini，14 種格式 | ✅ Whisper/Gemini | ✅ faster-whisper（本地） | ❌ |
| 工具確認（互動式） | ✅ 原生 inline keyboard | ⚠️ 文字核可提示 | ⚠️ 文字型選項 | ❌ |
| ask_user（picker） | ✅ 原生 button/modal | ⚠️ `/models` picker | ⚠️ 文字型選項，最多 4 個 | ❌ |
| 格式參考（lazy-load 工具） | ✅ `format_chatbot` | ❌ | ❌ | ❌ |
| Scheduler 輸出推送 | ✅ | ✅ | ✅ | ❌ |
| 跨 session 推送（來自任一 session） | ✅ `send_to_chatbot` | ❌ | ⚠️ `send_message` 工具 | ❌ |
| 離線接收（daemon） | ✅ | ✅ | ✅ | ❌ |

***

### 6. Discord 功能比較

| 功能 | **Agenvoy** | **OpenClaw** | **Hermes Agent** | **Claude Code Channels** |
|---------|-------------|-------------|------------------|--------------------------|
| 文字回覆 | ✅ | ✅ | ✅ | ✅ |
| 語音回覆（TTS） | ✅ Gemini TTS | ✅ | ✅ | ❌ |
| 傳送檔案 | ✅ 批次 10/訊息 | ✅ | ✅ | ❌ |
| 接收附件 | ✅ photo/doc/voice/video | ✅ | ✅ | ❌ |
| 工具確認（互動式） | ✅ select menu 按鈕 | ✅ `/model` picker | ⚠️ 文字型選項 | ❌ |
| ask_user（modal） | ✅ select/multi-select/modal | ⚠️ 受限 | ⚠️ 文字型選項 | ❌ |
| 格式參考（lazy-load 工具） | ✅ `format_chatbot` | ❌ | ❌ | ❌ |
| Guild mention 防護 | ✅ | ✅ | ✅ | ❌ |
| 感知 Discord Markdown | ✅ 完整規格作為 lazy-load 工具 | ⚠️ 部分 | ⚠️ 部分 | ❌ |
| 感知字元上限 | ✅ prompt 內 1600 字元硬上限 | ❌ | ❌ | ❌ |
| 跨 session 推送（來自任一 session） | ✅ `send_to_chatbot` | ❌ | ⚠️ `send_message` 工具 | ❌ |

***

### 7. Scheduler

| | **Agenvoy** | **OpenClaw** | **Hermes Agent** | **Claude Code** | **Codex CLI** | **Gemini CLI** |
|--|--|--|--|--|--|--|
| Cron jobs | ✅ SKILL.md + cron | ✅ 內建 | ✅ 內建 | ✅ 雲端輔助 cron/task | ❌ | ❌ |
| 一次性任務 | ✅ | ✅ `at` 格式 | ✅ 自然語言 | ✅ 雲端輔助 | ❌ | ❌ |
| TUI CRUD | ✅ | ✅ `openclaw cron` | ✅ `cronjob` 工具 | ❌ | ❌ | ❌ |
| fsnotify hot-reload | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ |
| 推送輸出至 Telegram/Discord | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ |
| AI 工具管理（新增/列出/移除） | ✅ | ❌ | ✅ `cronjob` 工具 | ❌ | ❌ | ❌ |
| 本地執行（免雲端） | ✅ | ✅ | ✅ | ❌ 依賴雲端 | ❌ | ❌ |

> **Scheduler 層**：建構於 pardnchiu/go-scheduler 之上，一個自維護的生態系套件，提供 cron 表達式解析、一次性任務、fsnotify hot-reload，以及完整的輸出路由回聊天平台。

***

### 8. 工具生態系

| | **Agenvoy** | **OpenClaw** | **Hermes Agent** | **Claude Code** | **Codex CLI** | **Gemini CLI** |
|--|--|--|--|--|--|--|
| MCP 支援 | ✅ client | ✅ client | ✅ client + server | ✅ client | ❌ | ✅ client |
| 自訂工具（自動探索） | ✅ AI 生成 | ❌ | ✅ 自動建立 skill | ❌ | ❌ | ❌ |
| API 工具探索（先 search-api 再 add） | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ |
| 工具 registry（跨機器發布 + 安裝） | ✅ pkg.agenvoy.com（Cloudflare Worker + R2 + D1，email 驗證 + 降版防護） | ⚠️ ClawHub（skills + plugins） | ⚠️ agentskills.io（僅 skills） | ❌ | ❌ | ❌ |
| Skill 系統 | ✅ SKILL.md lazy-load | ✅ SKILL.md 5400+ 社群 | ✅ SKILL.md agentskills.io | ✅ CLAUDE.md | ❌ | ❌ |
| Skill 自我改進（失敗時自動修復） | ✅ trace 後重寫再 auto-commit | ❌ | ✅ | ❌ | ❌ | ❌ |
| 格式參考作為 lazy-load 工具 | ✅ `format_chatbot` | ❌ | ❌ | ❌ | ❌ | ❌ |
| 文件 RAG（外部知識庫） | ✅ KuraDB（in-process 向量 + 語意/關鍵字） | ❌（僅對話記憶） | ❌（僅對話記憶） | ❌ | ❌ | ❌ |
| 媒體轉錄 STT | ✅ Gemini，14 種格式 | ✅ Whisper/Gemini | ✅ faster-whisper（本地） | ❌ | ❌ | ❌ |
| TTS 語音輸出 | ✅ Gemini TTS | ✅ ElevenLabs/Hume/MS | ✅ Edge TTS/ElevenLabs/OpenAI | ❌ | ❌ | ❌ |
| Computer use / 瀏覽器 | ✅ go-rod + Playwright MCP | ✅ Chrome CDP | ✅ browser CDP + computer-use（cua-driver） | ✅ beta | ❌ | ❌ |

> **工具 sandbox 架構**：建構於 pardnchiu/go-faas（Function as a Service）之上。每個 AI 生成的工具都以獨立的 function 單元執行，擁有自己的生命週期與安全邊界。是所有比較產品中唯一採 FaaS 層級 sandbox 設計者。

***

### 9. 記憶系統

| | **Agenvoy** | **OpenClaw** | **Hermes Agent** | **Claude Code** | **Codex CLI** | **Gemini CLI** |
|--|--|--|--|--|--|--|
| 指令檔系統 | ✅ SKILL.md | ✅ SKILL.md | ✅ SKILL.md | ✅ CLAUDE.md | ❌ | ❌ |
| 對話歷史搜尋 | ✅ 三層：context + ToriiDB 向量 + SQLite FTS5 | ✅ LanceDB 向量 | ✅ SQLite FTS5 | ❌ | ❌ | ❌ |
| 外部文件 RAG（原生、in-process） | ✅ KuraDB（語意 + 關鍵字，OpenAI embeddings） | ❌（使用 MCP） | ❌（使用 MCP） | ❌ | ❌ | ❌ |
| 錯誤記憶 | ✅ ToriiDB | ❌ | ❌ | ❌ | ❌ | ❌ |
| Action log | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ |
| 長期持久記憶 | ✅ SQLite 全文歸檔（雙寫，永不遺失資料） | ✅ Wiki 式 MEMORY.md | ✅ MEMORY.md + USER.md | ⚠️ CLAUDE.md 手動 | ❌ | ❌ |
| 跨 session 記憶 | ⚠️ 預設 session 隔離，可擴充 | ✅ 內建跨 session | ✅ 內建跨 session | ⚠️ 預設 session 隔離，可擴充 | ⚠️ session 隔離 | ⚠️ session 隔離 |

> **三層對話記憶**：(1) **Context** — 最新 16 則訊息載入 LLM context + 週期性 summary；(2) **ToriiDB** — 自研的嵌入式向量資料庫（pardnchiu/ToriiDB），對近期對話做語意相似度搜尋；(3) **SQLite FTS5** — 透過 pardnchiu/go-sqlkit 的全文歸檔，每則訊息雙寫，即使歷史被壓縮也永不遺失資料。

***

### 10. 依賴與部署

| | **Agenvoy** | **OpenClaw** | **Hermes Agent** | **Claude Code** | **Codex CLI** | **Gemini CLI** |
|--|--|--|--|--|--|--|
| 直接外部依賴 | **12** | 大量（pnpm monorepo） | 30-40 core + 60+ 選用 | 50+ | 40+ | 40+ |
| 自維護生態系套件 | 6（go-bot / go-pkg / go-scheduler / ToriiDB / go-faas / KuraDB） | 0 | 0 | 0 | 0 | 0 |
| Runtime | Go（靜態 binary） | Node.js | Python | Node.js | Node.js + Rust | Node.js |
| 部署 | **單一 binary** | npm install | pip + docker/VPS | npm install | npm install | npm install |

***

### Agenvoy 的定位

| 面向 | 細節 |
|-----------|--------|
| **明確優勢** | 單一 Go binary、12 個依賴、自維護生態系（pardnchiu universe）、dispatcher 路由、Session Canvas、原生平台 UI（真實按鈕/modal）、OTP 驗證、跨 session 傳送至 Telegram/Discord、API 工具自動探索、格式參考作為 lazy-load 工具、純本地 scheduler（免雲端） |
| **不相上下** | Telegram/Discord daemon、TTS/STT、scheduler 輸出推送、Skill 系統、MCP、瀏覽器自動化、附件處理、provider 覆蓋度（compat 層涵蓋任何 OpenAI 相容 endpoint） |
| **競品領先之處** | Hermes context 壓縮引擎（token-budget 壓縮）、OpenClaw 24+ 平台、Hermes MCP server 模式、Hermes 本地 STT、OpenClaw/Hermes 內建跨 session 記憶、Claude Code Computer Use beta、Claude Code 雲端 cron/task |
| **Codex CLI** | 功能最少 — 僅 CLI + TUI + OpenAI OAuth，無 daemon、無聊天平台、無 scheduler |

***

## Agenvoy vs Hermes vs Pi：設計哲學比較

> 來源：[Agenvoy, Hermes, Pi — An AI Agent Platform Comparison](https://dev.to/pardnchiu/agenvoy-hermes-pi-an-ai-agent-platform-comparison-3p7p)

### 三個專案，三條不同賽道

| 專案 | 最接近的類比 | 最適合 |
|---------|-----------------|----------|
| **Agenvoy** | 一套完整、著重安全、內建能力深厚的 AI agent 平台 | 想要開箱即用、已具備大量功能之系統的人 |
| **Hermes** | 一套整合面廣、功能豐富、面向大規模部署的 agent 系統 | 需要串接眾多 model、平台與 channel 的人 |
| **Pi** | 一套輕量、高彈性、易於客製的 AI 框架 | 想自建 workflow 或將 AI 嵌入產品的人 |

***

### Agenvoy：高完整度、強安全、更深的自動化與分享

**優勢**

- **執行期動態建立工具** — 當系統發現缺少某項能力時，可在 workflow 中途建立一個真正可執行的工具，然後從中斷處繼續。其他系統說「我就用手上有的工具處理」；Agenvoy 說「如果我缺工具，我當場建一個」。
- **跨 AI 系統的工具分享** — Agenvoy 不只為自己建工具 — 還能將這些工具暴露給其他 AI 框架使用。你可以透過 Claude Code 建立一個工具、透過 Codex 使用、透過 Hermes 修復 — 全部跑在 Agenvoy 的 sandbox 中，全部即時跨 harness 分享。
- **預設安全隔離** — 安全從一開始就設計進去。Agenvoy「預設將安全視為框架的核心原則」；另外兩者比較像是「你需要的話可以自己加隔離」。
- **內建記憶與語意檢索** — 不只保存過往對話，還把「context 檢索」建進系統本身。三者之中，它是唯一將語意檢索作為預設層內建者。
- **對非程式設計者友善** — 著重讓使用者透過自然語言客製自己的 AI，而非每一步都需要改程式。

**劣勢**

- 在 model、平台與外部服務的覆蓋面上並非最廣。
- 生態系較小 — 資源較少、社群較小、外部文件較少。
- 在大規模整合情境下，相較於 Hermes 可能不具優勢。

***

### Hermes：整合面最廣、治理更成熟

**優勢**

- **強大的整合能力** — 非常適合串接各種 model、平台與外部服務。
- **廣泛的平台支援** — 跨多種環境運作：訊息平台、workflow 與服務。
- **成熟的自我演化與治理** — 著重系統如何隨時間維護、修補、組織與演化。
- **與 Agenvoy 互補** — 因為 Hermes 能接入眾多能力，而 Agenvoy 能供應工具給其他系統，兩者配合良好。

**劣勢**

- 複雜度較高 — 整合面越廣代表系統越重，學習與維護成本越高。
- 對只想快速上手的人不理想。
- 安全與合規需要更謹慎的逐一使用者評估。

***

### Pi：最輕量、最有彈性

**優勢**

- **高彈性** — 非常適合自訂 workflow 並親手形塑系統。
- **適合產品嵌入** — 輕量設計使其易於作為自有產品的一部分整合。
- **廣泛的 model 與 provider 支援** — 眾多 model 選擇與 provider 選項。

**劣勢**

- 開箱即用並非最完整 — 強項是彈性，而非內建的就緒度。
- 對一般使用者可能不是最友善 — 更像框架而非產品。
- 核心能力（記憶、工具成長、安全隔離）需自行建構。

***

### 選型指南

| 選擇 | 當你想要 |
|--------|---------------|
| **Agenvoy** | 完整系統、強安全、記憶連續性、最少組裝、自然語言客製、服務其他 AI 系統的工具、動態工具建立 |
| **Hermes** | 最廣的整合、多 channel 大規模部署、成熟治理、願意接受較高複雜度 |
| **Pi** | 輕量核心、高彈性、產品嵌入、客製自由、廣泛的 provider 選擇 |
