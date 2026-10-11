<p align="center">
  <picture style="margin-down: 1rem">
    <img src="./doc/logo.svg" alt="Agenvoy" width="320">
  </picture>
</p>

<p align="center">
  <strong>Agenvoy · TUI-only edition</strong>
</p>

<p align="center">
  A lightweight build of <code>master</code> with the web UI removed — TUI and OpenAI-compatible completions only.
</p>

<p align="center">
<a href="https://trendshift.io/repositories/41899?utm_source=trendshift-badge&amp;utm_medium=badge&amp;utm_campaign=badge-trendshift-41899" target="_blank" rel="noopener noreferrer">
<img src="https://trendshift.io/api/badge/trendshift/repositories/41899/daily?language=Go" alt="agenvoy%2FAgenvoy | Trendshift" width="250" height="55"/>
</a>
</p>

<p align="center">
  <a href="https://github.com/pardnchiu/agenvoy/releases"><img src="https://img.shields.io/github/v/tag/pardnchiu/agenvoy?include_prereleases&style=for-the-badge" alt="Version"></a>
  <a href="LICENSE"><img src="https://img.shields.io/github/license/pardnchiu/agenvoy?include_prereleases&style=for-the-badge" alt="License"></a>
</p>

<p align="center">
  <strong>English</strong> · <a href="doc/README.zh.md">繁體中文</a>
</p>

---

## What this branch is

This is the TUI-only edition of [Agenvoy](https://github.com/pardnchiu/agenvoy). It is `master` with the web UI stripped out, nothing more: the agent runtime, tools, memory, scheduling, MCP, sandbox, and the Telegram / Discord channels are the same code as `master`.

**All features and behaviour follow the latest `master`.** For the product overview, install instructions, and tool reference, see the [README on `master`](https://github.com/pardnchiu/agenvoy/blob/master/README.md) and [agenvoy.com/docs](https://agenvoy.com/docs/). Anything those describe as a web UI feature does not exist here.

## What is removed

| Removed                                                         | Location on `master`                                        |
| --------------------------------------------------------------- | ----------------------------------------------------------- |
| Embedded web dashboard, PWA, and service worker                 | `page/`, `internal/runtime/routes/page.go`, `sw.go`         |
| Desktop web-app wrapper and vendor asset download               | `internal/runtime/webapp/`                                  |
| Web-only HTTP API (`/v1/send`, `/v1/session/*`, `/v1/mcp`, ...) | `internal/runtime/routes/handler/`                          |
| Web tool-confirm relay for `chat-` sessions                     | `handler/confirm.go`                                        |
| Follow-up suggestions and voice live                            | `internal/agents/exec/followup/`, `handler/voice.go`        |

Everything those endpoints used to manage is handled by the TUI through the daemon IPC socket.

## HTTP API

The daemon listens on `127.0.0.1:17989`. Every other route returns a JSON 404.

**OpenAI compatible**

| Method | Path                   | Notes                                                |
| ------ | ---------------------- | ---------------------------------------------------- |
| POST   | `/v1/chat/completions` | Streaming and non-streaming; `X-Agenvoy-Agent: 1` enables agent mode |
| GET    | `/v1/models`           | Configured models plus `auto`                        |
| GET    | `/v1/models/*id`       | Single model object                                  |

**TUI internal** (localhost only)

| Method       | Path                                | Used by                                     |
| ------------ | ----------------------------------- | ------------------------------------------- |
| GET          | `/v1/log`                           | Daemon log stream (SSE) for verification codes |
| GET          | `/v1/providers`                     | `/model add`                                |
| GET          | `/v1/provider/:provider/console`    | `/model add` console link (302)             |
| GET          | `/v1/roles`, `/v1/role/*name`       | `/role`                                     |
| POST / PATCH | `/v1/role`                          | `/role`                                     |
| DELETE       | `/v1/channel/:channel/chat`         | `/channel`                                  |

## Build

```bash
make build   # builds agen and installs it to /usr/local/bin/agen
make app     # stops the daemon, rebuilds, and attaches the TUI
```

The one-line install script on `master` installs the `master` release, which includes the web UI. Build from this branch to get the TUI-only binary.

## Staying in sync with `master`

```bash
git fetch origin master
git merge origin/master   # resolve conflicts, keep the web UI removed
```

Conflicts land almost entirely on the removed paths and their seams: `page/`, `internal/runtime/webapp/`, `internal/runtime/routes/new.go`, `internal/runtime/routes/handler/`, and `cmd/app/daemon.go`. Resolve them by deleting the web side again and keeping the routes limited to the two groups above; new web-only handlers from `master` are dropped, new TUI or runtime features are kept.

## License

This project is dual-licensed.

- **Open source** — [GNU Affero General Public License v3.0](LICENSE). You may use, modify and distribute it, provided derivative works and any network-accessible service built on it are released under the same license, source included.
- **Commercial** — for use that cannot meet the AGPL-3.0 source-disclosure requirement, a commercial license is available. It covers rights to the software only; any work on the project requested from the developer is quoted and charged separately. See [COMMERCIAL.md](COMMERCIAL.md).

Under section 7(b) of the AGPL-3.0, the **License** and **Source** entries under `Settings > Other` in the web interface must be preserved: they carry the copyright notice, the warranty disclaimer, the license statement, and the corresponding-source link that section 13 requires. They may be relocated or restyled while remaining prominently visible, but not removed or hidden. A commercial license removes this requirement.

---

©️ 2026 [邱敬幃 Pardn Chiu](https://www.linkedin.com/in/pardnchiu)
