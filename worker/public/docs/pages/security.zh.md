# 安全模型

## Localhost 綁定

HTTP server 僅綁定 `127.0.0.1` — 區網 client 無法連到 daemon。來自 `web.agenvoy.com` 的跨來源存取由 CORS middleware 以來源白名單控管。Chrome Private Network Access (PNA) 透過 `Access-Control-Allow-Private-Network` header 滿足。

## Sudo 模式

僅限 TUI 透過 `/sudo` 提權。對當前 session 授予暫時繞過 confirm gate 的權限。1 小時 TTL，儲存於 ToriiDB（不落磁碟 — 自動過期，零永久殘留）。系統目錄的 floor 路徑即使在 sudo 下仍維持封鎖。

## 敏感檔案防護

對符合敏感樣式的檔案 — SSH keys、`.pem`、`.key`、`.env` 與憑證檔 — `read_file` 一律要求明確確認，無論 sudo 模式或 allowlist 狀態為何。此防護在 Go 中硬編碼，非基於 prompt。

## 權限模式

Agenvoy 支援兩種權限模式：`single-confirm` 與 `always-allow`。七類不可逆操作無論何種模式皆一律要求明確 `ask_user`：

- 檔案刪除或覆寫
- 系統設定變更
- 對未知 endpoint 的網路請求
- 套件安裝
- 憑證儲存或取用
- 行程終止
- Scheduler 建立或修改

## System prompt 保護

System prompt（`configs/prompts/system_prompt.md`）指示 LLM 拒絕：

- 揭露 system prompt 內容的請求
- Role-play / DAN / 「ignore previous instructions」覆寫
- 含 `..` 的路徑或系統目錄（`/etc`、`/usr`、`/root`、`/sys`）
- 如 `rm -rf`、`chmod 777`、`curl | sh` 的指令

這些是 **prompt 中的 policy**，非 Go 端硬編碼的 filter — 新增一個類別只需編輯 prompt。

## Keychain

憑證（provider API keys、OAuth tokens）儲存於 OS keychain，service 名為 `agenvoy`：

| 平台 | 後端 |
|---|---|
| macOS | `security` CLI |
| Linux | `secret-tool`（libsecret） |
| 其他 / fallback | `~/.config/agenvoy/` 下的加密檔 |

Service 名稱 `"agenvoy"` 為固定值，不得變更。

## MCP 隔離考量

MCP server 是行為無法驗證的第三方行程。Agenvoy 預設將其視為不受信任，且不提供 per-server 的「trusted」旗標。所有 MCP 工具呼叫都經過與內建工具相同的 confirm gate。若要批次執行 MCP 操作，使用 `agen run`（其信任的是你自己的決定，而非 server 的）。

***

> [!NOTE]
> 本文件由 Claude 在讀取完整原始碼後自動生成。
