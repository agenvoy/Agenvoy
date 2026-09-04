<p align="center">
  <picture style="margin-down: 1rem">
    <img src="./logo.svg" alt="Agenvoy" width="320">
  </picture>
</p>

<p align="center">
  <strong>在自己的電腦上執行的開源 AI Agent：可建立工具、完成工作，也能與其他 Agent 共用</strong>
</p>

<p align="center">
  一個單一 Go 執行檔，直接在你的電腦上運作。它能處理多步驟任務、搜尋本機檔案、安排重複工作，<br>
  也能透過 MCP 與 Claude Code、Codex 等 Agent 共用安全沙箱中的工具。
</p>

<p align="center">
<a href="https://trendshift.io/repositories/41899?utm_source=trendshift-badge&amp;utm_medium=badge&amp;utm_campaign=badge-trendshift-41899" target="_blank" rel="noopener noreferrer">
<img src="https://trendshift.io/api/badge/trendshift/repositories/41899/daily?language=Go" alt="agenvoy%2FAgenvoy | Trendshift" width="250" height="55"/>
</a>
</p>

<p align="center">
  <a href="https://github.com/pardnchiu/agenvoy/releases"><img src="https://img.shields.io/github/v/tag/pardnchiu/agenvoy?include_prereleases&style=for-the-badge" alt="Version"></a>
  <a href="../LICENSE"><img src="https://img.shields.io/github/license/pardnchiu/agenvoy?include_prereleases&style=for-the-badge" alt="License"></a>
</p>

<p align="center">
  <a href="../README.md">English</a> · <strong>繁體中文</strong>
</p>

## 為什麼選擇 Agenvoy

Agenvoy 把 AI Agent 放在你的個人電腦上執行：檔案、工具、排程與記憶都由你掌控。它不只回答問題，也能實際執行任務，並在需要時建立新工具。

- **遇到缺口可建立工具** — 找不到合適工具時，可建立、測試並保留新工具供後續使用。
- **工具可跨 Agent 共用** — Agenvoy、Claude Code、Codex 等 Agent 可使用同一套沙箱工具。
- **看得到執行過程** — 命令輸出會同步顯示在 TUI 與 Web 儀表板，最後仍會提供完整結果。
- **依任務選擇合適模型** — 依 Skill 與任務需求路由模型，也可分別設定圖片生成模型、語音轉文字（STT）與文字轉語音（TTS）。
- **可串接外部服務** — 支援 stdio 與 HTTP MCP server、OAuth 登入，以及遠端工具清單即時更新。
- **本機優先、保有控制權** — 檔案搜尋、排程與記憶在你的機器上執行，不必把工作交給託管服務。
- **可私下從聊天軟體存取** — Telegram 與 Discord 由本機 daemon 主動連線，不需公開你的主機。

## 你可以用它做什麼

<details>
<summary><strong>問即時問題，拿到即時答案（Web Search／Tool Generate）</strong></summary>

> 台北現在天氣如何？
>
> Agent 會找即時資料、呼叫工具，再整理成答案回覆你。
>
> 如果缺工具，它會自己建立。

