<p align="center">
  <picture style="margin-down: 1rem">
    <img src="./logo.svg" alt="Agenvoy" width="320">
  </picture>
</p>

<p align="center">
  <strong>台灣自研的開源 AI Agent Harness — 自行建立、測試並重用工具</strong>
</p>

<p align="center">
  單一 Go binary，自架在你自己的電腦上：處理多步驟工作、搜尋本機檔案、排程重複任務，<br>
  並透過 MCP 把沙箱工具庫分享給 Claude Code、Codex 等 Agent。
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

## 為什麼是 Agenvoy

- **工具自我擴充** — 缺少工具時直接建立，不會停在原地。
- **一套沙箱工具庫** — 在 Agenvoy、Claude Code、Codex 等 Agent 間共用同一套工具庫。
- **即時執行回饋** — 將節流後的命令輸出串流至 TUI 與 Web 儀表板，同時保留完整的最終結果。
- **Skill 感知模型路由** — 將符合的 Skill 說明帶入 Agent 選擇，讓 dispatcher 依實際任務契約挑選模型。
- **音訊模型路由** — 從已設定的 OpenAI 或 Gemini provider 中，分別選擇語音轉文字與文字轉語音模型。
- **可擴充整合** — 連接 stdio 或 HTTP MCP server，支援 OAuth 登入並即時刷新遠端工具清單。
- **自架執行** — 在自己的機器上執行排程、記憶與檔案搜尋。
- **本機優先的 Agent** — Agent 直接執行在你的個人電腦上，使用本機檔案、工具、記憶與排程，不必將工作移交給託管服務。
- **私有聊天機器人存取** — Telegram 與 Discord 透過 token 連回本機 daemon，不需要曝光主機或設定入站網路。
- **Agent 應用 + MCP 伺服器** — 由同一個 binary 同時提供兩種角色。

## 你可以用它做什麼

<table>
<tr>
<td width="50%" valign="top">

### 問即時問題，拿到即時答案

> 台北現在天氣如何？
>
> Agent 會找即時資料、呼叫工具，再整理成答案回覆你。
>
> 如果缺工具，它會自己建立。

</td>
<td width="50%" valign="top">

### 一句話變成自動化流程

> 每天早上 8 點回報台積電股價
>
> Agent 會確認：
>
> - 要推送到哪裡
> - 你想要什麼格式
> - 何時執行
>
> 然後自動建立排程。

</td>
</tr>
<tr>
<td>

[![](https://i.ytimg.com/vi/floMBsAfziY/maxresdefault.jpg)](https://youtu.be/floMBsAfziY)

</td>
<td>

[![](https://i.ytimg.com/vi/5To3joKlFpU/maxresdefault.jpg)](https://youtu.be/5To3joKlFpU)

</td>
</tr>
<tr>
<td width="50%" valign="top">

### 直接詢問你的本機檔案

> 找出去年所有發票
>
> 哪份文件提到 Prompt guide？
>
> Agent 會搜尋你的本機檔案並直接回答。

</td>
<td width="50%" valign="top">

### 完成多步驟工作

> 幫我整理今天的 GitHub commits，並生成進度報告
>
> Agent 會拆解任務、呼叫工具、整合結果，再回覆給你。

</td>
</tr>
<tr>
<td>

[![](https://i.ytimg.com/vi/vqoQ6Qvl8qU/maxresdefault.jpg)](https://youtu.be/vqoQ6Qvl8qU)

</td>
<td>

[![](https://i.ytimg.com/vi/nIV1xz_HIJg/maxresdefault.jpg)](https://youtu.be/nIV1xz_HIJg)

</td>
</tr>
</table>

### 能跟你已經在用的 Agent 一起工作

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

## 適合誰使用

Agenvoy 適合開發者、技術營運，以及需要超越聊天能力的 AI 工作流：

- 想自架 AI Agent、同時保有沙箱安全邊界的人
- 想在多個 Agent 間重用工具的團隊
- 需要把自動化、檔案搜尋與定期報告整合在一起的使用者

---

## 核心能力

| 能力         | 說明                                                |
| :----------- | :-------------------------------------------------- |
| 自動工具生成 | 缺工具時自行建立並保存                              |
| 自我排程     | 一句話建立定時任務                                  |
| 長期記憶     | 保留重要資訊與上下文                                |
| 圖片生成     | 透過已設定的 provider 生成圖片                           |
| 即時命令輸出 | 將 `run_command` 進度串流至 TUI 與 Web 儀表板            |
| 安全檔案邊界 | 存取敏感路徑或 `$HOME` 外路徑前要求確認                  |
| MCP OAuth    | 登入 HTTP MCP server，並將 token 保存至作業系統 keychain |
| 知識筆記     | 回答前先讀你留下的筆記                              |
| 檔案搜尋     | 從本機檔案回答問題                                  |
| Sub-Agent    | 多 Agent 協作                                       |
| MCP client   | 以官方 go-sdk 連接外部 MCP 服務（工具清單即時刷新） |
| MCP server   | 讓任何 MCP 相容 Agent 使用你的沙箱工具              |
| 推理指引     | 透過 `reasoning_guide(topic=...)` 按需載入規則      |
| Tool Market  | 分享與安裝工具                                      |
| 語音轉錄     | 音訊與影片轉文字                                    |
| 自我改進     | 執行失敗後自動修正                                  |

---

## Web 儀表板

當你的機器上已啟動 daemon，直接在瀏覽器開啟 **[web.agenvoy.com](https://web.agenvoy.com)** 即可連上儀表板。

<p align="center">
  <a href="https://youtu.be/n8tHHSCwOjE">
    <img src="https://img.youtube.com/vi/n8tHHSCwOjE/maxresdefault.jpg" alt="Agenvoy Web 儀表板示範" width="640">
  </a>
</p>

<p align="center">
  <a href="https://youtu.be/n8tHHSCwOjE">▶ 觀看 Web 儀表板操作影片</a>
</p>

## 聊天機器人整合

Agenvoy 目前只支援 **Telegram 與 Discord** 作為聊天機器人頻道。兩者都由本機 daemon 主動向外連線，因此不需要將主機暴露到公開網際網路；使用者只要提供 bot token，不必設定入站連接埠、反向代理或其他網路環境。除非後續平台能帶來明確的安全性改善，否則不會擴充其他聊天機器人平台。

語音輸出目前暫時無法在 Telegram 與 Discord 使用；音訊模型設定仍可供本機工具使用，並保留給未來頻道支援。

## 一鍵安裝

> MacBook 建議額外執行 `sudo pmset -c sleep 0`，避免休眠中斷排程。

```bash
curl -fsSL https://agenvoy.com/scripts/install.sh | bash
```

---

## 開發者建議

一組省錢好上手的模型配置：

- 申請免費的 **[NVIDIA NIM](https://build.nvidia.com/explore/discover)** API token，使用 `gpt-oss-20b` 作為 **dispatcher** 模型。
- 用訂閱制當作 **主力** 模型：
  - OpenAI ChatGPT Plus（$20/月）
  - SuperGrok（$30/月）

NVIDIA NIM 已不再支援 `gpt-oss-120b`，因此不再將它列為 fallback 或 summary 模型推薦。

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
