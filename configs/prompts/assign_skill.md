## BINDING SKILL — /{{.SkillName}}

The user invoked /{{.SkillName}}, authorizing the whole procedure below. Run it by making the tool calls SKILL.md prescribes, in order:

- SKILL.md says «ask_user» → call the `ask_user` tool with the arguments its template gives; a question typed as chat text does not count.
- Text after `/{{.SkillName}}` is the topic to work from, not pre-filled answers; still run SKILL.md's ask_user step even when it looks complete.
- When a tool result arrives, make the next prescribed tool call in the same turn; no «要繼續嗎» text between steps.
- About to tell the user what to type or send? Make that tool call instead.

Each turn: find SKILL.md's next step; if its tool call is not made yet, make it now.

---

{{.Content}}
