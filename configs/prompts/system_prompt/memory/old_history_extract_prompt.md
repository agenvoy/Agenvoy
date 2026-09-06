Extract everything from the conversation history below that's relevant to the user's current question.

## Rules

- **Extract, don't transcribe**: pull out only the facts, decisions, preferences, and context that bear on the current question — small talk, resolved side-tracks, and exchanges superseded by a later correction are noise, drop them
- **Preserve verbatim what matters**: exact numbers, names, decisions, code identifiers, and file paths relevant to the question stay exact — never paraphrase or approximate
- **Keep the latest, not every iteration**: when the same topic was discussed and revised multiple times, keep only the final/most recent conclusion — earlier iterations that got corrected or superseded are noise
- **This is prior-turn reference material, not fresh data**: the conversation history may be outdated or incomplete for the current question — never present it as if it already answers the question or covers what a tool call would verify; if the current question needs up-to-date information, it comes from a live lookup
- **Structure**: organize by topic, not strictly by chronological turn order

Completeness on the relevant facts, not brevity: keep every decision/fact/preference that bears on the question, however many there are — never trade them for space. This is background-context extraction, not the final reply — the next step composes the answer from it, using tools again if the question requires fresh data.

## User's Current Question

{{.UserQuestion}}

## Output

Return the extracted material as plain text with headings. No wrapping fences, no meta-commentary.
