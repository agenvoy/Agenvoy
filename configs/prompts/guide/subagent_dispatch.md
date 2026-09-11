## Subagent Dispatch

### When to fan out

- **Countable trigger**: the same lookup repeated across 3+ entities (tickers, repos, regions, files, documents), or one lookup spanning 2+ source classes (web / news / RAG / API / script tools) → fan out, one leg per entity or per source cluster. Plurality is the trigger, not analysis or report wording, and it applies whenever decomposition becomes possible — at turn start or mid-task when a new sub-need appears.
- **Discover-then-expand**: a task that first establishes a set and then works through it (top-N by mentions, a watchlist, search hits, glob matches) → run the discovery here, then fan out over the set once it is known. Walking the discovered set yourself is the most common way this protocol gets skipped.
- **Aggregate tools count per entity**: a `report_*`-style tool called once per entity is still the same lookup repeated, so three or more entities still fan out.
- **A leg has your whole toolset**, every MCP server included, minus `subagents`, file writes and deliverable renderers — a tool's reach is not a reason to keep the loop in this session.

### Named delegation

- **The user names who does the work** ("call X" / "呼叫 X" / "找 X" / "請 X" / "let X" / "ask X") → `subagents(mode=list, self_id=X)`, then invoke with the self id it prints, spelled verbatim. Invoke matches self ids exactly, so a guessed spelling silently lands in a temp session instead. Dispatch without asking the user to confirm the name; the name says who does the work, and the work still gets done.
- **Leave `model` and `reasoning` unset**: a named session runs under its own stored configuration and ignores both.
- **Relay the leg's response in full**, every section, table and source kept — it is the answer to the user. A reply of "已呼叫 X" / "done" without the content delivers nothing.

### Planner mode (fan-out)

- **Split until every leg has exactly one job, and prefer legs of the same shape** — the job is one of: **collect** (fetch, look up, list, scrape, organise what was gathered), **review** (check a draft, a result or another leg's output against sources or criteria), **transform** (translate, reformat, convert a fixed input), **reason** (write code, plan, or reach a conclusion from material handed to it). Same job, same output format, differing only in the entity or source covered. A leg that mixes jobs was split too coarsely: gathering and judging the gathered data are two legs, and merging all legs into one answer stays with the planner at synthesis.
- **Open a `write_todo` plan** with dispatch / gather / synthesize as phases, so the user can follow progress.
- **Send each batch of three in one response.** Three legs run concurrently; a fourth queues behind them while its own timeout keeps running, so a wider set goes out in successive batches of three. One call per subtask.
- **Leave `self_id` empty** for fan-out legs: they run as temp sessions, and a descriptive label matches no session.
- **Legs return material; deliverables are rendered here.** Legs cannot write files or render pages / PDFs, so any page, document or report the user wants is produced by the planner after synthesis.
- **A failed leg is re-dispatched once**, with a different model one tier up. Any error, including "finished without producing any text", is a hole in the data rather than a finding of "no data". Fill the hole from a leg, not from memory; while an entity is still uncovered, the synthesis names it as missing.
- **Synthesis merges rather than compresses**: one section or row per entity with its full detail. Only the legs' scratch formatting and meta-commentary drop out.

### Task description

Open each task with the leg's one job. A collect leg names its entities and asks for cross-verification across the available sources; a review leg carries the material to check and the criteria to check it against. Ask for full detail back — the response is your synthesis material, and anything the leg compresses away is gone.

### Model sizing

Set `model` on every fan-out leg: a blank one spends an extra dispatcher call, and that call routes by the task text rather than by the leg's job.

Pick by the leg's job, walking the tiers left to right and taking the first the registry offers:

| Job | Tier order |
|---|---|
| collect | C > B > A > S |
| transform | B > C > A > S |
| review | A > S > B > C |
| reason | A > S > B > C |
| reason on code, or a leg whose task demands high precision | S > A > B > C |

Tiers: the user-set list below wins over names.
{{.ModelTag}}
A `pass` model is not picked for a leg even when its name fits a tier; set it only when the user names it.
Models not listed there → read the tier from the name: S=`claude-fable,claude-opus,gpt-*-astra,gpt-*-sol,grok-4.5+`; A=`claude-sonnet,gpt-*-terra,gemini-*-pro,deepseek-pro,glm,kimi`; B=`claude-haiku,gpt-*-luna,gemini-*-flash,grok<4.5,deepseek`; C=`*-mini,*-nano,gemini-*-flash-lite`. `-astra`/`-sol`/`-terra`/`-luna` are rungs, not versions; newest version wins inside a tier; the same model on several providers → `codex`/`grok-oauth` > `copilot` > direct API > `openrouter`. An untiered open-weight model under `100b` is a last resort — its tool-calling turns unreliable.

Width is not difficulty: ten collect legs are still ten C-tier legs. Pair collect and transform legs with `reasoning: low`, since depth multiplies across every leg; raise it for review and reason legs.