[![](https://i.ytimg.com/vi/floMBsAfziY/maxresdefault.jpg)](https://youtu.be/floMBsAfziY)

</details>

<details>
<summary><strong>一句話變成自動化流程（Scheduler）</strong></summary>

> 每天早上 8 點回報台積電股價
>
> Agent 會確認：
>
> - 要推送到哪裡
> - 你想要什麼格式
> - 何時執行
>
> 然後自動建立排程。

[![](https://i.ytimg.com/vi/5To3joKlFpU/maxresdefault.jpg)](https://youtu.be/5To3joKlFpU)

</details>

<details>
<summary><strong>直接詢問你的本機檔案（File Search／RAG）</strong></summary>

> 找出去年所有發票
>
> 哪份文件提到 Prompt guide？
>
> Agent 會搜尋你的本機檔案並直接回答。

[![](https://i.ytimg.com/vi/vqoQ6Qvl8qU/maxresdefault.jpg)](https://youtu.be/vqoQ6Qvl8qU)

</details>

<details>
<summary><strong>完成多步驟工作（Skills／Sub-agents）</strong></summary>

> 幫我整理今天的 GitHub commits，並生成進度報告
>
> Agent 會拆解任務、呼叫工具、整合結果，再回覆給你。

[![](https://i.ytimg.com/vi/nIV1xz_HIJg/maxresdefault.jpg)](https://youtu.be/nIV1xz_HIJg)

</details>

<details>
<summary><strong>能跟你已經在用的 Agent 一起工作（MCP Server）</strong></summary>

> Agenvoy 也是一個 MCP server。
>
> Claude Code、Codex、OpenCode 與其他 AI Agent 連上後，可以：
>
> - 使用你所有的沙箱工具
> - 在缺工具時自動建立新工具
> - 讓所有 Agent 共用同一套工具
>
> 一行設定，即時共享工具庫。
> 影片中建立的工具：[`fetch_weather`](demo/fetch_weather/) · [`fetch_crypto_price`](demo/fetch_crypto_price/)

<table>
<tr>
<td width="33%" valign="top">

#### Claude Code 建立天氣工具 (1)

</td>
<td width="33%" valign="top">

#### Codex 重用它並建立加密貨幣工具 (2)

</td>
<td width="33%" valign="top">

#### Agenvoy 測試兩個工具 (3)

</td>
</tr>
<tr>
<td>

[![](https://i.ytimg.com/vi/on5IaoxBO1E/maxresdefault.jpg)](https://youtu.be/on5IaoxBO1E)

</td>
<td>

[![](https://i.ytimg.com/vi/2DDFCIcbnso/maxresdefault.jpg)](https://youtu.be/2DDFCIcbnso)

</td>
<td>

[![](https://i.ytimg.com/vi/KPs4o9xDFjM/maxresdefault.jpg)](https://youtu.be/KPs4o9xDFjM)

</td>
</tr>
</table>

</details>

## 適合誰使用

Agenvoy 適合開發者、技術營運，以及需要超越聊天能力的 AI 工作流：

- 想自架 AI Agent、同時保有沙箱安全邊界的人
- 想在多個 Agent 間重用工具的團隊
- 需要把自動化、檔案搜尋與定期報告整合在一起的使用者

---

## 從瀏覽器操控你的 Agent

在瀏覽器管理 session、工具、排程與記憶。儀表板內建於執行檔中 — 啟動 daemon 後開啟 [http://127.0.0.1:17989](http://127.0.0.1:17989)。由你自己的機器提供服務，資料不離開你的裝置。

<p align="center">
  <a href="https://youtu.be/oaXxrTNvLaU">
    <img src="https://img.youtube.com/vi/oaXxrTNvLaU/maxresdefault.jpg" alt="Agenvoy Web 儀表板示範" width="640">
  </a>
</p>

## 聊天機器人整合

目前支援 **Telegram 與 Discord**。Agenvoy 的本機 daemon 會主動連向這些平台，因此你只要設定 bot token：不必開放入站連接埠、不必架反向代理，也不必把自己的主機公開到網路上。

自 **v0.34.4** 起，Telegram 與 Discord 暫停「語音輸入後自動以語音回覆」的預設流程；仍可使用 STT／TTS 生成音訊，並將音訊檔傳送到頻道。

---

## 一鍵安裝

<details open>
<summary><strong>macOS／Linux 發行版</strong></summary>

在終端機執行：

```bash
curl -fsSL https://agenvoy.com/scripts/install.sh | bash
```

> **macOS 提示：**若使用 MacBook 執行排程，建議額外執行：
>
> ```bash
> sudo pmset -c sleep 0
> ```
>
> 避免電腦休眠而中斷排程。

</details>

<details>
<summary><strong>Windows（透過 WSL）</strong></summary>

請先以系統管理員身分開啟 PowerShell，查看並安裝一個 Linux 發行版：

```powershell
wsl --online --list
wsl --install <發行版名稱>
```

完成安裝、重新開機並進入 WSL 終端機後，執行：

```bash
curl -fsSL https://agenvoy.com/scripts/install.sh | bash
```

</details>

## 建議的模型設定

以下是一組容易上手、成本較低的配置：

1. 選擇一個訂閱制模型作為日常主要模型，例如：
   - OpenAI ChatGPT Plus（$20／月）
   - SuperGrok（$30／月）
2. **如果有多種 provider 的需求**，可以申請免費的 **[NVIDIA NIM](https://build.nvidia.com/explore/discover)** API token，並將 `gpt-oss-20b` 設為 **dispatcher** 模型，搭配智慧路由，兼顧快速回應。

---

## 核心能力

| 能力         | 說明                                                     |
| :----------- | :------------------------------------------------------- |
| 自動工具生成 | 缺工具時自行建立並保存                                   |
| 自我排程     | 一句話建立定時任務                                       |
| 長期記憶     | 保留重要資訊與上下文                                     |
| 圖片生成     | 透過已設定的 provider 生成圖片                           |
| 即時命令輸出 | 將 `run_command` 進度串流至 TUI 與 Web 儀表板            |
| 安全檔案邊界 | 存取敏感路徑或 `$HOME` 外路徑前要求確認                  |
| MCP OAuth    | 登入 HTTP MCP server，並將 token 保存至作業系統 keychain |
| 知識筆記     | 回答前先讀你留下的筆記                                   |
| 檔案搜尋     | 從本機檔案回答問題                                       |
| Sub-Agent    | 多 Agent 協作                                            |
| MCP client   | 以官方 go-sdk 連接外部 MCP 服務（工具清單即時刷新）      |
| MCP server   | 讓任何 MCP 相容 Agent 使用你的沙箱工具                   |
| 推理指引     | 透過 `reasoning_guide(topic=...)` 按需載入規則           |
| Tool Market  | 分享與安裝工具                                           |
| 語音轉錄     | 音訊與影片轉文字                                         |
| 自我改進     | 執行失敗後自動修正                                       |

---

## 文件

完整文件請見 **[agenvoy.com/docs](https://agenvoy.com/docs/)**

- [Getting Started](https://agenvoy.com/docs/getting-started)
- [Sessions & Agents](https://agenvoy.com/docs/sessions)
- [Providers](https://agenvoy.com/docs/providers)
- [Built-in Tools](https://agenvoy.com/docs/built-in-tools)
- [MCP Server](https://agenvoy.com/docs/mcp-server)
- [MCP Client](https://agenvoy.com/docs/mcp-client)
- [Memory System](https://agenvoy.com/docs/memory-system)
- [Skill System](https://agenvoy.com/docs/skill-basics)
- [Sandbox](https://agenvoy.com/docs/sandbox)
- [Architecture](https://agenvoy.com/docs/architecture)

## License

本專案以 [Apache License 2.0](../LICENSE) 授權。

## Author

歡迎 [開 issue](https://github.com/pardnchiu/agenvoy/issues/new) 分享想法。

<a href="https://github.com/pardnchiu/agenvoy/graphs/contributors">
  <img src="https://contrib.rocks/image?repo=pardnchiu/agenvoy&cache_bust=2026-05-12" alt="Agenvoy contributors" />
</a>

---

©️ 2026 [邱敬幃 Pardn Chiu](https://www.linkedin.com/in/pardnchiu)
