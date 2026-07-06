Extract everything needed to answer the user's question from the tool execution history below.

## Rules

- **Extract, don't transcribe**: raw tool output (HTML, page boilerplate, navigation/ad text, unrelated markup, empty/failed calls) is mostly noise — pull out only the facts, figures, quotes, and data points that bear on the user's question and drop the rest
- **Structured payloads (JSON/API responses)**: never keep the raw blob — pull out only the fields/values relevant to the question and drop unused keys, metadata, pagination wrappers, and unrelated array entries
- **Integrate**: merge results from multiple tool calls that return overlapping or related data into one coherent set of facts
- **Deduplicate**: remove exact or near-duplicate information across tool results
- **Preserve verbatim what matters**: exact numbers, quotes, file paths, line numbers, code snippets, error messages, and command outputs relevant to the question must stay exact — never paraphrase, round, or approximate these
- **Structure**: organize by topic or source, not by chronological tool-call order

Completeness on the relevant facts, not brevity: keep every fact/figure/quote that bears on the question, however many there are — do not compress or drop retained facts to save space. This is answer-focused extraction of the source material, not the final reply itself — the next step still composes the actual answer from this.

## User's Question

{{.UserQuestion}}

## Output

Return the extracted material as plain text with headings. No wrapping fences, no meta-commentary.
