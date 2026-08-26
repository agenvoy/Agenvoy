<p align="center">
  <picture style="margin-down: 1rem">
    <img src="./doc/logo.svg" alt="Agenvoy" width="320">
  </picture>
</p>

<p align="center">
  <strong>Self-hosted AI agent harness that builds, tests, and reuses its own tools</strong>
</p>

<p align="center">
  Open source, single Go binary. Runs on your own machine, handles multi-step work, searches local files,<br>
  schedules recurring tasks, and shares its sandboxed tool library with Claude Code, Codex,<br>
  and any other MCP client.
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
  <strong>English</strong> · <a href="./doc/README.zh.md">繁體中文</a>
</p>

## Why Agenvoy

- **Self-extending tools** — Builds the missing tool instead of stopping.
- **One sandboxed tool library** — Shares the same tool library across Agenvoy, Claude Code, Codex, and other agents.
- **Live command feedback** — Streams throttled command output to the TUI and Web dashboard while preserving the complete final result.
- **Skill-aware model routing** — Passes matched Skill descriptions into agent selection so the dispatcher can choose models against the actual task contract.
- **Origin-aware confirmations** — Routes approval prompts back to the originating CLI, Web, Telegram, or Discord channel without cross-channel interception.
- **Extensible integrations** — Connects stdio or HTTP MCP servers, supports OAuth login, and refreshes remote tool catalogs.
- **Self-hosted execution** — Runs scheduling, memory, and file search on your own machine.
- **Agent app and MCP server** — Provides both roles from the same binary.

## What you can do with it

<table>
<tr>
<td width="50%" valign="top">

### Ask for live answers

> What's the weather in Taipei?
>
> The agent finds current data, calls tools, and gives you the answer.
>
> If a tool doesn't exist, it builds one.

</td>
<td width="50%" valign="top">

### Turn one sentence into automation

> Report TSMC stock price every morning at 8am
>
> The agent asks:
>
> - Where to push results
> - What format you want
> - When to run
>
> Then creates the schedule automatically.

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

### Ask your local files questions

> Find all invoices from last year
>
> Which document mentions Prompt guide?
>
> The agent searches your local files and answers directly.

</td>
<td width="50%" valign="top">

### Finish multi-step work

> Summarize today's GitHub commits and generate a progress report
>
> The agent breaks down the task, calls tools, combines results, and replies.

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

### Works with the agents you already use

> Agenvoy is also an MCP server.
>
> Claude Code, Codex, OpenCode, and other AI agents can connect and:
>
> - Use all your sandboxed tools
> - Auto-build new tools when none exist
> - Share every tool across all agents
>
> One line of config. Instant shared tool library.
> Tools created in the demo: [`fetch_weather`](doc/demo/fetch_weather/) · [`fetch_crypto_price`](doc/demo/fetch_crypto_price/)

<table>
<tr>
<td width="33%" valign="top">

#### Claude Code creates a weather tool (1)

</td>
<td width="33%" valign="top">

#### Codex reuses it and creates a crypto tool (2)

</td>
<td width="33%" valign="top">

#### Agenvoy tests both tools (3)

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

## Who it's for

Agenvoy is for developers, technical operators, and AI-heavy workflows that need more than chat:

- People who want a self-hosted agent with sandbox guardrails
- Teams that want reusable tools across agents
- Users who need automation, file search, and scheduled reporting in one place

---

## Core capabilities

| Capability           | Description                                                              |
| :------------------- | :----------------------------------------------------------------------- |
| Auto tool generation | Builds and saves tools when they're missing                              |
| Self-scheduling      | Create cron jobs with a single sentence                                  |
| Long-term memory     | Retains key info and context                                             |
| Knowledge notes      | Reads the notes you keep, before it answers                              |
| File search          | Answers from your local files                                            |
| Sub-Agent            | Multi-agent collaboration                                                |
| MCP client           | Connect to external MCP services via official go-sdk (live tool refresh) |
| MCP server           | Expose sandboxed tools to any MCP-compatible agent                       |
| Reasoning guides     | On-demand rules via `reasoning_guide(topic=...)`                         |
| Tool Market          | Share and install tools                                                  |
| Image generation     | Generate images through a configured provider                         |
| Live command output  | Stream `run_command` progress to the TUI and Web dashboard            |
| Secure file boundary | Confirm sensitive paths and out-of-home access before granting them   |
| MCP OAuth            | Log in to HTTP MCP servers and persist tokens in the OS keychain      |
| Transcription        | Audio and video to text                                                  |
| Self-improvement     | Auto-fixes after execution failures                                      |

---

## Web Dashboard

Open **[web.agenvoy.com](https://web.agenvoy.com)** in your browser to reach the dashboard once the daemon is running on your machine.

<p align="center">
  <a href="https://youtu.be/n8tHHSCwOjE">
    <img src="https://img.youtube.com/vi/n8tHHSCwOjE/maxresdefault.jpg" alt="Agenvoy Web Dashboard demo" width="640">
  </a>
</p>

<p align="center">
  <a href="https://youtu.be/n8tHHSCwOjE">▶ Watch the Web Dashboard walkthrough</a>
</p>

## One-line install

> On MacBook, also run `sudo pmset -c sleep 0` to prevent sleep from interrupting schedules.

```bash
curl -fsSL https://agenvoy.com/scripts/install.sh | bash
```

---

## Developer Recommendations

A cost-effective model setup to get started:

- Apply for a free **[NVIDIA NIM](https://build.nvidia.com/explore/discover)** API token, then use it for:
  - `gpt-oss-20b` as the **dispatcher** model
  - `gpt-oss-120b` as the **fallback** and **summary** model
- Use a subscription plan as your **primary** model:
  - OpenAI ChatGPT Plus ($20/mo)
  - SuperGrok ($30/mo)

---

## Docs

Full documentation at **[agenvoy.com/docs](https://agenvoy.com/docs/)**

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

This project is licensed under the [Apache License 2.0](LICENSE).

## Author

Just [open an issue](https://github.com/pardnchiu/agenvoy/issues/new) to share an idea.

<a href="https://github.com/pardnchiu/agenvoy/graphs/contributors">
  <img src="https://contrib.rocks/image?repo=pardnchiu/agenvoy&cache_bust=2026-05-12" alt="Agenvoy contributors" />
</a>

---

©️ 2026 [邱敬幃 Pardn Chiu](https://www.linkedin.com/in/pardnchiu)
