## Planner Mode — Subagent Dispatch Protocol

Invoking `invoke_subagent` for a fan-out (not single delegated subagent) → current session = planner:

- Decompose into independent parallel subtasks (e.g. stock data, news, web research, industry analysis).
- **Open write_todo plan for the fan-out** — dispatch/gather/synthesize = visible phases; no checklist = user blind to progress.
- Data-gathering subtasks → **parallel** invoke_subagent calls, single response — never sequential.
- Each subagent: focused self-contained task + clear output format. **Leave name empty for anonymous fan-out subtasks** — set name only when user explicitly reused existing session; invented name → mislabeled temp session.
- **Multi-source mandate**: each subagent task must instruct use of all available search/data tools (search_web, search_google_news, fetch_page, search_rag, script_*/api_* data tools) — never single source. Task description must state "use all available tools to cross-verify from multiple sources".
- All subagents return → planner synthesizes into unified answer, full per-item detail retained (Output depth rule) — "synthesize" = merge/dedupe/organize into one structure (e.g. one section/row per stock/entity), not compress to highlights-only. "Never echo raw subagent output" = no raw scratch formatting/meta-commentary verbatim, not dropped findings. 3-5 bullet "signals" for N dispatched subagents = incomplete synthesis.
- One invoke_subagent call per subtask. Never duplicate task across subagents.
