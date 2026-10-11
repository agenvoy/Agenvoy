<p align="center">
  <picture style="margin-down: 1rem">
    <img src="./logo.svg" alt="Agenvoy" width="320">
  </picture>
</p>

<p align="center">
  <strong>Agenvoy · TUI-only 版本</strong>
</p>

<p align="center">
  <code>master</code> 移除 Web UI 後的輕量版本，只保留 TUI 與 OpenAI 相容 completions。
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

---

## 這個分支是什麼

這是 [Agenvoy](https://github.com/pardnchiu/agenvoy) 的 TUI-only 版本，內容就是 `master` 拿掉 Web UI，沒有其他差異：Agent runtime、工具、記憶、排程、MCP、沙箱，以及 Telegram／Discord 頻道都與 `master` 同一份程式碼。

**所有功能與行為一律以 `master` 最新版本為準。** 產品介紹、安裝方式與工具說明請見 [`master` 的 README](https://github.com/pardnchiu/agenvoy/blob/master/doc/README.zh.md) 與 [agenvoy.com/docs](https://agenvoy.com/docs/)；其中屬於 Web UI 的功能在這裡不存在。

## 移除了什麼

| 移除項目                                                    | 在 `master` 的位置                                          |
| ----------------------------------------------------------- | ----------------------------------------------------------- |
| 內嵌 Web 儀表板、PWA 與 service worker                      | `page/`、`internal/runtime/routes/page.go`、`sw.go`         |
| 桌面 web app 包裝與 vendor 資產下載                         | `internal/runtime/webapp/`                                  |
| Web 專用 HTTP API（`/v1/send`、`/v1/session/*`、`/v1/mcp`...） | `internal/runtime/routes/handler/`                          |
| `chat-` session 的 Web 工具確認轉送                         | `handler/confirm.go`                                        |
| 追問建議（follow-up）與 voice live                          | `internal/agents/exec/followup/`、`handler/voice.go`        |

原本由這些 endpoint 管理的操作，全部改由 TUI 透過 daemon IPC socket 處理。

## HTTP API

Daemon 監聽 `127.0.0.1:17989`，其餘路由一律回 JSON 404。

**OpenAI 相容**

| Method | Path                   | 說明                                                 |
| ------ | ---------------------- | ---------------------------------------------------- |
| POST   | `/v1/chat/completions` | 支援串流與非串流；帶 `X-Agenvoy-Agent: 1` 啟用 agent 模式 |
| GET    | `/v1/models`           | 已設定的模型加上 `auto`                              |
| GET    | `/v1/models/*id`       | 單一模型物件                                         |

**TUI 內部**（僅限本機）

| Method       | Path                                | 使用者                              |
| ------------ | ----------------------------------- | ----------------------------------- |
| GET          | `/v1/log`                           | Daemon log 串流（SSE），擷取驗證碼  |
| GET          | `/v1/providers`                     | `/model add`                        |
| GET          | `/v1/provider/:provider/console`    | `/model add` 主控台連結（302）      |
| GET          | `/v1/roles`、`/v1/role/*name`       | `/role`                             |
| POST / PATCH | `/v1/role`                          | `/role`                             |
| DELETE       | `/v1/channel/:channel/chat`         | `/channel`                          |

## 建置

```bash
make build   # 建置 agen 並安裝到 /usr/local/bin/agen
make app     # 停止 daemon、重新建置並開啟 TUI
```

`master` 的一鍵安裝腳本安裝的是 `master` release，包含 Web UI；要取得 TUI-only 執行檔請從這個分支建置。

## 與 `master` 同步

```bash
git fetch origin master
git merge origin/master   # 解衝突時維持 Web UI 移除狀態
```

衝突幾乎都落在被移除的路徑與銜接點：`page/`、`internal/runtime/webapp/`、`internal/runtime/routes/new.go`、`internal/runtime/routes/handler/`、`cmd/app/daemon.go`。解法是再次刪除 Web 端、路由只保留上述兩組；`master` 新增的 Web 專用 handler 捨棄，新的 TUI 或 runtime 功能保留。

## License

本專案採雙授權。

- **開源授權** — [GNU Affero General Public License v3.0](../LICENSE)。可自由使用、修改與散布，但衍生作品與基於本專案提供的網路服務，須以相同授權釋出並提供原始碼。
- **商業授權** — 無法滿足 AGPL-3.0 原始碼公開義務的使用情境，可洽詢商業授權。僅涵蓋軟體本身的使用權利；任何委由開發者進行、與本專案有關的改動皆另行計價。詳見 [COMMERCIAL.zh.md](./COMMERCIAL.zh.md)。

依 AGPL-3.0 第 7(b) 條，Web 介面 `Settings > Other` 底下的 **License** 與 **Source** 兩張卡片必須保留：它們承載著作權聲明、無擔保聲明、授權聲明，以及第 13 條要求的對應原始碼連結。可以調整位置或樣式，但須維持顯眼可及，不得移除或隱藏。取得商業授權即免除此義務。

---

©️ 2026 [邱敬幃 Pardn Chiu](https://www.linkedin.com/in/pardnchiu)
