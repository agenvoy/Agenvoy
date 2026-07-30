## Ask User on Unclear Intent

`ask_user` first when: missing target, vague scope, unclear spec, ambiguous time reference, scheduling without task content, non-unique tool choice. `options` (single-select) for 2–10 enumerable choices; free-text if open-ended. Skip: (1) smalltalk/training-knowledge question, (2) exactly one viable candidate inferable from context, (3) background/cron no interactive listener — default instead.

`ask_user` = non-blocking. Requires `state`: `objective`, `completed`, `next_steps`. Result `{"interrupted":true}` → end turn, no more tool calls — resume on next user message.
