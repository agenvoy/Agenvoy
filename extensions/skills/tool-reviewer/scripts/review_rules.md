# Tool Design Review Rules

Mirrors the "Tool 註冊" contract in project `CLAUDE.md` under the lazy-schema model. Use as the sole rubric — do not invent extra criteria.

**Lazy-schema context**: every tool's `name` + `description` is always in LLM context; the `parameters` JSON schema is replaced with a stub `{"type":"object","properties":{}}` unless `AlwaysLoad=true`. The LLM decides whether to invoke a tool (or call `search_tools` to load its schema) using `name` + `description` alone. Schema is the call contract, loaded on demand.

---

## Rule 1 — Name is self-explanatory

**Why**: name is the anchor for LLM selection; the description elaborates the trigger. A vague name forces the LLM to compare descriptions across many tools, wasting attention.

**Pass**: name is unambiguous, distinguishable from siblings, conveys both *what* and *which* (e.g. which kind of search, which kind of edit), and uses the same verb / suffix shape as its same-domain siblings.

**Fail patterns** (deterministic checks marked with → rule code):
- Generic verbs (`process`, `handle`, `do`, `manage`, `execute`, `perform`, `dispatch`, `run`) → `R1_GENERIC_VERB`
- Mixed `_` / `-` separators in the same name (Agenvoy convention is snake_case) → `R1_MIXED_SEPARATOR`
- Dynamic Go identifier the parser can't resolve (use a literal or a same-file `const`) → `R1_DYNAMIC_NAME`
- Names that collide on prefix with sibling tools, forcing the LLM to read description to disambiguate (LLM judgment — use scanner's `name_clusters` as the comparison anchor)
- Names that bury the discriminator in the description (`verify` when `cross_review_with_external_agents` is meant)
- Verb redundancy where the second token is implied by the first (`patch_edit` — `patch` already means edit)
- Verb inconsistency within a sibling cluster (one tool uses `analyze_*` while every other cluster member uses `fetch_*`)
- Inconsistent suffix vocabulary across same-domain tools (`read_tool_error` vs `remember_error` vs `search_error_memory` — pick one shape)

**Examples**:
- ✅ `invoke_subagent` / ❌ `dispatch_internal`
- ✅ `cross_review_with_external_agents` / ❌ `verify`
- ✅ `search_conversation_history` / ❌ `search_history` (collides with git / shell history)
- ✅ `fetch_youtube_transcript` / ❌ `analyze_youtube` (verb mismatches sibling `fetch_*` cluster)
- ✅ `apply_patch` / ❌ `patch_edit` (verb redundancy)

When flagging, propose the better name. Cite the relevant sibling cluster from the scan's `name_clusters` output so the suggestion is anchored, not abstract.

---

## Rule 2 — Description writes WHEN to invoke

**Why**: under the lazy-schema model, `description` is the only signal the LLM has before deciding to call (or to call `search_tools` to load the schema first). An action-only description ("Fetches RSS feed") tells the LLM what the tool does after invocation — too late. The description must front-load *when* and *why* to invoke.

**Pass**: description carries enough trigger signal that another LLM, seeing only the name + description, can decide whether this tool matches the user intent. Required content:
- Trigger conditions / user-intent patterns that map to this tool
- Use-case examples (concrete scenarios where this tool wins)
- Contrast with similar tools when the cluster is ambiguous (`prefer over X when Y`)
- Constraints / preconditions the LLM should check before calling

**Fail patterns**:
- **Action-only description** — single sentence describing what the tool executes with no trigger context (`R2_SHORT_DESC` deterministic flag fires when length < 60 chars; LLM heuristic catches longer-but-still-action-only cases)
- Missing contrast when sibling tools exist (e.g. `search_tools` vs `list_tools` — when to use which)
- `**bold**` / markdown emphasis — adds tokens without changing meaning (`R2_BOLD_MARKDOWN`)
- Output schema dumps (`Returns {"name", "path", ...}`) — belongs in `parameters` documentation, not selection text
- Call-contract details (type/unit/enum/default) inside description — those belong in `parameters[*].description` (Rule 3)
- Implementation gossip (`uses readability under the hood`, `auto-skips .gitignore`) unrelated to selection
- Manual-style filler ("This tool will help you ...", "When you need to ...") — write declarative, not conversational

**What is now welcome (previously forbidden)**:
- Numbered / bulleted trigger conditions
- Multiple paragraphs when trigger context genuinely needs them
- Tool-vs-tool comparisons that aid selection (`prefer over invoke_subagent when ...`)

When flagging, propose the expanded version with trigger signals filled in.

---

## Rule 3 — Schema fields are complete call contracts

**Why**: schema loads on demand. When it loads, it must give the LLM everything needed to fill parameters correctly without trial-and-error. Missing examples or unit specifications cause runtime errors that waste tokens and confuse users.

**Pass**: every entry in `parameters.properties` has a `description` covering:
- Type and unit (seconds vs milliseconds, bytes vs MiB, ISO timestamp vs Unix epoch)
- Accepted values (`enum` with per-value meaning, regex pattern, value range)
- At least one concrete example when the type is non-trivial (cron expression, file path with placeholders, JSON body shape)
- Interaction with other parameters (`required when mode=advanced`, `ignored when X is set`)
- Edge cases / failure modes the LLM should anticipate

**Fail patterns** (deterministic checks marked with → rule code):
- `properties` entry missing `description` or empty/whitespace-only → `R3_PARAM_NO_DESC`
- Non-trivial type (`object`, `array`, or has `enum`) with description shorter than 20 chars → `R3_PARAM_SHORT_DESC`
- Description repeats the field name ("user_id: the user id")
- Numeric field without unit specified
- `enum` listed without per-value meaning
- Cron / regex / template syntax without a concrete example
- Parameter description re-pitching the whole tool (that's Rule 2's job)

When flagging, propose the expanded description with the missing dimensions filled in.

---

## Rule 4 — English only

**Why**: mixed-language descriptions create token noise and hurt smaller / multilingual provider models (Gemini, NVIDIA, etc.). Internal user-facing handler return strings may stay in their original language; the *tool definition* (description, parameter descriptions, enums) must be English.

**Pass**: all of `description`, `parameters[*].description`, `parameters[*].enum` text are ASCII / English.

**Fail patterns**:
- Any CJK / Hangul / Hiragana / Katakana codepoint in tool or parameter description → `R4_NON_ENGLISH_DESCRIPTION` / `R4_NON_ENGLISH_PARAM`
- Mixed bilingual descriptions (`Inspect a file 檢查檔案`)
- Full-width punctuation (`，` `。` `「」`) — even if surrounding text is English

When flagging, propose the English rewrite.

---

## Rule 5 — Optional fields require explicit `default`

**Why**: without `default`, the LLM has to guess what omission means. With `default`, the schema *itself* tells the model what happens when the field is dropped.

**Pass**:
- Every parameter NOT in `required[]` declares `"default": <value>` → `R5_OPTIONAL_NO_DEFAULT` flags absence
- Every parameter IN `required[]` does NOT declare `default` → `R5_REQUIRED_HAS_DEFAULT` flags presence

**Fail patterns**:
- Optional field with no `default`
- Required field with `default` (semantically contradictory — pick one)
- `default: null` used as a placeholder when a real default exists in the handler

Handler still owns nil / missing defense — never trust schema default to materialize at the call site.

---

## Severity

| Severity | Rules | Reason |
|---|---|---|
| **High** | R1 (name clarity), R2 (description trigger coverage) | Wrong tool gets selected or never gets selected at all |
| **Medium** | R3 (schema completeness), R4 (English), R5 (optional default) | Tool callable but parameter filling unreliable / token waste |
| **Low** | Cosmetic — bold markdown alone | Stylistic; flag but don't escalate |

Promote Medium → High when multiple Medium violations stack on the same tool.
