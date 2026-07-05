# 快速開始

## 前置需求

- Go 1.25.1 或更高版本
- Linux（透過 bubblewrap 沙箱；若缺少 `bwrap` 會自動經 apt/dnf/yum/pacman/apk 安裝）或 macOS（`sandbox-exec`）
- 至少一個 LLM provider 帳號（Copilot 訂閱，或 OpenAI / Claude / Gemini / Nvidia 的 API key）
- 選用：`pdftotext`（poppler-utils）供 `read_file` 解析 PDF
- 選用：`OPENAI_API_KEY` 以啟用語意搜尋（`text-embedding-3-small`）

## 安裝

```bash
git clone https://github.com/pardnchiu/Agenvoy.git
cd agenvoy
make build
```

`make build` 會編譯、將最新 git tag 嵌入為 `projectVersion`，並將 binary 安裝至 `/usr/local/bin/agen`。

## 至少配置一個 provider

Agenvoy 需要至少一個 LLM provider 才能運作：

```bash
agen model add
```

互動式提示會逐步帶你完成 provider 選擇、model 選擇與憑證儲存。Token 會落在 OS keychain（macOS 用 `security`、Linux 用 `secret-tool`，其餘則 fallback 至加密檔案），服務名稱固定為 `agenvoy`。

主要 config 位於 `~/.config/agenvoy/config.json`。

## 首次執行

```bash
# Create a named cli- session and switch the primary pointer to it
agen session new my-assistant

# Launch the full stack (TUI + Discord + Telegram + REST)
make app
```

TUI 啟動後輸入框始終保持 focus——直接輸入訊息，按 **Enter** 送出（**Alt+Enter** 插入換行）。輸入 **`/`** 開啟模糊比對的指令面板（比對指令名稱或說明）；**Tab**/**Enter** 確認選中項，**Esc** 關閉面板。**Shift+Tab** 切換工具呼叫的自動核准。輸入框為空時，**Shift+W**/**Shift+S** 可切換 dispatcher model，**Shift+A**/**Shift+D** 可切換 reasoning level。

一次性 CLI 用法：

```bash
make cli "summarize the latest changes in main.go"
make run "use playwright to open example.com and screenshot"
```

`make cli` 會確認每個非唯讀工具呼叫；`make run` 則自動核准一切。

## 後續步驟

- Core Concepts — session、agent routing、iteration 迴圈與三階段工具派送
- Providers — 支援的 LLM 後端與 dispatcher model
- MCP Integration — 接入外部工具 server
- CLI Reference — 完整命令清單

***

> [!NOTE]
> 本文件由 Claude 讀取完整原始碼後自動生成。
