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

Agenvoy runs AI agents on your personal computer: you stay in control of files, tools, schedules, and memory. It does more than answer questions—it performs work and can build new tools when needed.

- **Builds tools when there is a gap** — Creates, tests, and retains a new tool when no suitable one exists.
- **Shares tools across agents** — Agenvoy, Claude Code, Codex, and other agents can use the same sandboxed tool library.
- **Shows execution as it happens** — Streams command output to the TUI and Web dashboard while preserving the complete final result.
- **Routes to the right model for the task** — Routes models by Skill and task requirements; configure separate image generation, speech-to-text (STT), and text-to-speech (TTS) models.
- **Connects external services** — Supports stdio and HTTP MCP servers, OAuth login, and live remote tool-catalog refreshes.
- **Local-first and in your control** — File search, schedules, and memory run on your machine rather than a hosted service.
- **Private access from chat platforms** — The local daemon initiates connections to Telegram and Discord, so you do not need to expose your host.

## What you can do with it

<details>
<summary><strong>Ask live questions and get live answers (Web Search / Tool Generate)</strong></summary>

> What's the weather in Taipei?
>
> The agent finds current data, calls tools, and gives you the answer.
>
> If a tool doesn't exist, it builds one.

[![](https://i.ytimg.com/vi/floMBsAfziY/maxresdefault.jpg)](https://youtu.be/floMBsAfziY)

</details>

<details>
<summary><strong>Turn one sentence into automation (Scheduler)</strong></summary>

> Report TSMC stock price every morning at 8am
>
> The agent asks:
>
> - Where to push results
> - What format you want
> - When to run
>
> Then creates the schedule automatically.

[![](https://i.ytimg.com/vi/5To3joKlFpU/maxresdefault.jpg)](https://youtu.be/5To3joKlFpU)

</details>

<details>
<summary><strong>Ask questions about your local files (File Search / RAG)</strong></summary>

> Find all invoices from last year
>
> Which document mentions Prompt guide?
>
> The agent searches your local files and answers directly.

[![](https://i.ytimg.com/vi/vqoQ6Qvl8qU/maxresdefault.jpg)](https://youtu.be/vqoQ6Qvl8qU)

</details>

<details>
<summary><strong>Finish multi-step work (Skills / Sub-agents)</strong></summary>

> Summarize today's GitHub commits and generate a progress report
>
> The agent breaks down the task, calls tools, combines results, and replies.

[![](https://i.ytimg.com/vi/nIV1xz_HIJg/maxresdefault.jpg)](https://youtu.be/nIV1xz_HIJg)

</details>

<details>
<summary><strong>Work with the agents you already use (MCP Server)</strong></summary>

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

</details>

## Who it's for

Agenvoy is for developers, technical operators, and AI-heavy workflows that need more than chat:

- People who want a self-hosted agent with sandbox guardrails
- Teams that want reusable tools across agents
- Users who need automation, file search, and scheduled reporting in one place

---

## Drive Your Agent From the Browser

Manage sessions, tools, schedules, and memory from a browser. The dashboard ships inside the binary — start the daemon and open http://127.0.0.1:17989. It is served by your own machine, so nothing leaves your device.

<p align="center">
  <a href="https://youtu.be/oaXxrTNvLaU">
    <img src="https://img.youtube.com/vi/oaXxrTNvLaU/maxresdefault.jpg" alt="Agenvoy Web Dashboard demo" width="640">
  </a>
</p>

## Chatbot Integrations

Agenvoy currently supports **Telegram and Discord** as chatbot channels. The local daemon initiates outbound connections to these platforms, so you only need to configure a bot token—without exposing inbound ports, setting up a reverse proxy, or making your host public.

Since **v0.34.4**, Telegram and Discord have paused the default flow that automatically replies to voice input with voice output. You can still use STT/TTS tools to generate audio and send the resulting audio files to either channel.

---

## One-line install

<details open>
<summary><strong>macOS / Linux distributions</strong></summary>

Run this in a terminal:

```bash
curl -fsSL https://agenvoy.com/scripts/install.sh | bash
```

> **macOS tip:** If you run schedules on a MacBook, also run:
>
> ```bash
> sudo pmset -c sleep 0
> ```
>
> This prevents sleep from interrupting schedules.

</details>

<details>
<summary><strong>Windows (via WSL)</strong></summary>

First open PowerShell as an administrator, then list and install a Linux distribution:

```powershell
wsl --online --list
wsl --install <distribution-name>
```

After installation, restart your computer, open a WSL terminal, and run:

```bash
curl -fsSL https://agenvoy.com/scripts/install.sh | bash
```

</details>

## Developer Recommendations

A cost-effective model setup to get started:

1. Choose a subscription model for everyday primary use, such as:
   - OpenAI ChatGPT Plus ($20/mo)
   - SuperGrok ($30/mo)
2. **If you need multiple providers**, you can apply for a free **[NVIDIA NIM](https://build.nvidia.com/explore/discover)** API token and set `gpt-oss-20b` as the **dispatcher** model. This enables intelligent routing with fast responses.

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
| Image generation     | Generate images through a configured provider                            |
| Live command output  | Stream `run_command` progress to the TUI and Web dashboard               |
| Secure file boundary | Confirm sensitive paths and out-of-home access before granting them      |
| MCP OAuth            | Log in to HTTP MCP servers and persist tokens in the OS keychain         |
| Transcription        | Audio and video to text                                                  |
| Self-improvement     | Auto-fixes after execution failures                                      |

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
