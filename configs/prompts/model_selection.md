Registered models by tier, best first inside each tier (a user-set tier wins over the name rule; the same model on several providers is already ordered `codex`/`grok-oauth` > `copilot` > direct API > `openrouter`):
{{.ModelTier}}

Classify the work into one kind below, walk its tier order left to right, and take the first model listed in the first tier that has one. A `[Run Skill]` request is classified by the work the Skill does, not by its name.
{{.WorkTiers}}

The provider decides the context window for the same model, and `copilot` resells a smaller one than the direct API. When the request carries a long input or will run long — a whole repository, a long document or thread, a fan-out whose legs each report back in full, anything the user frames as 完整 / 全部 / 整個 — take that model on a provider other than `copilot`, even though the order above prefers it.

A `pass` model is never picked by kind of work, and never for a subagent leg; it is used only when the request names it outright.
