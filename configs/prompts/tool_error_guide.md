# Tool Error Recovery Contract

## Loop: adjust → retry → success

1. Read the returned error message — it determines the adjustment direction, not a guess.
2. Check injected hints first — resolved = apply directly; failed = avoid, try a different approach.
3. No hints injected and this is the 2nd+ retry → `search_error_history(keyword)` before retrying — resolved = apply; failed/abandoned = avoid.
4. Never retry with identical arguments. Every retry must change something based on the error, hint, or history hit.
5. Max 3 attempts total per error before treating it as failed/abandoned.

## script_* / api_* auto-repair

When the failing tool is self-authored (`script_*` or `api_*`), repair it in place instead of working around it — `ext_*` (installed extension) tools are not patchable, fall back to the adjust → retry loop above:

1. Diagnose the error: runtime exception → tag=`script`; parameter/schema mismatch → tag=`json`; API tool definition (url/auth/endpoint) → tag=`api`.
2. `patch_tool(name, tag, old_string, new_string)` to fix it.
3. Retry the same tool call (counts toward the 3-attempt max).
4. Do not fall back to `send_http_request`, `run_command curl ...`, `run_command python3 -c "..."`, or any other shortcut — repair the tool, never bypass it.

## [RETRY_REQUIRED] responses

If a tool result starts with `[RETRY_REQUIRED]`, retry immediately with the fixed arguments it specifies — never output that content as text to the user. Injected hints are binding, not suggestions.

## On success or exhaustion, record

Once the loop resolves (or the 3 attempts are exhausted):
- Non-trivial fix confirmed working, a strategy confirmed non-working, or 3+ approaches exhausted → call `remember_error` with the matching outcome (`resolved` / `failed` / `abandoned`).
- Skip `remember_error` for trivial typos, 1st-retry fixes, or transient errors (network blip, timeout) that don't generalize.
- Batch `remember_error` with other tool calls in the same turn; only call it alone when no other call remains.
