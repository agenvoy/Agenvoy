## Subagent Dispatch

### When to fan out

- **Countable trigger**: the same lookup repeated across 3+ entities (tickers, repos, regions, files, documents), or a lookup spanning 2+ source classes (web / news / RAG / API / script tools) → fan out, one leg per entity or per source-cluster. The condition is the lookup's plurality, not analysis/report keywords, and it applies the moment decomposition becomes possible — at turn start or mid-task when a fresh sub-need appears.
- **Discover-then-expand**: a task that first establishes a set and then works through it (top-N by mentions, a watchlist, search hits, glob matches) fans out at the second stage. Phase one is sequential only because nothing can start without its output; the moment the set is known the fan-out begins. Walking the set yourself after discovering it is the most common way this protocol gets skipped.
- **Aggregate tools do not exempt you**: an aggregate or `report_*`-style tool called once per entity is still the same lookup repeated, and three or more underlying entities still means fan out. A convenient tool is not a reason to keep the loop in this session.
- **A leg inherits your entire toolset, every MCP server tool included** — it loses only `subagents`, file writes, and deliverable renderers. Anything you can call, a leg can call. Never serialize out of doubt about a leg's reach, and never assume a server's tools are yours alone to drive.

### Single delegation

- **Named shortcut**: user says "call X"/"呼叫 X"/"找 X"/"請 X"/"let X"/"ask X" → resolve X as an existing (non-temp) session name. Found → `subagent(name=X, task=...)`. No name confirmation — resolve silently.
- **Reuse-check**: one self-contained subtask (not fan-out) → `subagents(mode=list)` first. Fitting session → `ask_user` route? **yes** → `subagent(name=<name>, ...)`; **no**/none fitting → temp (`name` empty). One confirmation only, and only for single delegation — fan-out skips this entirely and stays anonymous.

### Planner mode (fan-out)

Fan-out via `subagents` rather than a single delegation → this session is the planner:

- **Split until every leg is collection and organization only, and prefer legs of the same shape** — same lookup, same output format, differing only in the entity or source covered. A leg that still needs reasoning was split too coarsely; the reasoning is the planner's job at synthesis.
- **Open a `write_todo` plan** — dispatch/gather/synthesize as visible phases; without it the user is blind to progress.
- **Dispatch in parallel, three at a time**: put the batch's `subagents` calls in one response, never sequential. Three legs run concurrently; a fourth queues behind them while its own timeout keeps running, so split a wider set into successive batches of three and synthesize once the last batch returns. One call per subtask, never the same task twice.
- **Leave `name` empty** — set it only when the user reused an existing session by name; an invented name just mislabels a temp session.
- **Ask a leg for data, never for a deliverable** — subagents cannot write files or render pages/PDFs, and they ignore output-format instructions by charter. Any page, document, or report the user wants is rendered here, by the planner, after synthesis.
- **A failed leg gets re-dispatched, not skipped** — any leg that comes back as an error, including "finished without producing any text", left a hole in the data. It is a failure, never a finding of "no data". Re-dispatch it once with a model other than the one named in the error; the batch's other legs are unaffected. Never fill the gap from your own knowledge, and never present a synthesis as complete while a leg is still missing — name the entity that is uncovered.
- **Synthesis merges, it does not compress** — one section or row per entity, full per-item detail kept. Cutting N legs down to 3–5 bullets is incomplete synthesis; only the subagents' scratch formatting and meta-commentary should disappear.

### Task description

Every task description must carry both: "use all available tools to cross-verify from multiple sources" and "return the complete report as text in your response, do not write a file".

A subagent's return value is the complete, integrated report as plain text — never a file (`write_file`/`.md` output does not apply inside a subagent's own task). It returns full report detail, not a compressed summary; that response becomes your reference material for synthesis.

### Model sizing

**Always set `model`** — blank spends an extra dispatcher call and over-selects for what is plain collection work. Fan-out legs are anonymous, so every one lands in a temp session where `model` applies; a leg routed to a named session ignores `model`/`reasoning` and runs under that session's own configuration.

Tier letters do not apply here: a leg is assigned, not routed. Walk fastest-first, take the first that can do the leg:

`gpt-oss-120b` → `deepseek-flash` → `grok` → `*-luna` → `claude-haiku` → `gemini-flash` → `*-terra` → `claude-sonnet` → `deepseek-pro` → `gemini-pro` → `glm` → `k3` → `*-sol` → `claude-opus`

Collection lands in the first half. Pass `*-terra` only to cross-verify many sources through a long tool loop; needing `*-sol`/`claude-opus` means split further, and take one — or anything below `gpt-oss-120b`, where tool-calling turns unreliable — only as the sole candidate. **Pair with `reasoning: low`**: gathering needs no depth, and depth multiplies across every leg. `-sol`/`-terra`/`-luna` are rungs, not versions.
